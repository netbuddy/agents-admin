package auth

import (
	"net/http"

	"agents-admin/internal/shared/model"
)

// ListAgentTypes 列出所有预定义 Agent 类型
//
// GET /api/v1/agent-types
func (h *Handler) ListAgentTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agent_types": model.PredefinedAgentTypes,
	})
}

// GetAgentType 获取指定 Agent 类型
//
// GET /api/v1/agent-types/{id}
func (h *Handler) GetAgentType(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for _, at := range model.PredefinedAgentTypes {
		if at.ID == id {
			writeJSON(w, http.StatusOK, at)
			return
		}
	}
	writeError(w, http.StatusNotFound, "agent type not found")
}
