// Package executor Workspace 管理器
//
// 负责任务执行前的 Workspace 准备工作：
//   - Git 类型：克隆仓库到指定目录
//   - Local 类型：验证目录存在
//   - Volume 类型：准备 Docker Volume
package nodemanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WorkspaceManager Workspace 管理器
type WorkspaceManager struct {
	baseDir string // 工作空间基础目录
}

// NewWorkspaceManager 创建 Workspace 管理器
func NewWorkspaceManager(baseDir string) *WorkspaceManager {
	// 确保基础目录存在
	if baseDir == "" {
		baseDir = "/tmp/agent-workspaces"
	}
	os.MkdirAll(baseDir, 0755)

	return &WorkspaceManager{
		baseDir: baseDir,
	}
}

// WorkspaceConfig Workspace 配置（从 TaskSpec 解析）
type WorkspaceConfig struct {
	Type   string     `json:"type"`   // git, local, volume
	Git    *GitConfig `json:"git"`    // Git 配置
	Local  *LocalCfg  `json:"local"`  // Local 配置
	Volume *VolumeCfg `json:"volume"` // Volume 配置
}

// GitConfig Git 仓库配置
type GitConfig struct {
	URL    string `json:"url"`    // 仓库地址
	Branch string `json:"branch"` // 分支
	Commit string `json:"commit"` // 指定 commit
	Depth  int    `json:"depth"`  // 克隆深度
}

// LocalCfg 本地目录配置
type LocalCfg struct {
	Path     string `json:"path"`
	ReadOnly bool   `json:"read_only"`
}

// VolumeCfg Volume 配置
type VolumeCfg struct {
	Name    string `json:"name"`
	SubPath string `json:"sub_path"`
}

// PreparedWorkspace 准备好的工作空间
type PreparedWorkspace struct {
	Path       string   // 工作空间路径
	ReadOnly   bool     // 是否只读
	MountArgs  []string // Docker 挂载参数
	Cleanup    func()   // 清理函数
	WorkingDir string   // 容器内工作目录
}

// Prepare 准备工作空间
//
// 根据配置类型执行不同的准备逻辑：
//   - git: 克隆仓库
//   - local: 验证目录
//   - volume: 准备 Volume
//
// 返回准备好的工作空间信息，包含 Docker 挂载参数
func (m *WorkspaceManager) Prepare(ctx context.Context, runID string, config *WorkspaceConfig) (*PreparedWorkspace, error) {
	if config == nil {
		return nil, nil // 无 Workspace 配置
	}

	switch config.Type {
	case "git":
		return m.prepareGit(ctx, runID, config.Git)
	case "local":
		return m.prepareLocal(ctx, runID, config.Local)
	case "volume":
		return m.prepareVolume(ctx, runID, config.Volume)
	default:
		return nil, fmt.Errorf("不支持的 Workspace 类型: %s", config.Type)
	}
}

// prepareGit 准备 Git 工作空间
func (m *WorkspaceManager) prepareGit(ctx context.Context, runID string, config *GitConfig) (*PreparedWorkspace, error) {
	if config == nil || config.URL == "" {
		return nil, fmt.Errorf("Git URL 不能为空")
	}

	// 创建工作目录
	workDir := filepath.Join(m.baseDir, runID)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("创建工作目录失败: %w", err)
	}

	log.Printf("[Workspace] 克隆 Git 仓库: %s -> %s", config.URL, workDir)

	// 构建 git clone 命令
	args := []string{"clone"}

	// 深度克隆
	if config.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", config.Depth))
	}

	// 指定分支
	if config.Branch != "" {
		args = append(args, "-b", config.Branch)
	}

	args = append(args, config.URL, workDir)

	// 执行克隆
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(workDir)
		return nil, fmt.Errorf("git clone 失败: %w, 输出: %s", err, string(output))
	}

	// 如果指定了 commit，切换到该 commit
	if config.Commit != "" {
		cmd = exec.CommandContext(ctx, "git", "checkout", config.Commit)
		cmd.Dir = workDir
		if output, err := cmd.CombinedOutput(); err != nil {
			os.RemoveAll(workDir)
			return nil, fmt.Errorf("git checkout 失败: %w, 输出: %s", err, string(output))
		}
	}

	log.Printf("[Workspace] Git 仓库准备完成: %s", workDir)

	// 容器内工作目录
	containerWorkDir := "/workspace"

	return &PreparedWorkspace{
		Path:       workDir,
		ReadOnly:   false,
		MountArgs:  []string{"-v", fmt.Sprintf("%s:%s", workDir, containerWorkDir)},
		WorkingDir: containerWorkDir,
		Cleanup: func() {
			log.Printf("[Workspace] 清理工作目录: %s", workDir)
			os.RemoveAll(workDir)
		},
	}, nil
}

