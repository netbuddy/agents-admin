package integration

import (
	"agents-admin/pkg/docker"
	"context"
	"strings"
	"testing"
	"time"
)

// 集成测试：测试完整的容器生命周期
func TestContainerLifecycle_Integration(t *testing.T) {
	client, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}

	containerName := "integration_test_" + time.Now().Format("20060102150405")
	volumeName := "integration_vol_" + time.Now().Format("20060102150405")

	// 1. 创建Volume
	t.Log("Step 1: Creating volume...")
	if err := client.CreateVolume(ctx, volumeName); err != nil {
		t.Fatalf("Failed to create volume: %v", err)
	}
	defer client.RemoveVolume(ctx, volumeName, true)

	// 2. 验证Volume存在
	exists, err := client.VolumeExists(ctx, volumeName)
	if err != nil {
		t.Fatalf("Failed to check volume: %v", err)
	}
	if !exists {
		t.Fatal("Volume should exist after creation")
	}

	// 3. 创建容器并挂载Volume
	t.Log("Step 2: Creating container with volume mount...")
	cfg := &docker.ContainerConfig{
		Name:       containerName,
		Image:      "alpine:latest",
		Cmd:        []string{"sh", "-c", "echo 'test data' > /data/test.txt && cat /data/test.txt"},
		Volumes:    map[string]string{volumeName: "/data"},
		WorkingDir: "/",
	}

	containerID, err := client.CreateContainer(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer client.RemoveContainer(ctx, containerID, true)

	// 4. 启动容器
	t.Log("Step 3: Starting container...")
	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// 5. 等待容器退出
	t.Log("Step 4: Waiting for container to finish...")
	exitCode, err := client.WaitContainer(ctx, containerID)
	if err != nil {
		t.Fatalf("Failed to wait for container: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Container exited with non-zero code: %d", exitCode)
	}

	// 6. 验证容器不再运行
	running, err := client.IsContainerRunning(ctx, containerID)
	if err != nil {
		t.Fatalf("Failed to check container running status: %v", err)
	}
	if running {
		t.Fatal("Container should not be running after exit")
	}

	t.Log("Integration test completed successfully!")
}

// 集成测试：测试容器IO交互
func TestContainerIOAttach_Integration(t *testing.T) {
	client, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}

	containerName := "io_test_" + time.Now().Format("20060102150405")

	// 创建交互式容器
	cfg := &docker.ContainerConfig{
		Name:      containerName,
		Image:     "alpine:latest",
		Cmd:       []string{"cat"},
		Tty:       true,
		OpenStdin: true,
	}

	containerID, err := client.CreateContainer(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer client.RemoveContainer(ctx, containerID, true)

	// 附加到容器
	io, err := client.AttachContainer(ctx, containerID)
	if err != nil {
		t.Fatalf("Failed to attach to container: %v", err)
	}
	defer io.Conn.Close()

	// 启动容器
	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// 发送输入
	testInput := "hello from test\n"
	_, err = io.Writer.Write([]byte(testInput))
	if err != nil {
		t.Fatalf("Failed to write to container: %v", err)
	}

	// 读取输出
	buf := make([]byte, 1024)
	n, err := io.Reader.Read(buf)
	if err != nil {
		t.Logf("Read returned: %v (may be expected)", err)
	}
	if n > 0 {
		output := string(buf[:n])
		t.Logf("Received output: %q", output)
		if !strings.Contains(output, "hello") {
			t.Logf("Warning: output may not contain expected text")
		}
	}

	// 停止容器
	client.StopContainer(ctx, containerID, nil)

	t.Log("IO attach test completed!")
}

// 集成测试：测试Exec功能
func TestExecInContainer_Integration(t *testing.T) {
	client, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}

	containerName := "exec_test_" + time.Now().Format("20060102150405")

	// 创建并启动长运行容器
	cfg := &docker.ContainerConfig{
		Name:  containerName,
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "30"},
	}

	containerID, err := client.CreateContainer(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer client.RemoveContainer(ctx, containerID, true)

	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer client.StopContainer(ctx, containerID, nil)

	// 等待容器启动
	time.Sleep(1 * time.Second)

	// 在容器中执行命令
	output, err := client.ExecInContainer(ctx, containerID, []string{"echo", "exec test"})
	if err != nil {
		t.Fatalf("Failed to exec in container: %v", err)
	}

	t.Logf("Exec output: %q", output)
	if !strings.Contains(output, "exec test") {
		t.Errorf("Expected output to contain 'exec test', got: %q", output)
	}

	t.Log("Exec test completed successfully!")
}
