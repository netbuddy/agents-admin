// Package executor Instance 工作线程
//
// P2-1 重构：将 Docker 操作从 API Server 下沉到 Executor
// 负责轮询 API Server 获取待处理的 Instance，然后执行实际的 Docker 操作
package nodemanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// InstanceWorker Instance 工作线程
type InstanceWorker struct {
	config        Config
	httpClient    *http.Client
	lastReconcile time.Time
}

// NewInstanceWorker 创建 Instance 工作线程
func NewInstanceWorker(cfg Config) *InstanceWorker {
	return &InstanceWorker{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		lastReconcile: time.Time{},
	}
}

// Start 启动 Instance 工作线程
func (w *InstanceWorker) Start(ctx context.Context) {
	log.Printf("[InstanceWorker] 启动实例工作线程，节点: %s", w.config.NodeID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[InstanceWorker] 工作线程停止")
			return
		case <-ticker.C:
			w.processPendingInstances(ctx)
			// 定期对账：修复“DB 状态/容器真实状态”不一致问题（尤其是历史容器/手工插入数据场景）
			if w.lastReconcile.IsZero() || time.Since(w.lastReconcile) >= 30*time.Second {
				w.lastReconcile = time.Now()
				w.reconcileInstances(ctx)
			}
		}
	}
}

// processPendingInstances 处理待处理的实例
func (w *InstanceWorker) processPendingInstances(ctx context.Context) {
	instances, err := w.fetchPendingInstances(ctx)
	if err != nil {
		log.Printf("[InstanceWorker] 获取待处理实例失败: %v", err)
		return
	}

	for _, inst := range instances {
		w.processInstance(ctx, inst)
	}
}

// instanceInfo 实例信息结构
type instanceInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AccountID     string `json:"account_id"`
	AgentTypeID   string `json:"agent_type_id"`
	ContainerName string `json:"container_name"`
	NodeID        string `json:"node_id"`
	Status        string `json:"status"`
}

// fetchAllInstances 获取所有实例（用于对账）
func (w *InstanceWorker) fetchAllInstances(ctx context.Context) ([]instanceInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", w.config.APIServerURL+"/api/v1/instances", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	var result struct {
		Instances []instanceInfo `json:"instances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return result.Instances, nil
}

// fetchPendingInstances 获取待处理的实例列表
func (w *InstanceWorker) fetchPendingInstances(ctx context.Context) ([]instanceInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		w.config.APIServerURL+"/api/v1/nodes/"+w.config.NodeID+"/instances", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	var result struct {
		Instances []instanceInfo `json:"instances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return result.Instances, nil
}

// processInstance 处理单个实例
func (w *InstanceWorker) processInstance(ctx context.Context, inst instanceInfo) {
	log.Printf("[InstanceWorker] 处理实例: %s (状态: %s)", inst.ID, inst.Status)

	switch inst.Status {
	case "pending":
		w.startOrCreateInstance(ctx, inst)
	case "creating":
		// 检查容器是否已创建完成
		w.checkInstanceCreation(ctx, inst)
	case "stopping":
		w.stopInstance(ctx, inst)
	default:
		log.Printf("[InstanceWorker] 跳过状态 %s 的实例: %s", inst.Status, inst.ID)
	}
}

// startOrCreateInstance 启动或创建实例容器
func (w *InstanceWorker) startOrCreateInstance(ctx context.Context, inst instanceInfo) {
	log.Printf("[InstanceWorker] 处理 pending 实例: %s (container_name=%q)", inst.ID, inst.ContainerName)

	// 优先使用 DB 中的 container_name；若不存在/已被污染，则尝试根据实例信息发现历史容器名
	resolvedName, ok := w.resolveContainerName(ctx, inst)
	if ok {
		inst.ContainerName = resolvedName
		w.startExistingContainer(ctx, inst)
		return
	}

	// 未发现历史容器，则创建新容器（幂等：若同名容器已存在会转为启动）
	w.createInstance(ctx, inst)
}

