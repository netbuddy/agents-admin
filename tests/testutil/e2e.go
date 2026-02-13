// Package testutil 提供 E2E 测试共享基础设施
//
// E2EClient 封装了 HTTPS + Cookie 认证的 HTTP 客户端，
// 供 tests/e2e/ 下各子包复用。
package testutil

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"
)

// E2EClient 端到端测试共享客户端
type E2EClient struct {
	BaseURL  string
	Client   *http.Client
	LoggedIn bool
}

// SetupE2EClient 初始化 E2E 客户端
// 自动读取环境变量、创建 HTTPS 客户端、等待服务就绪、登录
// 返回 error 时调用者应 os.Exit(0) 跳过测试
func SetupE2EClient() (*E2EClient, error) {
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://localhost:8080"
	}

	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	c := &E2EClient{
		BaseURL: baseURL,
		Client:  httpClient,
	}

	// 等待 API Server 就绪
	if !c.waitForAPI(15 * time.Second) {
		return nil, fmt.Errorf("API Server not ready at %s", baseURL)
	}

	// 登录
	if err := c.login(); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "e2e: connected to %s\n", baseURL)
	return c, nil
}

func (c *E2EClient) waitForAPI(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := c.Client.Get(c.BaseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func (c *E2EClient) login() error {
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@agents-admin.local"
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin123456"
	}

	payload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	resp, err := c.Client.Post(c.BaseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login returned %d: %s", resp.StatusCode, string(body))
	}
	c.LoggedIn = true
	fmt.Fprintf(os.Stderr, "e2e: logged in as %s\n", email)
	return nil
}

// ---- HTTP 请求辅助方法 ----

// Get 发送 GET 请求
func (c *E2EClient) Get(path string) (*http.Response, error) {
	return c.Client.Get(c.BaseURL + path)
}

// Post 发送 POST 请求（JSON body）
func (c *E2EClient) Post(path string, body interface{}) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reader = bytes.NewReader(jsonBody)
	}
	return c.Client.Post(c.BaseURL+path, "application/json", reader)
}

// PostString 发送 POST 请求（字符串 body）
func (c *E2EClient) PostString(path, body string) (*http.Response, error) {
	return c.Client.Post(c.BaseURL+path, "application/json", bytes.NewBufferString(body))
}

// Do 执行自定义请求
func (c *E2EClient) Do(method, path string, body interface{}) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reader = bytes.NewReader(jsonBody)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.Client.Do(req)
}

// Delete 发送 DELETE 请求
func (c *E2EClient) Delete(path string) (*http.Response, error) {
	return c.Do("DELETE", path, nil)
}

// Patch 发送 PATCH 请求
func (c *E2EClient) Patch(path string, body interface{}) (*http.Response, error) {
	return c.Do("PATCH", path, body)
}

// Put 发送 PUT 请求
func (c *E2EClient) Put(path string, body interface{}) (*http.Response, error) {
	return c.Do("PUT", path, body)
}

// ---- 响应解析辅助 ----

// ReadJSON 解析 JSON 响应到 map
func ReadJSON(resp *http.Response) map[string]interface{} {
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

// ReadJSONKeepBody 解析 JSON 响应但不关闭 Body（调用者需手动关闭）
func ReadJSONKeepBody(resp *http.Response) map[string]interface{} {
	var result map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)
	return result
}
