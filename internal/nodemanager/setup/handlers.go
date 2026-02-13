package setup

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agents-admin/internal/nodemanager"
	"agents-admin/internal/shared/sysinstall"

	"gopkg.in/yaml.v3"
)

// SetupConfig 向导提交的配置（HTTP-Only 架构：不再包含 Redis 配置）
type SetupConfig struct {
	APIServer struct {
		URL string `json:"url" yaml:"url"`
	} `json:"api_server" yaml:"api_server"`
	Node struct {
		ID           string            `json:"id" yaml:"id"`
		WorkspaceDir string            `json:"workspace_dir" yaml:"workspace_dir"`
		Labels       map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	} `json:"node" yaml:"node"`
	TLS struct {
		Enabled  bool   `json:"enabled" yaml:"enabled"`
		CAFile   string `json:"ca_file" yaml:"ca_file"`
		CASource string `json:"ca_source,omitempty" yaml:"-"` // download|skip, 不写入 YAML
	} `json:"tls" yaml:"tls"`
}

// InfoResponse /setup/api/info 的响应
type InfoResponse struct {
	Hostname            string   `json:"hostname"`
	IPs                 []string `json:"ips"`
	DefaultWorkspaceDir string   `json:"default_workspace_dir"`
	GeneratedNodeID     string   `json:"generated_node_id"`
}

// ValidateRequest /setup/api/validate 的请求体
type ValidateRequest = SetupConfig

// ValidateResponse /setup/api/validate 的响应
type ValidateResponse struct {
	Valid  bool                   `json:"valid"`
	Checks map[string]CheckResult `json:"checks"`
}

// CheckResult 单项检查结果
type CheckResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ApplyResponse /setup/api/apply 的响应
type ApplyResponse struct {
	Success        bool   `json:"success"`
	ConfigPath     string `json:"config_path"`
	Message        string `json:"message"`
	SystemdInstall bool   `json:"systemd_install,omitempty"`
}

