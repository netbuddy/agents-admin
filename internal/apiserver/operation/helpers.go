package operation

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
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
