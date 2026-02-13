package health

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"agents-admin/tests/testutil"
)

// TestHealth_Check 验证健康检查端点可用
func TestHealth_Check(t *testing.T) {
	resp, err := c.Get("/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	result := testutil.ReadJSON(resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Health check returned %d", resp.StatusCode)
	}
	if result["status"] != "ok" {
		t.Errorf("Status = %v, want 'ok'", result["status"])
	}
}

// TestHealth_Metrics 验证 Prometheus 指标端点
func TestHealth_Metrics(t *testing.T) {
	resp, err := c.Get("/metrics")
	if err != nil {
		t.Fatalf("Metrics request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Metrics returned %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)

	if len(body) == 0 {
		t.Error("Metrics body is empty")
	}
	if !strings.Contains(body, "go_") && !strings.Contains(body, "process_") {
		t.Log("Metrics may not contain standard Go metrics")
	}
}

// TestHealth_NodeBootstrap 验证节点引导配置端点
func TestHealth_NodeBootstrap(t *testing.T) {
	resp, err := c.Get("/api/v1/node-bootstrap")
	if err != nil {
		t.Fatalf("Node bootstrap request failed: %v", err)
	}
	defer resp.Body.Close()

	// 引导端点应返回 200（即使无配置也应有默认值）
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Node bootstrap returned %d", resp.StatusCode)
	}
}