// isContainerExists 检查容器是否存在
func (w *InstanceWorker) isContainerExists(ctx context.Context, containerName string) bool {
	cmd := exec.CommandContext(ctx, "docker", "inspect", containerName)
	return cmd.Run() == nil
}

// isContainerRunning 检查容器是否运行中
func (w *InstanceWorker) isContainerRunning(ctx context.Context, containerName string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", containerName)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

// listAllContainerNames 列出所有容器名称（包含 stopped）
func (w *InstanceWorker) listAllContainerNames(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	lines := strings.Split(raw, "\n")
	var names []string
	for _, line := range lines {
		n := strings.TrimSpace(line)
		if n != "" {
			names = append(names, n)
		}
	}
	return names, nil
}

// resolveContainerName 解析实例对应的容器名：
// 1) 使用 DB 的 container_name（若存在且容器存在）
// 2) 尝试历史命名规则（agent_inst_<accountID>_<suffix>）
// 3) 尝试新命名规则（agent_<instanceID>）
// 4) 最后按 suffix 在所有容器名中模糊匹配（只有唯一匹配才接受）
func (w *InstanceWorker) resolveContainerName(ctx context.Context, inst instanceInfo) (string, bool) {
	if inst.ContainerName != "" && w.isContainerExists(ctx, inst.ContainerName) {
		return inst.ContainerName, true
	}

	suffix := strings.TrimPrefix(inst.ID, "inst-")
	if suffix != "" {
		candidate1 := fmt.Sprintf("agent_inst_%s_%s", inst.AccountID, suffix)
		if w.isContainerExists(ctx, candidate1) {
			return candidate1, true
		}

		candidate2 := fmt.Sprintf("agent_%s", inst.ID)
		if w.isContainerExists(ctx, candidate2) {
			return candidate2, true
		}

		names, err := w.listAllContainerNames(ctx)
		if err == nil {
			var matches []string
			for _, n := range names {
				if strings.Contains(n, suffix) {
					matches = append(matches, n)
				}
			}
			if len(matches) == 1 {
				return matches[0], true
			}
		}
	}

	return "", false
}

// startExistingContainer 启动已存在的容器
func (w *InstanceWorker) startExistingContainer(ctx context.Context, inst instanceInfo) {
	log.Printf("[InstanceWorker] 启动已存在的容器: %s", inst.ContainerName)

	if inst.ContainerName == "" {
		log.Printf("[InstanceWorker] 实例 %s 缺少 container_name，无法启动", inst.ID)
		_ = w.updateInstanceStatus(ctx, inst.ID, "error", nil)
		return
	}

	// 幂等：已运行则直接回填 running（并修正 container_name）
	if running, err := w.isContainerRunning(ctx, inst.ContainerName); err == nil && running {
		if err := w.updateInstanceStatus(ctx, inst.ID, "running", &inst.ContainerName); err != nil {
			log.Printf("[InstanceWorker] 更新状态失败: %v", err)
		}
		return
	}

	cmd := exec.CommandContext(ctx, "docker", "start", inst.ContainerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[InstanceWorker] 启动容器失败: %v, 输出: %s", err, string(output))
		_ = w.updateInstanceStatus(ctx, inst.ID, "error", nil)
		return
	}

	// 更新状态为 running
	if err := w.updateInstanceStatus(ctx, inst.ID, "running", &inst.ContainerName); err != nil {
		log.Printf("[InstanceWorker] 更新状态失败: %v", err)
		return
	}

	log.Printf("[InstanceWorker] 容器 %s 启动成功", inst.ContainerName)
}

// stopInstance 停止实例容器
func (w *InstanceWorker) stopInstance(ctx context.Context, inst instanceInfo) {
	resolvedName, ok := w.resolveContainerName(ctx, inst)
	if ok {
		inst.ContainerName = resolvedName
	}
	if inst.ContainerName == "" || !w.isContainerExists(ctx, inst.ContainerName) {
		log.Printf("[InstanceWorker] 实例 %s 无可用容器，标记为 stopped", inst.ID)
		_ = w.updateInstanceStatus(ctx, inst.ID, "stopped", nil)
		return
	}

	// 幂等：容器不在运行中，直接标记 stopped
	if running, err := w.isContainerRunning(ctx, inst.ContainerName); err == nil && !running {
		_ = w.updateInstanceStatus(ctx, inst.ID, "stopped", &inst.ContainerName)
		return
	}

	log.Printf("[InstanceWorker] 停止容器: %s", inst.ContainerName)

	cmd := exec.CommandContext(ctx, "docker", "stop", "-t", "10", inst.ContainerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[InstanceWorker] 停止容器失败: %v, 输出: %s", err, string(output))
		// 即使失败也尝试标记为 stopped（避免界面卡死）
	}

	// 更新状态为 stopped
	if err := w.updateInstanceStatus(ctx, inst.ID, "stopped", &inst.ContainerName); err != nil {
		log.Printf("[InstanceWorker] 更新状态失败: %v", err)
		return
	}

	log.Printf("[InstanceWorker] 容器 %s 已停止", inst.ContainerName)
}

// createInstance 创建实例容器
func (w *InstanceWorker) createInstance(ctx context.Context, inst instanceInfo) {
	log.Printf("[InstanceWorker] 创建实例容器: %s", inst.ID)

	// 生成容器名称（与旧数据隔离：使用 instanceID 作为唯一键）
	containerName := fmt.Sprintf("agent_%s", inst.ID)

	// 幂等：同名容器已存在则转为启动/回填
	if w.isContainerExists(ctx, containerName) {
		log.Printf("[InstanceWorker] 发现同名容器已存在，转为启动: %s", containerName)
		inst.ContainerName = containerName
		w.startExistingContainer(ctx, inst)
		return
	}

	// 更新状态为 creating
	if err := w.updateInstanceStatus(ctx, inst.ID, "creating", nil); err != nil {
		log.Printf("[InstanceWorker] 更新状态失败: %v", err)
		return
	}

	// 获取账号信息（包含 Volume 名称）
	account, err := w.getAccount(ctx, inst.AccountID)
	if err != nil {
		log.Printf("[InstanceWorker] 获取账号失败: %v", err)
		_ = w.updateInstanceStatus(ctx, inst.ID, "error", nil)
		return
	}

	if account.VolumeName == "" {
		log.Printf("[InstanceWorker] 账号没有 Volume: %s", inst.AccountID)
		_ = w.updateInstanceStatus(ctx, inst.ID, "error", nil)
		return
	}

	// 获取 Agent 类型配置
	agentType, err := w.getAgentType(ctx, inst.AgentTypeID)
	if err != nil {
		log.Printf("[InstanceWorker] 获取 Agent 类型失败: %v", err)
		_ = w.updateInstanceStatus(ctx, inst.ID, "error", nil)
		return
	}

	// 创建 Docker 容器
	// docker run -d --name <container> -v <volume>:<auth_dir> -t -i <image>
	runArgs := []string{
		"run", "-d",
		"--name", containerName,
		// 标记为系统管理容器（用于孤儿清理/审计）
		"--label", "agents-admin.managed=true",
		"--label", fmt.Sprintf("agents-admin.instance_id=%s", inst.ID),
		"--label", fmt.Sprintf("agents-admin.account_id=%s", inst.AccountID),
		"--label", fmt.Sprintf("agents-admin.node_id=%s", inst.NodeID),
		"-v", fmt.Sprintf("%s:%s", account.VolumeName, agentType.AuthDir),
		"--restart", "unless-stopped",
		"-t",
		"-i",
		agentType.Image,
	}

	log.Printf("[InstanceWorker] 执行: docker %v", runArgs)

	cmd := exec.CommandContext(ctx, "docker", runArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[InstanceWorker] 创建容器失败: %v, 输出: %s", err, string(output))
		_ = w.updateInstanceStatus(ctx, inst.ID, "error", nil)
		return
	}

	// 更新状态为 running，回填容器名称
	if err := w.updateInstanceStatus(ctx, inst.ID, "running", &containerName); err != nil {
		log.Printf("[InstanceWorker] 更新状态失败: %v", err)
		return
	}

	log.Printf("[InstanceWorker] 实例 %s 创建成功，容器: %s", inst.ID, containerName)
}

// checkInstanceCreation 检查容器创建状态
func (w *InstanceWorker) checkInstanceCreation(ctx context.Context, inst instanceInfo) {
	if inst.ContainerName == "" {
		return
	}

	// 检查容器是否运行中
	running, err := w.isContainerRunning(ctx, inst.ContainerName)
	if err != nil {
		log.Printf("[InstanceWorker] 检查容器状态失败: %v", err)
		return
	}

	if running {
		// 容器正在运行，更新状态
		_ = w.updateInstanceStatus(ctx, inst.ID, "running", &inst.ContainerName)
	}
}

// reconcileInstances 对账：将“稳定状态实例”的 DB 状态与容器真实状态对齐
func (w *InstanceWorker) reconcileInstances(ctx context.Context) {
	instances, err := w.fetchAllInstances(ctx)
	if err != nil {
		log.Printf("[InstanceWorker] 对账失败（拉取实例列表）：%v", err)
		return
	}

	// 额外：清理“DB 已删除但容器仍残留”的孤儿容器，避免资源泄漏
	// 说明：API DeleteInstance 当前会直接删除 DB 记录，因此需要数据面做 GC。
	w.cleanupOrphanInstanceContainers(ctx, instances)

	for _, inst := range instances {
		// 只对本节点实例对账
		if inst.NodeID != w.config.NodeID {
			continue
		}

		// 跳过“声明式过渡态”（由用户操作触发，等待 Executor 处理）
		switch inst.Status {
		case "pending", "creating", "stopping":
			continue
		}

		resolvedName, ok := w.resolveContainerName(ctx, inst)
		if !ok {
			// 容器不存在：running 视为异常，其他保持原样
			if inst.Status == "running" {
				log.Printf("[InstanceWorker] 对账：实例 %s 标记 running 但找不到容器，置为 error", inst.ID)
				_ = w.updateInstanceStatus(ctx, inst.ID, "error", nil)
			}
			continue
		}

		running, err := w.isContainerRunning(ctx, resolvedName)
		if err != nil {
			continue
		}

		if running {
			if inst.Status != "running" || inst.ContainerName != resolvedName {
				log.Printf("[InstanceWorker] 对账：实例 %s 容器在运行，修正 DB 状态为 running (container=%s)", inst.ID, resolvedName)
				_ = w.updateInstanceStatus(ctx, inst.ID, "running", &resolvedName)
			}
			continue
		}

		// 容器不在运行
		if inst.Status == "running" {
			log.Printf("[InstanceWorker] 对账：实例 %s DB=running 但容器未运行，修正为 stopped (container=%s)", inst.ID, resolvedName)
			_ = w.updateInstanceStatus(ctx, inst.ID, "stopped", &resolvedName)
		}
	}
}

// isManagedInstanceContainerName 判断是否为本系统管理的“实例容器”命名
// 目前兼容两类命名：
// - 新命名：agent_<instanceID> -> 形如 agent_inst-<suffix>
// - 旧命名：agent_inst_<...>
func isManagedInstanceContainerName(name string) bool {
	return strings.HasPrefix(name, "agent_inst-") || strings.HasPrefix(name, "agent_inst_")
}

type containerMeta struct {
	image   string
	running bool
	status  string
	managed bool // 是否带 agents-admin.managed=true 标签
}

func (w *InstanceWorker) inspectContainerMeta(ctx context.Context, containerName string) (*containerMeta, error) {
	// 输出格式：image|running|status|managedLabel
	// managedLabel 可能为空
	cmd := exec.CommandContext(
		ctx,
		"docker", "inspect",
		"-f", `{{.Config.Image}}|{{.State.Running}}|{{.State.Status}}|{{index .Config.Labels "agents-admin.managed"}}`,
		containerName,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("unexpected inspect output: %q", strings.TrimSpace(string(out)))
	}
	return &containerMeta{
		image:   parts[0],
		running: parts[1] == "true",
		status:  parts[2],
		managed: parts[3] == "true",
	}, nil
}

func (w *InstanceWorker) cleanupOrphanInstanceContainers(ctx context.Context, instances []instanceInfo) {
	// keep：DB 中仍存在的实例容器名（以及新命名的“预期容器名”）
	keep := make(map[string]struct{}, len(instances)*2)
	for _, inst := range instances {
		if inst.ContainerName != "" {
			keep[inst.ContainerName] = struct{}{}
		}
		// 新命名规则容器名：agent_<instanceID>
		keep[fmt.Sprintf("agent_%s", inst.ID)] = struct{}{}
	}

	names, err := w.listAllContainerNames(ctx)
	if err != nil {
		log.Printf("[InstanceWorker] 孤儿容器清理失败（列出容器）：%v", err)
		return
	}

	for _, name := range names {
		if !isManagedInstanceContainerName(name) {
			continue
		}
		if _, ok := keep[name]; ok {
			continue
		}

		meta, err := w.inspectContainerMeta(ctx, name)
		if err != nil {
			continue
		}

		// 只清理 runner 容器，避免误删用户自定义的同名前缀容器
		if !strings.HasPrefix(meta.image, "runners/") {
			continue
		}

		// 安全策略：
		// - 对 legacy 命名（agent_inst_...）且无 managed 标签的运行容器：默认不清理，避免误删
		// - 对新命名（agent_inst-... == agent_<instanceID>）的容器：即便无标签，只要已判定为 orphan，就可清理
		// - 有 managed 标签的容器：可清理运行/停止态 orphan（删除实例即期望释放资源）
		if meta.running && !meta.managed && strings.HasPrefix(name, "agent_inst_") {
			log.Printf("[InstanceWorker] 发现孤儿运行容器但无管理标签（legacy 命名），跳过清理: %s (image=%s)", name, meta.image)
			continue
		}

		log.Printf("[InstanceWorker] 清理孤儿容器: %s (image=%s, status=%s, running=%v, managed=%v)", name, meta.image, meta.status, meta.running, meta.managed)
		rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", name)
		out, err := rmCmd.CombinedOutput()
		if err != nil {
			log.Printf("[InstanceWorker] 删除孤儿容器失败: %s: %v, 输出: %s", name, err, string(out))
			continue
		}
	}
}

// accountInfo 账号信息结构
type accountInfo struct {
	ID          string `json:"id"`
	VolumeName  string `json:"volume_name"`
	AgentTypeID string `json:"agent_type"`
	Status      string `json:"status"`
}

// getAccount 获取账号信息
func (w *InstanceWorker) getAccount(ctx context.Context, accountID string) (*accountInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		w.config.APIServerURL+"/api/v1/accounts/"+accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	var account accountInfo
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &account, nil
}

// agentTypeInfo Agent 类型信息结构
type agentTypeInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	AuthDir string `json:"auth_dir"`
}

// getAgentType 获取 Agent 类型信息
func (w *InstanceWorker) getAgentType(ctx context.Context, agentTypeID string) (*agentTypeInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		w.config.APIServerURL+"/api/v1/agent-types/"+agentTypeID, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	var agentType agentTypeInfo
	if err := json.NewDecoder(resp.Body).Decode(&agentType); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &agentType, nil
}

// updateInstanceStatus 更新实例状态
func (w *InstanceWorker) updateInstanceStatus(ctx context.Context, instanceID, status string, containerName *string) error {
	payload := map[string]interface{}{
		"status": status,
	}
	if containerName != nil {
		payload["container_name"] = *containerName
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "PATCH",
		w.config.APIServerURL+"/api/v1/instances/"+instanceID,
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 返回错误状态: %d", resp.StatusCode)
	}

	return nil
}

// strPtr 已在 auth_controller.go 中定义
