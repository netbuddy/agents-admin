package node

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"agents-admin/internal/shared/model"
)

// ProvisionStore 部署记录存储接口
type ProvisionStore interface {
	CreateNodeProvision(ctx context.Context, p *model.NodeProvision) error
	UpdateNodeProvision(ctx context.Context, p *model.NodeProvision) error
	GetNodeProvision(ctx context.Context, id string) (*model.NodeProvision, error)
	ListNodeProvisions(ctx context.Context) ([]*model.NodeProvision, error)
}

// ProvisionRequest 部署请求（含敏感信息，不持久化）
type ProvisionRequest struct {
	NodeID       string `json:"node_id"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	SSHUser      string `json:"ssh_user"`
	AuthMethod   string `json:"auth_method"`
	Password     string `json:"password,omitempty"`
	PrivateKey   string `json:"private_key,omitempty"`
	Version      string `json:"version"`
	GithubRepo   string `json:"github_repo"`
	APIServerURL string `json:"api_server_url"`
}

// Provisioner 节点远程部署器
type Provisioner struct {
	store     ProvisionStore
	nodeStore NodePersistentStore
}

// NewProvisioner 创建部署器
func NewProvisioner(store ProvisionStore, nodeStore NodePersistentStore) *Provisioner {
	return &Provisioner{store: store, nodeStore: nodeStore}
}

// StartProvision 异步启动部署流程
func (p *Provisioner) StartProvision(ctx context.Context, req ProvisionRequest) (*model.NodeProvision, error) {
	if req.Port == 0 {
		req.Port = 22
	}
	if req.GithubRepo == "" {
		req.GithubRepo = "org/agents-admin"
	}

	provision := &model.NodeProvision{
		ID:           fmt.Sprintf("prov-%s", generateShortID()),
		NodeID:       req.NodeID,
		Host:         req.Host,
		Port:         req.Port,
		SSHUser:      req.SSHUser,
		AuthMethod:   req.AuthMethod,
		Status:       model.NodeProvisionStatusPending,
		Version:      req.Version,
		GithubRepo:   req.GithubRepo,
		APIServerURL: req.APIServerURL,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := p.store.CreateNodeProvision(ctx, provision); err != nil {
		return nil, fmt.Errorf("failed to create provision record: %w", err)
	}

	// 异步执行部署
	go p.execute(provision, req)

	return provision, nil
}

// execute 执行部署流程
func (p *Provisioner) execute(prov *model.NodeProvision, req ProvisionRequest) {
	ctx := context.Background()

	updateStatus := func(status model.NodeProvisionStatus, errMsg string) {
		prov.Status = status
		prov.ErrorMessage = errMsg
		prov.UpdatedAt = time.Now()
		if err := p.store.UpdateNodeProvision(ctx, prov); err != nil {
			log.Printf("[provision] failed to update status: %v", err)
		}
	}

	// 1. SSH 连接
	updateStatus(model.NodeProvisionStatusConnecting, "")
	log.Printf("[provision] %s: connecting to %s@%s:%d", prov.ID, req.SSHUser, req.Host, req.Port)

	client, err := p.sshConnect(req)
	if err != nil {
		updateStatus(model.NodeProvisionStatusFailed, fmt.Sprintf("SSH connect failed: %v", err))
		return
	}
	defer client.Close()

	// 2. 检测架构
	arch, err := p.remoteExec(client, "dpkg --print-architecture 2>/dev/null || (uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')")
	if err != nil {
		updateStatus(model.NodeProvisionStatusFailed, fmt.Sprintf("detect arch failed: %v", err))
		return
	}
	arch = strings.TrimSpace(arch)
	log.Printf("[provision] %s: detected arch=%s", prov.ID, arch)

	// 3. 下载 deb 包
	updateStatus(model.NodeProvisionStatusDownloading, "")
	debFile := fmt.Sprintf("agents-admin-node-manager_%s_%s.deb", prov.Version, arch)
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s",
		prov.GithubRepo, prov.Version, debFile)

	cmd := fmt.Sprintf("wget -q -O /tmp/%s '%s' || curl -sL -o /tmp/%s '%s'",
		debFile, downloadURL, debFile, downloadURL)
	if _, err := p.remoteExec(client, cmd); err != nil {
		updateStatus(model.NodeProvisionStatusFailed, fmt.Sprintf("download failed: %v", err))
		return
	}
	log.Printf("[provision] %s: downloaded %s", prov.ID, debFile)

	// 4. 安装 deb
	updateStatus(model.NodeProvisionStatusInstalling, "")
	installCmd := fmt.Sprintf("DEBIAN_FRONTEND=noninteractive dpkg -i /tmp/%s || apt-get install -f -y", debFile)
	if req.SSHUser != "root" {
		installCmd = "sudo " + installCmd
	}
	if _, err := p.remoteExec(client, installCmd); err != nil {
		updateStatus(model.NodeProvisionStatusFailed, fmt.Sprintf("install failed: %v", err))
		return
	}
	log.Printf("[provision] %s: installed deb", prov.ID)

	// 5. 写入配置文件
	updateStatus(model.NodeProvisionStatusConfiguring, "")
	configContent := fmt.Sprintf(`node:
  id: %s
  api_server_url: %s
  workspace_dir: /var/lib/agents-admin/workspaces
  labels:
    os: linux

redis:
  url: ""

etcd:
  endpoints: ""
  prefix: /agents
`, prov.NodeID, prov.APIServerURL)

	// 判断是否需要 TLS
	if strings.HasPrefix(prov.APIServerURL, "https://") {
		configContent += `
tls:
  enabled: true
  ca_file: /etc/agents-admin/certs/ca.pem
`
	}

	writeCmd := fmt.Sprintf("cat > /etc/agents-admin/nodemanager.yaml << 'CFGEOF'\n%sCFGEOF", configContent)
	if req.SSHUser != "root" {
		writeCmd = fmt.Sprintf("sudo bash -c \"cat > /etc/agents-admin/nodemanager.yaml << 'CFGEOF'\n%sCFGEOF\"", configContent)
	}
	if _, err := p.remoteExec(client, writeCmd); err != nil {
		updateStatus(model.NodeProvisionStatusFailed, fmt.Sprintf("config write failed: %v", err))
		return
	}

	// 6. 创建工作目录并启动服务
	mkdirCmd := "mkdir -p /var/lib/agents-admin/workspaces"
	restartCmd := "systemctl restart agents-admin-node-manager"
	if req.SSHUser != "root" {
		mkdirCmd = "sudo " + mkdirCmd
		restartCmd = "sudo " + restartCmd
	}
	p.remoteExec(client, mkdirCmd)
	if _, err := p.remoteExec(client, restartCmd); err != nil {
		updateStatus(model.NodeProvisionStatusFailed, fmt.Sprintf("service start failed: %v", err))
		return
	}
	log.Printf("[provision] %s: service started", prov.ID)

	// 7. 等待心跳验证
	if p.waitForHeartbeat(ctx, prov.NodeID, 30*time.Second) {
		updateStatus(model.NodeProvisionStatusCompleted, "")
		log.Printf("[provision] %s: node %s is online!", prov.ID, prov.NodeID)
	} else {
		updateStatus(model.NodeProvisionStatusCompleted, "service started but heartbeat not yet received")
		log.Printf("[provision] %s: service started, heartbeat pending", prov.ID)
	}

	// 清理下载文件
	cleanCmd := fmt.Sprintf("rm -f /tmp/%s", debFile)
	if req.SSHUser != "root" {
		cleanCmd = "sudo " + cleanCmd
	}
	p.remoteExec(client, cleanCmd)
}

// sshConnect 建立 SSH 连接
func (p *Provisioner) sshConnect(req ProvisionRequest) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	switch req.AuthMethod {
	case "password":
		authMethods = append(authMethods, ssh.Password(req.Password))
	case "pubkey":
		signer, err := ssh.ParsePrivateKey([]byte(req.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", req.AuthMethod)
	}

	config := &ssh.ClientConfig{
		User:            req.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", req.Host, req.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return client, nil
}

// remoteExec 在远程主机上执行命令
func (p *Provisioner) remoteExec(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("exec %q: %w (output: %s)", cmd, err, string(output))
	}
	return string(output), nil
}

// waitForHeartbeat 等待节点心跳
func (p *Provisioner) waitForHeartbeat(ctx context.Context, nodeID string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return false
		case <-ticker.C:
			node, err := p.nodeStore.GetNode(ctx, nodeID)
			if err != nil || node == nil {
				continue
			}
			if node.LastHeartbeat != nil && time.Since(*node.LastHeartbeat) < 15*time.Second {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
}

// generateShortID 生成短 ID（复用已有的 ID 生成逻辑）
func generateShortID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%1000000000)
}
