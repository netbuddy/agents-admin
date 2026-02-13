// Package auth 认证操作领域 - HTTP 处理
//
// 处理认证相关的系统操作：
//   - Agent 类型查询
//   - 账号管理（只读 + 删除）
//   - 认证操作创建（OAuth / API Key / Device Code）
//   - 认证结果处理（创建 Account）
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"agents-admin/internal/shared/model"
)

// writeJSON 将数据以 JSON 格式写入 HTTP 响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError 将错误信息以 JSON 格式写入 HTTP 响应
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// generateID 生成带前缀的唯一标识符
func generateID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// sanitizeName 将名称中的特殊字符替换为下划线
// 注意：必须与 nodemanager/auth_controller.go 的 sanitizeForVolume 保持一致
func sanitizeName(name string) string {
	replacer := strings.NewReplacer("@", "_", ".", "_", " ", "_", "-", "_")
	return replacer.Replace(name)
}

// findAgentType 查找 Agent 类型配置
func findAgentType(agentTypeID string) *model.AgentTypeConfig {
	for _, at := range model.PredefinedAgentTypes {
		if at.ID == agentTypeID {
			return &at
		}
	}
	return nil
}

// agentTypeSupportsMethod 检查 Agent 类型是否支持指定的认证方式
func agentTypeSupportsMethod(at *model.AgentTypeConfig, method string) bool {
	for _, m := range at.LoginMethods {
		if m == method {
			return true
		}
	}
	return false
}
