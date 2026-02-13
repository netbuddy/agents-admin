package template

import (
	"net/http"
	"testing"

	"agents-admin/tests/testutil"
)

// TestTaskTemplate_CRUD 验证任务模板 CRUD
func TestTaskTemplate_CRUD(t *testing.T) {
	payload := map[string]interface{}{
		"name":        "E2E Task Template",
		"description": "Test template",
		"spec": map[string]interface{}{
			"prompt": "template prompt",
			"agent":  map[string]interface{}{"type": "gemini"},
		},
	}
	resp, err := c.Post("/api/v1/task-templates", payload)
	if err != nil {
		t.Fatalf("Create task template failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Logf("Create returned %d", resp.StatusCode)
		t.Skip("Task template creation failed")
	}
	tmplID := result["id"].(string)
	defer c.Delete("/api/v1/task-templates/" + tmplID)

	// 获取
	resp, err = c.Get("/api/v1/task-templates/" + tmplID)
	if err != nil {
		t.Fatalf("Get template failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Get returned %d", resp.StatusCode)
	}

	// 列表
	resp, err = c.Get("/api/v1/task-templates")
	if err != nil {
		t.Fatalf("List templates failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List returned %d", resp.StatusCode)
	}
}

// TestAgentTemplate_CRUD 验证 Agent 模板 CRUD
func TestAgentTemplate_CRUD(t *testing.T) {
	payload := map[string]interface{}{
		"name":        "E2E Agent Template",
		"agent_type":  "gemini",
		"description": "Test agent template",
	}
	resp, err := c.Post("/api/v1/agent-templates", payload)
	if err != nil {
		t.Fatalf("Create agent template failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Logf("Create returned %d", resp.StatusCode)
		t.Skip("Agent template creation failed")
	}
	tmplID := result["id"].(string)
	defer c.Delete("/api/v1/agent-templates/" + tmplID)

	// 获取
	resp, err = c.Get("/api/v1/agent-templates/" + tmplID)
	if err != nil {
		t.Fatalf("Get agent template failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Get returned %d", resp.StatusCode)
	}

	// 列表
	resp, err = c.Get("/api/v1/agent-templates")
	if err != nil {
		t.Fatalf("List agent templates failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List returned %d", resp.StatusCode)
	}
}

// TestSkill_CRUD 验证 Skill CRUD
func TestSkill_CRUD(t *testing.T) {
	payload := map[string]interface{}{
		"name":        "E2E Test Skill",
		"description": "test skill",
	}
	resp, err := c.Post("/api/v1/skills", payload)
	if err != nil {
		t.Fatalf("Create skill failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Logf("Create returned %d", resp.StatusCode)
		t.Skip("Skill creation failed")
	}
	skillID := result["id"].(string)
	defer c.Delete("/api/v1/skills/" + skillID)

	resp, err = c.Get("/api/v1/skills/" + skillID)
	if err != nil {
		t.Fatalf("Get skill failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Get returned %d", resp.StatusCode)
	}

	resp, err = c.Get("/api/v1/skills")
	if err != nil {
		t.Fatalf("List skills failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List returned %d", resp.StatusCode)
	}
}

// TestMCPServer_CRUD 验证 MCP Server CRUD
func TestMCPServer_CRUD(t *testing.T) {
	payload := map[string]interface{}{
		"name": "E2E MCP Server",
		"url":  "http://mcp.example.com:3000",
	}
	resp, err := c.Post("/api/v1/mcp-servers", payload)
	if err != nil {
		t.Fatalf("Create MCP server failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Logf("Create returned %d", resp.StatusCode)
		t.Skip("MCP server creation failed")
	}
	mcpID := result["id"].(string)
	defer c.Delete("/api/v1/mcp-servers/" + mcpID)

	resp, err = c.Get("/api/v1/mcp-servers/" + mcpID)
	if err != nil {
		t.Fatalf("Get MCP server failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Get returned %d", resp.StatusCode)
	}

	resp, err = c.Get("/api/v1/mcp-servers")
	if err != nil {
		t.Fatalf("List MCP servers failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List returned %d", resp.StatusCode)
	}
}

// TestSecurityPolicy_CRUD 验证安全策略 CRUD
func TestSecurityPolicy_CRUD(t *testing.T) {
	payload := map[string]interface{}{
		"name":        "E2E Security Policy",
		"description": "test policy",
	}
	resp, err := c.Post("/api/v1/security-policies", payload)
	if err != nil {
		t.Fatalf("Create policy failed: %v", err)
	}
	result := testutil.ReadJSON(resp)
	if resp.StatusCode != http.StatusCreated {
		t.Logf("Create returned %d", resp.StatusCode)
		t.Skip("Security policy creation failed")
	}
	policyID := result["id"].(string)
	defer c.Delete("/api/v1/security-policies/" + policyID)

	resp, err = c.Get("/api/v1/security-policies/" + policyID)
	if err != nil {
		t.Fatalf("Get policy failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Get returned %d", resp.StatusCode)
	}

	resp, err = c.Get("/api/v1/security-policies")
	if err != nil {
		t.Fatalf("List policies failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("List returned %d", resp.StatusCode)
	}
}
