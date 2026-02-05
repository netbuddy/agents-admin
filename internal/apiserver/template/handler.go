// Package template 模板领域 - HTTP 处理
package template

import (
	"encoding/json"
	"net/http"
	"time"

	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/storage"
)

// Handler 模板领域 HTTP 处理器
type Handler struct {
	store storage.PersistentStore
}

// NewHandler 创建模板处理器
func NewHandler(store storage.PersistentStore) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册模板相关路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Task Templates
	mux.HandleFunc("GET /api/v1/task-templates", h.ListTaskTemplates)
	mux.HandleFunc("GET /api/v1/task-templates/{id}", h.GetTaskTemplate)
	mux.HandleFunc("POST /api/v1/task-templates", h.CreateTaskTemplate)
	mux.HandleFunc("DELETE /api/v1/task-templates/{id}", h.DeleteTaskTemplate)

	// Agent Templates
	mux.HandleFunc("GET /api/v1/agent-templates", h.ListAgentTemplates)
	mux.HandleFunc("GET /api/v1/agent-templates/{id}", h.GetAgentTemplate)
	mux.HandleFunc("POST /api/v1/agent-templates", h.CreateAgentTemplate)
	mux.HandleFunc("DELETE /api/v1/agent-templates/{id}", h.DeleteAgentTemplate)

	// Skills
	mux.HandleFunc("GET /api/v1/skills", h.ListSkills)
	mux.HandleFunc("GET /api/v1/skills/{id}", h.GetSkill)
	mux.HandleFunc("POST /api/v1/skills", h.CreateSkill)
	mux.HandleFunc("DELETE /api/v1/skills/{id}", h.DeleteSkill)

	// MCP Servers
	mux.HandleFunc("GET /api/v1/mcp-servers", h.ListMCPServers)
	mux.HandleFunc("GET /api/v1/mcp-servers/{id}", h.GetMCPServer)
	mux.HandleFunc("POST /api/v1/mcp-servers", h.CreateMCPServer)
	mux.HandleFunc("DELETE /api/v1/mcp-servers/{id}", h.DeleteMCPServer)

	// Security Policies
	mux.HandleFunc("GET /api/v1/security-policies", h.ListSecurityPolicies)
	mux.HandleFunc("GET /api/v1/security-policies/{id}", h.GetSecurityPolicy)
	mux.HandleFunc("POST /api/v1/security-policies", h.CreateSecurityPolicy)
	mux.HandleFunc("DELETE /api/v1/security-policies/{id}", h.DeleteSecurityPolicy)
}

// ============================================================================
// Task Template
// ============================================================================

func (h *Handler) ListTaskTemplates(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	templates, err := h.store.ListTaskTemplates(r.Context(), category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list task templates")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"templates": templates, "count": len(templates)})
}

func (h *Handler) GetTaskTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tmpl, err := h.store.GetTaskTemplate(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task template")
		return
	}
	if tmpl == nil {
		writeError(w, http.StatusNotFound, "task template not found")
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

func (h *Handler) CreateTaskTemplate(w http.ResponseWriter, r *http.Request) {
	var tmpl model.TaskTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	if tmpl.ID == "" {
		tmpl.ID = generateID("tmpl")
	}
	tmpl.CreatedAt = now
	tmpl.UpdatedAt = now

	if err := h.store.CreateTaskTemplate(r.Context(), &tmpl); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task template")
		return
	}
	writeJSON(w, http.StatusCreated, tmpl)
}

func (h *Handler) DeleteTaskTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteTaskTemplate(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task template")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Agent Template
// ============================================================================

func (h *Handler) ListAgentTemplates(w http.ResponseWriter, r *http.Request) {
	agentType := r.URL.Query().Get("agent_type")
	templates, err := h.store.ListAgentTemplates(r.Context(), agentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list agent templates")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"templates": templates, "count": len(templates)})
}

func (h *Handler) GetAgentTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tmpl, err := h.store.GetAgentTemplate(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get agent template")
		return
	}
	if tmpl == nil {
		writeError(w, http.StatusNotFound, "agent template not found")
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

func (h *Handler) CreateAgentTemplate(w http.ResponseWriter, r *http.Request) {
	var tmpl model.AgentTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	if tmpl.ID == "" {
		tmpl.ID = generateID("agent-tmpl")
	}
	tmpl.CreatedAt = now
	tmpl.UpdatedAt = now

	if err := h.store.CreateAgentTemplate(r.Context(), &tmpl); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create agent template")
		return
	}
	writeJSON(w, http.StatusCreated, tmpl)
}

func (h *Handler) DeleteAgentTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteAgentTemplate(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete agent template")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Skill
// ============================================================================

func (h *Handler) ListSkills(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	skills, err := h.store.ListSkills(r.Context(), category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list skills")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"skills": skills, "count": len(skills)})
}

func (h *Handler) GetSkill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	skill, err := h.store.GetSkill(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get skill")
		return
	}
	if skill == nil {
		writeError(w, http.StatusNotFound, "skill not found")
		return
	}
	writeJSON(w, http.StatusOK, skill)
}

func (h *Handler) CreateSkill(w http.ResponseWriter, r *http.Request) {
	var skill model.Skill
	if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	if skill.ID == "" {
		skill.ID = generateID("skill")
	}
	skill.CreatedAt = now
	skill.UpdatedAt = now

	if err := h.store.CreateSkill(r.Context(), &skill); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create skill")
		return
	}
	writeJSON(w, http.StatusCreated, skill)
}

func (h *Handler) DeleteSkill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteSkill(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete skill")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// MCP Server
// ============================================================================

func (h *Handler) ListMCPServers(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	servers, err := h.store.ListMCPServers(r.Context(), source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list MCP servers")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"mcp_servers": servers, "count": len(servers)})
}

func (h *Handler) GetMCPServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	server, err := h.store.GetMCPServer(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get MCP server")
		return
	}
	if server == nil {
		writeError(w, http.StatusNotFound, "MCP server not found")
		return
	}
	writeJSON(w, http.StatusOK, server)
}

func (h *Handler) CreateMCPServer(w http.ResponseWriter, r *http.Request) {
	var server model.MCPServer
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	if server.ID == "" {
		server.ID = generateID("mcp")
	}
	server.CreatedAt = now
	server.UpdatedAt = now

	if err := h.store.CreateMCPServer(r.Context(), &server); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create MCP server")
		return
	}
	writeJSON(w, http.StatusCreated, server)
}

func (h *Handler) DeleteMCPServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteMCPServer(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete MCP server")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Security Policy
// ============================================================================

func (h *Handler) ListSecurityPolicies(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	policies, err := h.store.ListSecurityPolicies(r.Context(), category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list security policies")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"policies": policies, "count": len(policies)})
}

func (h *Handler) GetSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	policy, err := h.store.GetSecurityPolicy(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get security policy")
		return
	}
	if policy == nil {
		writeError(w, http.StatusNotFound, "security policy not found")
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (h *Handler) CreateSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	var policy model.SecurityPolicyEntity
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	if policy.ID == "" {
		policy.ID = generateID("sec")
	}
	policy.CreatedAt = now
	policy.UpdatedAt = now

	if err := h.store.CreateSecurityPolicy(r.Context(), &policy); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create security policy")
		return
	}
	writeJSON(w, http.StatusCreated, policy)
}

func (h *Handler) DeleteSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteSecurityPolicy(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete security policy")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// 工具函数
// ============================================================================

func generateID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405")
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
