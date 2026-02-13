// Package sysconfig 系统配置管理 API
//
// 提供配置文件的查看和修改功能，供前端配置管理页面使用。
package sysconfig

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"agents-admin/internal/config"

	"gopkg.in/yaml.v3"
)

// Handler 配置管理 API 处理器
type Handler struct{}

// NewHandler 创建配置管理处理器
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/config", h.GetConfig)
	mux.HandleFunc("PUT /api/v1/config", h.UpdateConfig)
}

// ConfigResponse GET /api/v1/config 响应
type ConfigResponse struct {
	FilePath string                 `json:"file_path"`
	Content  string                 `json:"content"`
	Parsed   map[string]interface{} `json:"parsed"`
}

// GetConfig GET /api/v1/config — 读取当前配置文件
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	data, path, err := config.ReadConfigFile()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Config file not found: " + err.Error(),
		})
		return
	}

	// 解析为通用 map 供前端使用
	var parsed map[string]interface{}
	yaml.Unmarshal(data, &parsed)

	resp := ConfigResponse{
		FilePath: path,
		Content:  string(data),
		Parsed:   parsed,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UpdateConfigRequest PUT /api/v1/config 请求
type UpdateConfigRequest struct {
	Content string `json:"content"`
}

// UpdateConfig PUT /api/v1/config — 更新配置文件
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req UpdateConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON: " + err.Error()})
		return
	}

	if req.Content == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Content cannot be empty"})
		return
	}

	// 验证 YAML 语法
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(req.Content), &parsed); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid YAML: " + err.Error()})
		return
	}

	// 写入文件
	path, err := config.WriteConfigFile([]byte(req.Content))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to write config: " + err.Error()})
		return
	}

	log.Printf("Config file updated: %s", path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"file_path": path,
		"message":   "Configuration saved. Restart the service to apply changes.",
	})
}
