package main

import (
	"log"
	"net/http"
	"os"
)

// withCACertEndpoint 包装 handler，在 /ca.pem 路径提供 CA 证书下载
// 仅用于自签名证书模式，方便客户端下载并信任 CA
func withCACertEndpoint(next http.Handler, caFile string) http.Handler {
	if caFile == "" {
		return next
	}

	// 预读 CA 证书内容
	caData, err := os.ReadFile(caFile)
	if err != nil {
		log.Printf("[tls] WARNING: cannot read CA file %s for /ca.pem endpoint: %v", caFile, err)
		return next
	}

	log.Printf("[tls] CA cert download available at: /ca.pem")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ca.pem" {
			w.Header().Set("Content-Type", "application/x-pem-file")
			w.Header().Set("Content-Disposition", "attachment; filename=\"agents-admin-ca.pem\"")
			w.WriteHeader(http.StatusOK)
			w.Write(caData)
			return
		}
		next.ServeHTTP(w, r)
	})
}
