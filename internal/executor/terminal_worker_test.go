package executor

import "testing"

func TestBuildTTYDDockerRunArgs(t *testing.T) {
	container := "agent_inst_qwen-code_freebuddy_at_gmail_com_1769760979"
	shell := "bash"

	args := buildTTYDDockerRunArgs(container, shell)

	want := []string{
		"run", "-d",
		"--name", "ttyd_terminal",
		"--entrypoint", "ttyd",
		"-p", "7681:7681",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		ttydImage,
		"-W", "-p", "7681",
		"docker", "exec", "-it", container, shell,
	}
	if len(args) != len(want) {
		t.Fatalf("args len=%d want=%d; args=%v", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d]=%q want=%q; args=%v", i, args[i], want[i], args)
		}
	}

	// 2) sanity：不应引入 shell 包装（sh -lc）
	for _, a := range args {
		if a == "sh" || a == "-lc" {
			t.Fatalf("unexpected shell wrapper in args: %v", args)
		}
	}
}
