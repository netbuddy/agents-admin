// Package docker 实现 Docker 容器运行时
//
// 用于运行 Agent 任务的 Docker 容器管理
package docker

import (
	"context"
	"fmt"
	"io"

	"agents-admin/internal/nodemanager/runtime"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// Runtime Docker 容器运行时
type Runtime struct {
	client *client.Client
}

// New 创建 Docker 运行时
func New() (*Runtime, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Runtime{client: cli}, nil
}

// Name 返回运行时名称
func (r *Runtime) Name() string {
	return "docker"
}

// Close 关闭运行时
func (r *Runtime) Close() error {
	return r.client.Close()
}

// Ping 检查 Docker 连接
func (r *Runtime) Ping(ctx context.Context) error {
	_, err := r.client.Ping(ctx, client.PingOptions{})
	return err
}

// Create 创建运行时实例
func (r *Runtime) Create(ctx context.Context, config *runtime.InstanceConfig) (*runtime.Instance, error) {
	// 构建端口映射
	exposedPorts := make(network.PortSet)
	portBindings := make(network.PortMap)

	if config.Network != nil {
		for hostPort, containerPort := range config.Network.PortMap {
			port := network.MustParsePort(fmt.Sprintf("%d/tcp", containerPort))
			exposedPorts[port] = struct{}{}
			portBindings[port] = []network.PortBinding{
				{HostPort: fmt.Sprintf("%d", hostPort)},
			}
		}
	}

	// 构建挂载配置
	var binds []string
	for _, m := range config.Mounts {
		bind := fmt.Sprintf("%s:%s", m.Source, m.Target)
		if m.ReadOnly {
			bind += ":ro"
		}
		binds = append(binds, bind)
	}

	// 构建环境变量
	var env []string
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// 构建命令
	cmd := append(config.Command, config.Args...)

	// 创建容器配置
	opts := client.ContainerCreateOptions{
		Name:  config.Name,
		Image: config.Image,
		Config: &container.Config{
			Cmd:          cmd,
			Env:          env,
			WorkingDir:   config.WorkingDir,
			ExposedPorts: exposedPorts,
			Tty:          config.TTY,
			OpenStdin:    config.Stdin,
			AttachStdin:  config.Stdin,
			AttachStdout: true,
			AttachStderr: true,
		},
		HostConfig: &container.HostConfig{
			Binds:        binds,
			PortBindings: portBindings,
		},
	}

	// 设置资源限制
	if config.Resources != nil {
		opts.HostConfig.Resources = container.Resources{
			NanoCPUs: int64(config.Resources.CPULimit * 1e9),
			Memory:   config.Resources.MemoryLimit,
		}
	}

	result, err := r.client.ContainerCreate(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return &runtime.Instance{
		ID:      result.ID,
		Name:    config.Name,
		Runtime: r.Name(),
		Config:  config,
	}, nil
}

// Start 启动运行时实例
func (r *Runtime) Start(ctx context.Context, instanceID string) error {
	_, err := r.client.ContainerStart(ctx, instanceID, client.ContainerStartOptions{})
	return err
}

// Stop 停止运行时实例
func (r *Runtime) Stop(ctx context.Context, instanceID string) error {
	_, err := r.client.ContainerStop(ctx, instanceID, client.ContainerStopOptions{})
	return err
}

// Remove 删除运行时实例
func (r *Runtime) Remove(ctx context.Context, instanceID string, force bool) error {
	_, err := r.client.ContainerRemove(ctx, instanceID, client.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
	return err
}

// Exec 在运行时实例中执行命令
func (r *Runtime) Exec(ctx context.Context, instanceID string, cmd []string) (*runtime.ExecResult, error) {
	execResult, err := r.client.ExecCreate(ctx, instanceID, client.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := r.client.ExecAttach(ctx, execResult.ID, client.ExecAttachOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach exec: %w", err)
	}
	defer attachResp.Close()

	output, err := io.ReadAll(attachResp.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read exec output: %w", err)
	}

	// 获取退出码
	inspectResp, err := r.client.ExecInspect(ctx, execResult.ID, client.ExecInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec: %w", err)
	}

	return &runtime.ExecResult{
		ExitCode: inspectResp.ExitCode,
		Stdout:   string(output),
	}, nil
}

// Attach 附加到运行时实例获取 IO
func (r *Runtime) Attach(ctx context.Context, instanceID string) (*runtime.AttachResult, error) {
	resp, err := r.client.ContainerAttach(ctx, instanceID, client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach container: %w", err)
	}

	return &runtime.AttachResult{
		Conn:   resp.Conn,
		Reader: resp.Reader,
		Writer: resp.Conn,
	}, nil
}

// Status 获取运行时实例状态
func (r *Runtime) Status(ctx context.Context, instanceID string) (*runtime.InstanceStatus, error) {
	result, err := r.client.ContainerInspect(ctx, instanceID, client.ContainerInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return &runtime.InstanceStatus{
				State: runtime.StateUnknown,
				Error: "container not found",
			}, nil
		}
		return nil, err
	}

	state := mapContainerState(string(result.Container.State.Status))

	return &runtime.InstanceStatus{
		State:      state,
		ExitCode:   result.Container.State.ExitCode,
		StartedAt:  result.Container.State.StartedAt,
		FinishedAt: result.Container.State.FinishedAt,
		Error:      result.Container.State.Error,
	}, nil
}

// Logs 获取运行时实例日志
func (r *Runtime) Logs(ctx context.Context, instanceID string, tail int) (io.ReadCloser, error) {
	tailStr := "all"
	if tail > 0 {
		tailStr = fmt.Sprintf("%d", tail)
	}

	result, err := r.client.ContainerLogs(ctx, instanceID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tailStr,
		Follow:     false,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// mapContainerState 映射容器状态
func mapContainerState(status string) runtime.InstanceState {
	switch status {
	case "created":
		return runtime.StateCreated
	case "running":
		return runtime.StateRunning
	case "paused":
		return runtime.StatePaused
	case "restarting":
		return runtime.StateRunning
	case "removing":
		return runtime.StateRemoving
	case "exited":
		return runtime.StateExited
	case "dead":
		return runtime.StateStopped
	default:
		return runtime.StateUnknown
	}
}
