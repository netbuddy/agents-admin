package auth

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	// 测试Ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}
}

func TestVolumeOperations(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}

	volumeName := "test_volume_" + time.Now().Format("20060102150405")

	// 创建Volume
	t.Run("CreateVolume", func(t *testing.T) {
		err := client.CreateVolume(ctx, volumeName)
		if err != nil {
			t.Fatalf("Failed to create volume: %v", err)
		}
	})

	// 检查Volume存在
	t.Run("VolumeExists", func(t *testing.T) {
		exists, err := client.VolumeExists(ctx, volumeName)
		if err != nil {
			t.Fatalf("Failed to check volume: %v", err)
		}
		if !exists {
			t.Fatal("Volume should exist")
		}
	})

	// 删除Volume
	t.Run("RemoveVolume", func(t *testing.T) {
		err := client.RemoveVolume(ctx, volumeName, true)
		if err != nil {
			t.Fatalf("Failed to remove volume: %v", err)
		}
	})

	// 确认Volume已删除
	t.Run("VolumeNotExists", func(t *testing.T) {
		exists, err := client.VolumeExists(ctx, volumeName)
		if err != nil {
			t.Fatalf("Failed to check volume: %v", err)
		}
		if exists {
			t.Fatal("Volume should not exist")
		}
	})
}

func TestContainerOperations(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Docker daemon not running: %v", err)
	}

	containerName := "test_container_" + time.Now().Format("20060102150405")

	// 创建容器
	t.Run("CreateContainer", func(t *testing.T) {
		cfg := &ContainerConfig{
			Name:  containerName,
			Image: "alpine:latest",
			Cmd:   []string{"echo", "hello"},
		}

		containerID, err := client.CreateContainer(ctx, cfg)
		if err != nil {
			t.Fatalf("Failed to create container: %v", err)
		}
		if containerID == "" {
			t.Fatal("Container ID should not be empty")
		}

		// 清理
		defer client.RemoveContainer(ctx, containerID, true)

		// 检查容器存在
		exists, err := client.ContainerExists(ctx, containerID)
		if err != nil {
			t.Fatalf("Failed to check container: %v", err)
		}
		if !exists {
			t.Fatal("Container should exist")
		}

		// 启动容器
		err = client.StartContainer(ctx, containerID)
		if err != nil {
			t.Fatalf("Failed to start container: %v", err)
		}

		// 等待容器退出
		exitCode, err := client.WaitContainer(ctx, containerID)
		if err != nil {
			t.Fatalf("Failed to wait container: %v", err)
		}
		if exitCode != 0 {
			t.Fatalf("Expected exit code 0, got %d", exitCode)
		}
	})
}

func TestContainerConfig(t *testing.T) {
	cfg := &ContainerConfig{
		Name:       "test",
		Image:      "alpine:latest",
		Cmd:        []string{"sh", "-c", "echo hello"},
		Env:        []string{"FOO=bar"},
		WorkingDir: "/tmp",
		Volumes:    map[string]string{"test_vol": "/data"},
		PortMap:    map[int]int{8080: 80},
		Tty:        true,
		OpenStdin:  true,
	}

	if cfg.Name != "test" {
		t.Error("Name mismatch")
	}
	if cfg.Image != "alpine:latest" {
		t.Error("Image mismatch")
	}
	if len(cfg.Cmd) != 3 {
		t.Error("Cmd length mismatch")
	}
	if cfg.Tty != true {
		t.Error("Tty should be true")
	}
}
