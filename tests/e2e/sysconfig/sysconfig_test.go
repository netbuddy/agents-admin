package sysconfig

import (
	"net/http"
	"testing"

	"agents-admin/tests/testutil"
)

// TestSysConfig_Get 验证获取系统配置
func TestSysConfig_Get(t *testing.T) {
	resp, err := c.Get("/api/v1/config")
	if err != nil {
		t.Fatalf("Get config failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get config returned %d", resp.StatusCode)
	}
	t.Logf("Config keys: %v", result)
}

// TestSysConfig_Update 验证更新系统配置
func TestSysConfig_Update(t *testing.T) {
	// 先获取当前配置
	resp, err := c.Get("/api/v1/config")
	if err != nil {
		t.Fatalf("Get config failed: %v", err)
	}
	currentConfig := testutil.ReadJSON(resp)

	// 更新配置（使用当前值确保幂等）
	resp, err = c.Put("/api/v1/config", currentConfig)
	if err != nil {
		t.Fatalf("Update config failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Update config returned %d", resp.StatusCode)
	}
}
