// Package main Mock Runner - 模拟 Agent CLI 输出
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Event struct {
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

func emit(eventType string, data map[string]interface{}) {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Data:      data,
	}
	b, _ := json.Marshal(event)
	fmt.Println(string(b))
}

func main() {
	prompt := "default task"
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}

	emit("run_started", map[string]interface{}{
		"prompt": prompt,
	})

	time.Sleep(500 * time.Millisecond)

	emit("message", map[string]interface{}{
		"role":    "assistant",
		"content": "I'll analyze the task and start working on it...",
	})

	time.Sleep(1 * time.Second)

	emit("tool_use_start", map[string]interface{}{
		"tool_id":   "tool_001",
		"tool_name": "Read",
		"input": map[string]interface{}{
			"path": "src/main.go",
		},
	})

	time.Sleep(500 * time.Millisecond)

	emit("tool_result", map[string]interface{}{
		"tool_id": "tool_001",
		"success": true,
		"output":  "// main.go content\npackage main\n\nfunc main() {}\n",
	})

	time.Sleep(1 * time.Second)

	emit("message", map[string]interface{}{
		"role":    "assistant",
		"content": "I've read the file. Now I'll make the necessary changes...",
	})

	time.Sleep(500 * time.Millisecond)

	emit("tool_use_start", map[string]interface{}{
		"tool_id":   "tool_002",
		"tool_name": "Write",
		"input": map[string]interface{}{
			"path":    "src/main.go",
			"content": "// Updated main.go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n",
		},
	})

	time.Sleep(500 * time.Millisecond)

	emit("tool_result", map[string]interface{}{
		"tool_id": "tool_002",
		"success": true,
		"output":  "File written successfully",
	})

	time.Sleep(1 * time.Second)

	emit("command", map[string]interface{}{
		"command": "go build -o main ./src",
		"cwd":     "/workspace",
	})

	time.Sleep(1 * time.Second)

	emit("command_output", map[string]interface{}{
		"stdout":    "",
		"stderr":    "",
		"exit_code": 0,
	})

	time.Sleep(500 * time.Millisecond)

	emit("message", map[string]interface{}{
		"role":    "assistant",
		"content": "Task completed successfully! I've updated main.go and verified the build.",
	})

	time.Sleep(500 * time.Millisecond)

	emit("run_completed", map[string]interface{}{
		"status":      "success",
		"duration_ms": 6500,
		"artifacts": []map[string]interface{}{
			{"name": "events.jsonl", "path": ".agent/events.jsonl"},
		},
	})
}
