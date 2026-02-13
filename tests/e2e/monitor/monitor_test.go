package monitor

import (
	"net/http"
	"testing"

	"agents-admin/tests/testutil"
)

// TestMonitor_Stats 验证监控统计接口
func TestMonitor_Stats(t *testing.T) {
	resp, err := c.Get("/api/v1/monitor/stats")
	if err != nil {
		t.Fatalf("Get stats failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Stats returned %d", resp.StatusCode)
	}
	t.Logf("Stats: %v", result)
}

// TestMonitor_Workflows 验证工作流监控列表
func TestMonitor_Workflows(t *testing.T) {
	resp, err := c.Get("/api/v1/monitor/workflows")
	if err != nil {
		t.Fatalf("List workflows failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Workflows returned %d", resp.StatusCode)
	}
	if result["workflows"] == nil {
		t.Error("Workflows list not returned")
	}
}
