// Package docker 封装 Docker API 客户端
//
// 使用官方 github.com/moby/moby/client 库
// 提供容器管理、IO交互等功能，用于Agent认证流程
package docker

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// ContainerConfig 容器配置
type ContainerConfig struct {
	Name       string            // 容器名称
	Image      string            // 镜像名称
	Entrypoint []string          // 入口点（覆盖镜像默认）
	Cmd        []string          // 启动命令
	Env        []string          // 环境变量
	WorkingDir string            // 工作目录
	Volumes    map[string]string // 挂载卷 volume:container
	PortMap    map[int]int       // 端口映射 host:container
	Tty        bool              // 是否分配TTY
	OpenStdin  bool              // 是否打开标准输入
}

// ContainerIO 容器IO流
type ContainerIO struct {
	Conn   io.ReadWriteCloser // 连接
	Reader io.Reader          // 输出读取
	Writer io.Writer          // 输入写入
}

// Client Docker客户端封装
type Client struct {
	cli *client.Client
}

// NewClient 创建Docker客户端
func NewClient() (*Client, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	return c.cli.Close()
}

// Ping 检查Docker连接
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx, client.PingOptions{})
	return err
}

// CreateVolume 创建数据卷
func (c *Client) CreateVolume(ctx context.Context, name string) error {
	_, err := c.cli.VolumeCreate(ctx, client.VolumeCreateOptions{
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("failed to create volume %s: %w", name, err)
	}
	return nil
}

// VolumeExists 检查数据卷是否存在
func (c *Client) VolumeExists(ctx context.Context, name string) (bool, error) {
	result, err := c.cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return false, err
	}
	for _, v := range result.Items {
		if v.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// RemoveVolume 删除数据卷
func (c *Client) RemoveVolume(ctx context.Context, name string, force bool) error {
	_, err := c.cli.VolumeRemove(ctx, name, client.VolumeRemoveOptions{Force: force})
	return err
}

// CreateContainer 创建容器
func (c *Client) CreateContainer(ctx context.Context, cfg *ContainerConfig) (string, error) {
	// 构建端口映射
	exposedPorts := make(network.PortSet)
	portBindings := make(network.PortMap)
	for hostPort, containerPort := range cfg.PortMap {
		port := network.MustParsePort(fmt.Sprintf("%d/tcp", containerPort))
		exposedPorts[port] = struct{}{}
		portBindings[port] = []network.PortBinding{
			{HostIP: netip.IPv4Unspecified(), HostPort: fmt.Sprintf("%d", hostPort)},
		}
	}

	// 构建挂载配置
	var binds []string
	for hostPath, containerPath := range cfg.Volumes {
		binds = append(binds, fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	opts := client.ContainerCreateOptions{
		Name:  cfg.Name,
		Image: cfg.Image,
		Config: &container.Config{
			Entrypoint:   cfg.Entrypoint,
			Cmd:          cfg.Cmd,
			Env:          cfg.Env,
			WorkingDir:   cfg.WorkingDir,
			ExposedPorts: exposedPorts,
			Tty:          cfg.Tty,
			OpenStdin:    cfg.OpenStdin,
			AttachStdin:  cfg.OpenStdin,
			AttachStdout: true,
			AttachStderr: true,
		},
		HostConfig: &container.HostConfig{
			Binds:        binds,
			PortBindings: portBindings,
		},
	}

	result, err := c.cli.ContainerCreate(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return result.ID, nil
}

// StartContainer 启动容器
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	_, err := c.cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	return err
}

// AttachContainer 附加到容器获取IO
func (c *Client) AttachContainer(ctx context.Context, containerID string) (*ContainerIO, error) {
	resp, err := c.cli.ContainerAttach(ctx, containerID, client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach container: %w", err)
	}

	return &ContainerIO{
		Conn:   resp.Conn,
		Reader: resp.Reader,
		Writer: resp.Conn,
	}, nil
}

// StopContainer 停止容器
func (c *Client) StopContainer(ctx context.Context, containerID string, timeout *int) error {
	opts := client.ContainerStopOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}
	_, err := c.cli.ContainerStop(ctx, containerID, opts)
	return err
}

// RemoveContainer 删除容器
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	_, err := c.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
	return err
}

// ContainerExists 检查容器是否存在
func (c *Client) ContainerExists(ctx context.Context, containerID string) (bool, error) {
	_, err := c.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsContainerRunning 检查容器是否在运行
func (c *Client) IsContainerRunning(ctx context.Context, containerID string) (bool, error) {
	result, err := c.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return result.Container.State.Running, nil
}

// WaitContainer 等待容器退出
func (c *Client) WaitContainer(ctx context.Context, containerID string) (int64, error) {
	waitResult := c.cli.ContainerWait(ctx, containerID, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})

	select {
	case err := <-waitResult.Error:
		if err != nil {
			return -1, err
		}
		return 0, nil
	case resp := <-waitResult.Result:
		return resp.StatusCode, nil
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

// ExecInContainer 在容器中执行命令
func (c *Client) ExecInContainer(ctx context.Context, containerID string, cmd []string) (string, error) {
	execResult, err := c.cli.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := c.cli.ExecAttach(ctx, execResult.ID, client.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach exec: %w", err)
	}
	defer attachResp.Close()

	output, err := io.ReadAll(attachResp.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to read exec output: %w", err)
	}

	return string(output), nil
}

// FileExistsInVolume 检查文件是否存在于数据卷中
// 通过启动临时容器挂载数据卷并检查文件
func (c *Client) FileExistsInVolume(ctx context.Context, volumeName, mountPath, filePath string) (bool, error) {
	containerName := fmt.Sprintf("check_%s_%d", volumeName, time.Now().UnixNano())

	// 创建临时容器
	containerID, err := c.CreateContainer(ctx, &ContainerConfig{
		Name:    containerName,
		Image:   "alpine:latest",
		Cmd:     []string{"test", "-f", filePath},
		Volumes: map[string]string{volumeName: mountPath},
	})
	if err != nil {
		return false, err
	}
	defer c.RemoveContainer(ctx, containerID, true)

	// 启动容器并等待
	if err := c.StartContainer(ctx, containerID); err != nil {
		return false, err
	}

	exitCode, err := c.WaitContainer(ctx, containerID)
	if err != nil {
		return false, err
	}

	return exitCode == 0, nil
}

// ContainerLogs 获取容器日志
func (c *Client) ContainerLogs(ctx context.Context, containerID string, tail string) (io.ReadCloser, error) {
	result, err := c.cli.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Follow:     false,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
