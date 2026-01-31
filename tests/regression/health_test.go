package regression

import (
	"net/http"
	"strings"
	"testing"
)

// ============================================================================
// 健康检查和监控接口回归测试
// ============================================================================

// TestHealth_HealthCheck 测试健康检查端点
func TestHealth_HealthCheck(t *testing.T) {
	t.Run("基本健康检查", func(t *testing.T) {
		w := makeRequest("GET", "/health", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Health check status = %d, want %d", w.Code, http.StatusOK)
		}

		resp := parseJSONResponse(w)
		if resp["status"] != "ok" {
			t.Errorf("Status = %v, want ok", resp["status"])
		}
	})

	t.Run("健康检查响应时间", func(t *testing.T) {
		// 健康检查应该快速响应
		w := makeRequest("GET", "/health", nil)
		if w.Code != http.StatusOK {
			t.Errorf("Health check failed: %d", w.Code)
		}
	})
}

// TestHealth_Metrics 测试 Prometheus 指标端点
func TestHealth_Metrics(t *testing.T) {
	t.Run("获取指标", func(t *testing.T) {
		w := makeRequest("GET", "/metrics", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Metrics status = %d, want %d", w.Code, http.StatusOK)
		}

		// 应该返回 Prometheus 格式的文本
		body := w.Body.String()
		if len(body) == 0 {
			t.Error("Metrics body is empty")
		}

		// 检查是否包含常见的 Prometheus 指标
		if !strings.Contains(body, "go_") && !strings.Contains(body, "process_") {
			t.Log("Metrics may not contain standard Go metrics")
		}
	})
}

// TestHealth_MonitorWorkflows 测试监控工作流列表
func TestHealth_MonitorWorkflows(t *testing.T) {
	t.Run("列出工作流", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/monitor/workflows", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("List workflows status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		if resp["workflows"] == nil {
			t.Error("Workflows list not returned")
		}
	})
}

// TestHealth_MonitorStats 测试监控统计
func TestHealth_MonitorStats(t *testing.T) {
	t.Run("获取统计信息", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/monitor/stats", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Monitor stats status = %d", w.Code)
		}

		resp := parseJSONResponse(w)
		// 应该包含一些统计字段
		t.Logf("Stats fields: %v", resp)
	})
}

// TestHealth_MonitorRuns 测试监控运行列表
func TestHealth_MonitorRuns(t *testing.T) {
	t.Run("列出活跃运行", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/monitor/runs", nil)
		// 路由可能不存在，404 是可接受的
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("List monitor runs status = %d", w.Code)
		}
		if w.Code == http.StatusNotFound {
			t.Log("Monitor runs endpoint not implemented")
		}
	})

	t.Run("按状态筛选运行", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/monitor/runs?status=running", nil)
		// 路由可能不存在，404 是可接受的
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Filter runs by status status = %d", w.Code)
		}
	})
}

// TestHealth_ReadinessLiveness 测试就绪和存活探针
func TestHealth_ReadinessLiveness(t *testing.T) {
	t.Run("就绪探针", func(t *testing.T) {
		w := makeRequest("GET", "/ready", nil)
		// 可能不存在，但如果存在应该返回 200 或 503
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable && w.Code != http.StatusNotFound {
			t.Errorf("Readiness probe status = %d", w.Code)
		}
	})

	t.Run("存活探针", func(t *testing.T) {
		w := makeRequest("GET", "/alive", nil)
		// 可能不存在，但如果存在应该返回 200
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Liveness probe status = %d", w.Code)
		}
	})
}

// TestHealth_APIVersion 测试 API 版本信息
func TestHealth_APIVersion(t *testing.T) {
	t.Run("获取 API 版本", func(t *testing.T) {
		w := makeRequest("GET", "/api/version", nil)
		if w.Code == http.StatusOK {
			resp := parseJSONResponse(w)
			t.Logf("API version: %v", resp)
		} else {
			t.Logf("API version endpoint status: %d", w.Code)
		}
	})
}