// handleInfo GET /setup/api/info — 返回系统信息（预填表单默认值）
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hostname, _ := os.Hostname()
	ips := getLocalIPs()

	resp := InfoResponse{
		Hostname:            hostname,
		IPs:                 ips,
		DefaultWorkspaceDir: getDefaultWorkspaceDir(),
		GeneratedNodeID:     nodemanager.GenerateNodeID(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleValidate POST /setup/api/validate — 验证配置（连接测试）
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cfg ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON: " + err.Error()})
		return
	}

	// 自动填充工作空间目录
	if cfg.Node.WorkspaceDir == "" {
		cfg.Node.WorkspaceDir = getDefaultWorkspaceDir()
	}

	checks := make(map[string]CheckResult)
	allValid := true

	// 1. 测试 API Server 连接
	apiCheck := testAPIServer(cfg.APIServer.URL, cfg.TLS.CAFile, cfg.TLS.CASource == "skip")
	checks["api_server"] = apiCheck
	if !apiCheck.OK {
		allValid = false
	}

	// 3. 测试工作空间目录
	wsCheck := testWorkspaceDir(cfg.Node.WorkspaceDir)
	checks["workspace"] = wsCheck
	if !wsCheck.OK {
		allValid = false
	}

	resp := ValidateResponse{
		Valid:  allValid,
		Checks: checks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleApply POST /setup/api/apply — 保存配置并退出
func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cfg SetupConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON: " + err.Error()})
		return
	}

	// 自动填充 Node ID（如果未提供）
	if cfg.Node.ID == "" {
		cfg.Node.ID = nodemanager.GenerateNodeID()
		log.Printf("Auto-generated node ID: %s", cfg.Node.ID)
	}

	// 自动填充工作空间目录
	if cfg.Node.WorkspaceDir == "" {
		cfg.Node.WorkspaceDir = getDefaultWorkspaceDir()
	}

	// TLS 处理
	if cfg.TLS.CASource == "download" && cfg.TLS.CAFile == "" && cfg.APIServer.URL != "" {
		// 自动从 API Server 下载 CA 证书（TOFU）
		if caPath, err := downloadAndSaveCA(cfg.APIServer.URL, filepath.Dir(s.configPath)); err != nil {
			log.Printf("WARNING: failed to download CA: %v", err)
		} else {
			cfg.TLS.CAFile = caPath
			cfg.TLS.Enabled = true
			log.Printf("CA certificate downloaded to %s", caPath)
		}
	}
	if cfg.TLS.CASource == "skip" {
		cfg.TLS.Enabled = false
		cfg.TLS.CAFile = ""
	}

	// 如果以 root 运行，执行系统级安装
	systemdInstalled := false
	if sysinstall.IsRoot() {
		if err := sysinstall.EnsureSystemUser(); err != nil {
			log.Printf("WARNING: failed to create system user: %v", err)
		}
		if err := sysinstall.EnsureDirectories(); err != nil {
			log.Printf("WARNING: failed to create directories: %v", err)
		}
	}

	// 写入配置文件
	if err := writeConfig(s.configPath, &cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to write config: " + err.Error()})
		return
	}

	// root 安装：设置文件权限并安装 systemd 服务
	if sysinstall.IsRoot() {
		sysinstall.SetFileOwnership(filepath.Dir(s.configPath))

		if sysinstall.HasSystemd() && !isUnderSystemd() {
			exePath := sysinstall.GetExecutablePath()
			serviceContent := sysinstall.GenerateServiceFile(
				exePath,
				"agents-admin-node-manager",
				"Agents Admin Node Manager",
				"", // Node Manager 没有单独的 .env 文件
				"", // 无额外 After 依赖
			)
			if err := sysinstall.InstallSystemdService("agents-admin-node-manager", serviceContent); err != nil {
				log.Printf("WARNING: failed to install systemd service: %v", err)
			} else {
				systemdInstalled = true
			}
		}
	}

	// 检测是否在 systemd 下运行
	underSystemd := isUnderSystemd()

	message := "Configuration saved to " + s.configPath + "."
	if systemdInstalled {
		message += " Systemd service installed. Use 'sudo systemctl start agents-admin-node-manager' to start."
	} else if underSystemd {
		message += " The service will restart automatically."
	} else {
		message += " Please restart the program manually."
	}

	resp := ApplyResponse{
		Success:        true,
		ConfigPath:     s.configPath,
		Message:        message,
		SystemdInstall: systemdInstalled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Printf("Configuration saved to %s", s.configPath)

	// 延迟退出，让响应先发送
	go func() {
		time.Sleep(500 * time.Millisecond)
		if underSystemd {
			log.Println("Running under systemd, exiting for automatic restart...")
			os.Exit(0)
		} else {
			log.Println("Configuration complete. Please restart the program manually.")
			os.Exit(0)
		}
	}()
}

// writeConfig 写入 YAML 配置文件（合并已有配置，如 API Server 先安装的场景）
func writeConfig(path string, cfg *SetupConfig) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 读取已有配置（如 API Server 先安装）
	existing := make(map[string]interface{})
	if data, err := os.ReadFile(path); err == nil {
		yaml.Unmarshal(data, &existing)
	}

	// 构建 Node Manager 章节
	newSections := map[string]interface{}{
		"api_server": map[string]interface{}{
			"url": cfg.APIServer.URL,
		},
		"node": map[string]interface{}{
			"id":            cfg.Node.ID,
			"workspace_dir": cfg.Node.WorkspaceDir,
		},
		"tls": map[string]interface{}{
			"enabled": cfg.TLS.Enabled,
			"ca_file": cfg.TLS.CAFile,
		},
	}
	if len(cfg.Node.Labels) > 0 {
		newSections["node"].(map[string]interface{})["labels"] = cfg.Node.Labels
	}

	// 一级深度合并：保留已有章节（如 database），合并子字段
	mergeYAMLSections(existing, newSections)

	header := "# Agents Admin Configuration\n# Generated by Setup Wizard\n# " + time.Now().Format(time.RFC3339) + "\n\n"
	data, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, []byte(header+string(data)), 0640); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// mergeYAMLSections 一级深度合并：如果新旧值都是 map，合并子字段；否则新值覆盖旧值
func mergeYAMLSections(existing, newSections map[string]interface{}) {
	for key, newVal := range newSections {
		if existingVal, ok := existing[key]; ok {
			if existingMap, ok := existingVal.(map[string]interface{}); ok {
				if newMap, ok := newVal.(map[string]interface{}); ok {
					for k, v := range newMap {
						existingMap[k] = v
					}
					continue
				}
			}
		}
		existing[key] = newVal
	}
}

// isUnderSystemd 检测是否在 systemd 下运行
func isUnderSystemd() bool {
	// 检查 INVOCATION_ID 环境变量（systemd 设置）
	if os.Getenv("INVOCATION_ID") != "" {
		return true
	}
	// 检查 ppid 是否为 1（systemd 作为 init）
	if os.Getppid() == 1 {
		return true
	}
	return false
}

// getDefaultWorkspaceDir 自动检测默认工作空间目录
func getDefaultWorkspaceDir() string {
	dir := "/var/lib/agents-admin/workspaces"
	if info, err := os.Stat("/var/lib/agents-admin"); err == nil && info.IsDir() {
		testFile := filepath.Join("/var/lib/agents-admin", ".write_test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err == nil {
			os.Remove(testFile)
			return dir
		}
	}
	return "/tmp/agents-workspaces"
}

