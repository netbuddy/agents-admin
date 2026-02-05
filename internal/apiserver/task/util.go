package task

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	openapi "agents-admin/api/generated/go"
	"agents-admin/internal/shared/model"
)

// writeJSON 写入 JSON 响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError 写入错误响应
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// generateID 生成带前缀的随机 ID
// 格式：prefix-xxxxxxxxxxxx（prefix + 12 字符 hex）
func generateID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// ============================================================================
// OpenAPI -> Model 转换函数
// ============================================================================

// jsonBridgeConvert 通过 JSON 序列化/反序列化转换类型
// 用于 OpenAPI 类型与 model 类型之间的转换
func jsonBridgeConvert[T any](src interface{}) *T {
	if src == nil {
		return nil
	}
	data, err := json.Marshal(src)
	if err != nil {
		return nil
	}
	var dst T
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil
	}
	return &dst
}

// convertTaskContext 将 OpenAPI TaskContext 转换为 model.TaskContext
func convertTaskContext(src *openapi.TaskContext) *model.TaskContext {
	if src == nil {
		return nil
	}
	ctx := &model.TaskContext{}

	if src.InheritedContext != nil {
		for _, item := range *src.InheritedContext {
			ctx.InheritedContext = append(ctx.InheritedContext, convertContextItem(item))
		}
	}

	if src.ProducedContext != nil {
		for _, item := range *src.ProducedContext {
			ctx.ProducedContext = append(ctx.ProducedContext, convertContextItem(item))
		}
	}

	if src.ConversationHistory != nil {
		for _, msg := range *src.ConversationHistory {
			ctx.ConversationHistory = append(ctx.ConversationHistory, convertMessage(msg))
		}
	}

	return ctx
}

// convertContextItem 将 OpenAPI ContextItem 转换为 model.ContextItem
func convertContextItem(src openapi.ContextItem) model.ContextItem {
	item := model.ContextItem{
		Type: src.Type,
		Name: src.Name,
	}
	if src.Content != nil {
		item.Content = *src.Content
	}
	if src.Source != nil {
		item.Source = *src.Source
	}
	return item
}

// convertMessage 将 OpenAPI Message 转换为 model.Message
func convertMessage(src openapi.Message) model.Message {
	msg := model.Message{
		Role:    src.Role,
		Content: src.Content,
	}
	if src.Timestamp != nil {
		msg.Timestamp = *src.Timestamp
	}
	return msg
}
