package auth

import (
	"net/http"
	"testing"

	"agents-admin/tests/testutil"
)

// TestAuth_Me 验证当前登录用户信息
func TestAuth_Me(t *testing.T) {
	resp, err := c.Get("/api/v1/auth/me")
	if err != nil {
		t.Fatalf("Get me failed: %v", err)
	}
	result := testutil.ReadJSON(resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Me returned %d", resp.StatusCode)
	}
	if result["email"] == nil {
		t.Error("Me response missing email")
	}
}

// TestAuth_RefreshToken 验证令牌刷新
func TestAuth_RefreshToken(t *testing.T) {
	resp, err := c.Post("/api/v1/auth/refresh", nil)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	defer resp.Body.Close()

	// 刷新应该成功（Cookie 中已有有效令牌）
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Refresh returned %d", resp.StatusCode)
	}
}

// TestAuth_ChangePassword 验证密码修改流程
func TestAuth_ChangePassword(t *testing.T) {
	// 使用当前密码尝试修改为相同密码（验证接口可达性）
	payload := map[string]string{
		"old_password": "admin123456",
		"new_password": "admin123456",
	}
	resp, err := c.Put("/api/v1/auth/password", payload)
	if err != nil {
		t.Fatalf("Change password request failed: %v", err)
	}
	defer resp.Body.Close()

	// 接口应可达，可能返回 200（成功）或 400（新旧密码相同）
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Change password returned unexpected %d", resp.StatusCode)
	}
}

// TestAuth_UnauthorizedAccess 验证未认证请求被拒绝
func TestAuth_UnauthorizedAccess(t *testing.T) {
	// 创建无 Cookie 的新客户端
	noAuthClient, err := testutil.SetupE2EClient()
	if err != nil {
		// 如果无法连接，跳过
		t.Skip("Cannot create unauthenticated client")
	}
	// 清除 Cookie（模拟未登录）
	noAuthClient.Client.Jar = nil

	resp, err := noAuthClient.Get("/api/v1/tasks")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// 应该返回 401
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Unauthenticated request returned %d, want 401", resp.StatusCode)
	}
}