// ======== 连接测试函数 ========

// testAPIServer 测试 API Server 连接
func testAPIServer(url string, caFile string, skipVerify bool) CheckResult {
	if url == "" {
		return CheckResult{OK: false, Message: "API Server URL is required"}
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// 处理 HTTPS
	if strings.HasPrefix(url, "https://") {
		tlsConfig := &tls.Config{}
		if caFile != "" {
			caCert, err := os.ReadFile(caFile)
			if err != nil {
				return CheckResult{OK: false, Message: fmt.Sprintf("Failed to read CA file: %v", err)}
			}
			caPool := x509.NewCertPool()
			if !caPool.AppendCertsFromPEM(caCert) {
				return CheckResult{OK: false, Message: "Failed to parse CA certificate"}
			}
			tlsConfig.RootCAs = caPool
		} else {
			// HTTPS 首次连接时用 InsecureSkipVerify（TOFU 模式）
			tlsConfig.InsecureSkipVerify = true
		}
		client.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	}

	// 尝试连接 API Server 的健康检查端点
	resp, err := client.Get(url + "/api/v1/health")
	if err != nil {
		// 尝试根路径
		resp, err = client.Get(url)
		if err != nil {
			return CheckResult{OK: false, Message: fmt.Sprintf("Connection failed: %v", err)}
		}
	}
	defer resp.Body.Close()

	return CheckResult{OK: true, Message: fmt.Sprintf("Connected (HTTP %d)", resp.StatusCode)}
}

// testWorkspaceDir 测试工作空间目录
func testWorkspaceDir(dir string) CheckResult {
	if dir == "" {
		return CheckResult{OK: false, Message: "Workspace directory is required"}
	}

	// 尝试创建目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Cannot create directory: %v", err)}
	}

	// 测试写入
	testFile := filepath.Join(dir, ".setup_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Directory not writable: %v", err)}
	}
	os.Remove(testFile)

	return CheckResult{OK: true, Message: "Directory exists and is writable"}
}

