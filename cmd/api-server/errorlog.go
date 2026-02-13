package main

import (
	"io"
	"log"
	"strings"
)

// tlsErrorFilter 过滤 TLS handshake error 日志
// 自签名证书模式下，浏览器首次连接会触发大量 "TLS handshake error"，
// 这是预期行为，不应刷屏
type tlsErrorFilter struct{}

func (f *tlsErrorFilter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if strings.Contains(msg, "TLS handshake error") {
		// 静默丢弃，不打印
		return len(p), nil
	}
	// 非 TLS 错误正常输出
	return io.Discard.Write(p)
}

// newTLSFilteredLogger 创建一个过滤 TLS handshake error 的 logger
// 用于 http.Server.ErrorLog，抑制自签名证书导致的日志噪音
func newTLSFilteredLogger() *log.Logger {
	return log.New(&tlsErrorFilter{}, "", 0)
}
