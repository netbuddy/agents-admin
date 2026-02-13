package auth

import (
	"net/http"
	"testing"
)

func TestIsPublicRoute(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		expected bool
	}{
		// 公开路由
		{"login", "POST", "/api/v1/auth/login", true},
		{"register", "POST", "/api/v1/auth/register", true},
		{"health", "GET", "/health", true},
		{"heartbeat", "POST", "/api/v1/nodes/heartbeat", true},
		{"bootstrap", "GET", "/api/v1/node-bootstrap", true},
		{"metrics", "GET", "/metrics", true},
		{"ws", "GET", "/ws/monitor", true},

		// NodeManager 路由不再公开（需要 X-Node-Token）
		{"node runs needs token", "GET", "/api/v1/nodes/node-1/runs", false},
		{"node agents needs token", "GET", "/api/v1/nodes/node-1/agents", false},
		{"node terminal-sessions needs token", "GET", "/api/v1/nodes/node-1/terminal-sessions", false},
		{"patch agent needs token", "PATCH", "/api/v1/agents/inst-001", false},
		{"patch terminal-session needs token", "PATCH", "/api/v1/terminal-sessions/term-001", false},
		{"patch action needs token", "PATCH", "/api/v1/actions/act-123", false},
		{"get agent-type needs token", "GET", "/api/v1/agent-types/qwen-code", false},
		{"get account needs token", "GET", "/api/v1/accounts/acc-1", false},

		// 普通用户路由需要 JWT
		{"create operation", "POST", "/api/v1/operations", false},
		{"list operations", "GET", "/api/v1/operations", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPublicRoute(tt.method, tt.path)
			if got != tt.expected {
				t.Errorf("isPublicRoute(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.expected)
			}
		})
	}
}

func TestIsValidNodeToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string // 配置中的 NodeToken
		header   string // 请求中的 X-Node-Token
		expected bool
	}{
		{"valid token", "secret123", "secret123", true},
		{"wrong token", "secret123", "wrong", false},
		{"empty header", "secret123", "", false},
		{"no config token", "", "secret123", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/api/v1/nodes/node-1/runs", nil)
			if tt.header != "" {
				r.Header.Set("X-Node-Token", tt.header)
			}
			got := isValidNodeToken(r, tt.token)
			if got != tt.expected {
				t.Errorf("isValidNodeToken() = %v, want %v", got, tt.expected)
			}
		})
	}
}