// downloadAndSaveCA 从 API Server 下载 CA 证书并保存到 configDir/certs/ca.pem（TOFU 模式）
func downloadAndSaveCA(apiServerURL, configDir string) (string, error) {
	if !strings.HasPrefix(apiServerURL, "https://") {
		return "", fmt.Errorf("HTTPS URL required for CA download")
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	caURL := strings.TrimRight(apiServerURL, "/") + "/ca.pem"
	resp, err := client.Get(caURL)
	if err != nil {
		return "", fmt.Errorf("failed to download CA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API Server returned HTTP %d for /ca.pem", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(data) {
		return "", fmt.Errorf("downloaded file is not a valid PEM certificate")
	}

	certsDir := filepath.Join(configDir, "certs")
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create certs directory: %w", err)
	}

	caPath := filepath.Join(certsDir, "ca.pem")
	if err := os.WriteFile(caPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save CA file: %w", err)
	}

	return caPath, nil
}

// handleDownloadCA POST /setup/api/download-ca — 从 API Server 下载 CA 证书（TOFU 模式）
func (s *Server) handleDownloadCA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		APIServerURL string `json:"api_server_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.APIServerURL == "" || !strings.HasPrefix(req.APIServerURL, "https://") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "HTTPS API Server URL required"})
		return
	}

	// 用 InsecureSkipVerify 下载 /ca.pem（TOFU 信任首次使用）
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	caURL := strings.TrimRight(req.APIServerURL, "/") + "/ca.pem"
	resp, err := client.Get(caURL)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      false,
			"message": fmt.Sprintf("Failed to download CA: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      false,
			"message": fmt.Sprintf("API Server returned HTTP %d for /ca.pem (endpoint may not be available)", resp.StatusCode),
		})
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      false,
			"message": fmt.Sprintf("Failed to read response: %v", err),
		})
		return
	}

	// 验证 PEM 格式
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(data) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      false,
			"message": "Downloaded file is not a valid PEM certificate",
		})
		return
	}

	// 保存到 certs/ 目录
	certsDir := filepath.Join(filepath.Dir(s.configPath), "certs")
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      false,
			"message": fmt.Sprintf("Failed to create certs directory: %v", err),
		})
		return
	}

	caPath := filepath.Join(certsDir, "ca.pem")
	if err := os.WriteFile(caPath, data, 0644); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      false,
			"message": fmt.Sprintf("Failed to save CA file: %v", err),
		})
		return
	}

	// 用下载的 CA 重新验证连接
	verifyClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: caPool}},
	}
	healthURL := strings.TrimRight(req.APIServerURL, "/") + "/api/v1/health"
	verifyResp, verifyErr := verifyClient.Get(healthURL)
	if verifyErr != nil {
		verifyResp, verifyErr = verifyClient.Get(req.APIServerURL)
	}
	if verifyResp != nil {
		verifyResp.Body.Close()
	}

	verifyOK := verifyErr == nil
	msg := "Certificate downloaded and HTTPS connection verified"
	if !verifyOK {
		msg = fmt.Sprintf("Certificate saved but verification failed: %v", verifyErr)
	}

	log.Printf("CA certificate downloaded from API Server and saved to %s (verified: %v)", caPath, verifyOK)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      verifyOK,
		"ca_path": caPath,
		"message": msg,
	})
}

// BootstrapResponse handleBootstrap 的响应
type BootstrapResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	NodeID  string `json:"node_id"`
	CAPath  string `json:"ca_path"`
	TLS     bool   `json:"tls"`
}

// handleBootstrap POST /setup/api/bootstrap — 一键从 API Server 获取所有配置
// 流程：连接 API Server → 下载 CA（如果 HTTPS）→ 获取 bootstrap 配置 → 返回完整配置
func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		APIServerURL string `json:"api_server_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResp(w, http.StatusBadRequest, BootstrapResponse{Message: "Invalid JSON"})
		return
	}

	apiURL := strings.TrimRight(req.APIServerURL, "/")
	if apiURL == "" {
		jsonResp(w, http.StatusBadRequest, BootstrapResponse{Message: "API Server URL is required"})
		return
	}

	isHTTPS := strings.HasPrefix(apiURL, "https://")

	// Step 1: 创建 insecure 客户端用于首次连接
	insecureClient := &http.Client{Timeout: 15 * time.Second}
	if isHTTPS {
		insecureClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// Step 2: 测试 API Server 连通性
	healthResp, err := insecureClient.Get(apiURL + "/api/v1/health")
	if err != nil {
		healthResp, err = insecureClient.Get(apiURL + "/health")
	}
	if err != nil {
		jsonResp(w, http.StatusOK, BootstrapResponse{
			Message: fmt.Sprintf("Cannot connect to API Server: %v", err),
		})
		return
	}
	healthResp.Body.Close()

	// Step 3: 如果 HTTPS，下载 CA 证书（TOFU）
	caPath := ""
	if isHTTPS {
		caResp, err := insecureClient.Get(apiURL + "/ca.pem")
		if err == nil && caResp.StatusCode == http.StatusOK {
			caData, err := io.ReadAll(caResp.Body)
			caResp.Body.Close()
			if err == nil {
				caPool := x509.NewCertPool()
				if caPool.AppendCertsFromPEM(caData) {
					certsDir := filepath.Join(filepath.Dir(s.configPath), "certs")
					os.MkdirAll(certsDir, 0755)
					caPath = filepath.Join(certsDir, "ca.pem")
					os.WriteFile(caPath, caData, 0644)
					log.Printf("CA certificate downloaded and saved to %s", caPath)
				}
			}
		} else if caResp != nil {
			caResp.Body.Close()
		}
	}

	// Step 4: 用正式 CA（如果有）获取 bootstrap 配置
	bootstrapClient := insecureClient
	if caPath != "" {
		caCert, _ := os.ReadFile(caPath)
		caPool := x509.NewCertPool()
		if caPool.AppendCertsFromPEM(caCert) {
			bootstrapClient = &http.Client{
				Timeout:   10 * time.Second,
				Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: caPool}},
			}
		}
	}

	tlsEnabled := isHTTPS

	bsResp, err := bootstrapClient.Get(apiURL + "/api/v1/node-bootstrap")
	if err == nil && bsResp.StatusCode == http.StatusOK {
		var bsData struct {
			TLS struct {
				Enabled bool `json:"enabled"`
			} `json:"tls"`
		}
		if json.NewDecoder(bsResp.Body).Decode(&bsData) == nil {
			tlsEnabled = bsData.TLS.Enabled
		}
		bsResp.Body.Close()
	} else if bsResp != nil {
		bsResp.Body.Close()
	}

	nodeID := nodemanager.GenerateNodeID()

	jsonResp(w, http.StatusOK, BootstrapResponse{
		OK:      true,
		Message: "Bootstrap configuration retrieved successfully",
		NodeID:  nodeID,
		CAPath:  caPath,
		TLS:     tlsEnabled,
	})
}

func jsonResp(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
