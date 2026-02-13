package setup

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

// handleIndex 返回 index.html
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// handleStatic 处理静态文件请求 /setup/*
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// /setup/foo.js -> static/foo.js
	path := strings.TrimPrefix(r.URL.Path, "/setup/")
	if path == "" || path == "/" {
		s.handleIndex(w, r)
		return
	}

	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// 尝试读取文件
	f, err := sub.Open(path)
	if err != nil {
		// 文件不存在，返回 index.html（SPA fallback）
		s.handleIndex(w, r)
		return
	}
	f.Close()

	fileServer := http.StripPrefix("/setup/", http.FileServer(http.FS(sub)))
	fileServer.ServeHTTP(w, r)
}