// prepareLocal 准备 Local 工作空间
func (m *WorkspaceManager) prepareLocal(ctx context.Context, runID string, config *LocalCfg) (*PreparedWorkspace, error) {
	if config == nil || config.Path == "" {
		return nil, fmt.Errorf("Local 路径不能为空")
	}

	// 验证目录存在
	info, err := os.Stat(config.Path)
	if err != nil {
		return nil, fmt.Errorf("目录不存在: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", config.Path)
	}

	log.Printf("[Workspace] 使用本地目录: %s (只读: %v)", config.Path, config.ReadOnly)

	// 容器内工作目录
	containerWorkDir := "/workspace"

	// 构建挂载参数
	mountArg := fmt.Sprintf("%s:%s", config.Path, containerWorkDir)
	if config.ReadOnly {
		mountArg += ":ro"
	}

	return &PreparedWorkspace{
		Path:       config.Path,
		ReadOnly:   config.ReadOnly,
		MountArgs:  []string{"-v", mountArg},
		WorkingDir: containerWorkDir,
		Cleanup:    nil, // Local 类型不需要清理
	}, nil
}

// prepareVolume 准备 Volume 工作空间
func (m *WorkspaceManager) prepareVolume(ctx context.Context, runID string, config *VolumeCfg) (*PreparedWorkspace, error) {
	if config == nil || config.Name == "" {
		return nil, fmt.Errorf("Volume 名称不能为空")
	}

	log.Printf("[Workspace] 使用 Docker Volume: %s", config.Name)

	// 检查 Volume 是否存在
	cmd := exec.CommandContext(ctx, "docker", "volume", "inspect", config.Name)
	if err := cmd.Run(); err != nil {
		// Volume 不存在，创建它
		log.Printf("[Workspace] 创建 Docker Volume: %s", config.Name)
		cmd = exec.CommandContext(ctx, "docker", "volume", "create", config.Name)
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("创建 Volume 失败: %w, 输出: %s", err, string(output))
		}
	}

	// 容器内工作目录
	containerWorkDir := "/workspace"
	if config.SubPath != "" {
		containerWorkDir = filepath.Join(containerWorkDir, config.SubPath)
	}

	return &PreparedWorkspace{
		Path:       config.Name,
		ReadOnly:   false,
		MountArgs:  []string{"-v", fmt.Sprintf("%s:/workspace", config.Name)},
		WorkingDir: containerWorkDir,
		Cleanup:    nil, // Volume 是持久化的，不需要清理
	}, nil
}

// CleanupOldWorkspaces 清理过期的工作空间
func (m *WorkspaceManager) CleanupOldWorkspaces(ctx context.Context, maxAge time.Duration) error {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			path := filepath.Join(m.baseDir, entry.Name())
			log.Printf("[Workspace] 清理过期工作空间: %s (年龄: %v)", path, now.Sub(info.ModTime()))
			os.RemoveAll(path)
		}
	}

	return nil
}

// ParseWorkspaceConfig 从任务快照中解析 Workspace 配置
func ParseWorkspaceConfig(snapshot map[string]interface{}) *WorkspaceConfig {
	wsRaw, ok := snapshot["workspace"]
	if !ok || wsRaw == nil {
		return nil
	}

	ws, ok := wsRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	wsType, _ := ws["type"].(string)
	if wsType == "" {
		return nil
	}

	config := &WorkspaceConfig{Type: wsType}

	switch wsType {
	case "git":
		if gitRaw, ok := ws["git"].(map[string]interface{}); ok {
			config.Git = &GitConfig{
				URL:    getStringField(gitRaw, "url"),
				Branch: getStringField(gitRaw, "branch"),
				Commit: getStringField(gitRaw, "commit"),
				Depth:  getIntField(gitRaw, "depth"),
			}
		}
	case "local":
		if localRaw, ok := ws["local"].(map[string]interface{}); ok {
			config.Local = &LocalCfg{
				Path:     getStringField(localRaw, "path"),
				ReadOnly: getBoolField(localRaw, "read_only"),
			}
		}
	case "volume":
		if volumeRaw, ok := ws["volume"].(map[string]interface{}); ok {
			config.Volume = &VolumeCfg{
				Name:    getStringField(volumeRaw, "name"),
				SubPath: getStringField(volumeRaw, "sub_path"),
			}
		}
	}

	return config
}

// 辅助函数
func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getIntField(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func getBoolField(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// SanitizeWorkspacePath 清理工作空间路径中的不安全字符
func SanitizeWorkspacePath(path string) string {
	// 移除路径遍历字符
	path = strings.ReplaceAll(path, "..", "")
	path = strings.ReplaceAll(path, "//", "/")
	return filepath.Clean(path)
}
