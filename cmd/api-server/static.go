package main

import (
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
)

// newSPAHandler 创建一个 SPA（单页应用）HTTP handler
//
// 优先级：
//  1. API/WebSocket/metrics 路由 → 委托给 apiHandler
//  2. 静态文件匹配 → 从嵌入的文件系统提供服务
//  3. .html 后缀匹配 → Next.js 静态导出的页面路由（如 /accounts → /accounts.html）
//  4. 兜底 → 直接返回 index.html 内容（SPA 客户端路由接管）
//
// 注意：步骤4 不能使用 http.FileServer，因为 FileServer 对 /index.html 路径
// 会发送 301 重定向到 ./，导致非根路径（如 /some/path）产生无限重定向循环。
func newSPAHandler(apiHandler http.Handler, staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))

	// 预加载 index.html 内容到内存，避免每次请求都读取
	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		log.Fatalf("Failed to read index.html from embedded FS: %v", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path

		// 1. API、WebSocket、监控端点 → 交给原有路由处理
		if strings.HasPrefix(urlPath, "/api/") ||
			strings.HasPrefix(urlPath, "/ws/") ||
			strings.HasPrefix(urlPath, "/ttyd/") ||
			strings.HasPrefix(urlPath, "/spec") ||
			strings.HasPrefix(urlPath, "/docs") ||
			urlPath == "/health" ||
			urlPath == "/metrics" {
			apiHandler.ServeHTTP(w, r)
			return
		}

		// 2. 根路径 → 直接返回 index.html（避免 FileServer 的重定向行为）
		cleanPath := path.Clean(urlPath)
		if cleanPath == "/" {
			serveIndexHTML(w, indexHTML)
			return
		}

		// 3. 尝试精确匹配静态文件（JS/CSS/图片等）
		//    判断方式：尝试从 embed.FS 中 Open 文件，不依赖扩展名
		if tryServeFile(staticFS, cleanPath) {
			fileServer.ServeHTTP(w, r)
			return
		}

		// 4. 尝试 .html 后缀匹配（Next.js 静态导出: /accounts → /accounts.html）
		//    仅对无扩展名的路径尝试（有扩展名说明是静态资源，步骤3已处理）
		if !strings.Contains(path.Base(cleanPath), ".") {
			htmlPath := cleanPath + ".html"
			if tryServeFile(staticFS, htmlPath) {
				// 直接读取 .html 文件内容返回，避免 FileServer 重定向
				serveHTMLFile(w, staticFS, htmlPath)
				return
			}
		}

		// 5. SPA 兜底：直接返回 index.html 内容，由客户端路由处理
		//    不使用 fileServer.ServeHTTP，因为 FileServer 会对 /index.html
		//    发送 301 重定向到 ./（相对路径），导致非根路径无限循环
		serveIndexHTML(w, indexHTML)
	})
}

// serveIndexHTML 直接写入预加载的 index.html 内容
func serveIndexHTML(w http.ResponseWriter, content []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// htmlBufPool 用于复用 HTML 文件读取的缓冲区
var htmlBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 0, 32*1024) // 32KB 初始容量
		return &buf
	},
}

// serveHTMLFile 从 FS 中读取指定 HTML 文件并返回
func serveHTMLFile(w http.ResponseWriter, fsys fs.FS, filePath string) {
	cleanPath := strings.TrimPrefix(filePath, "/")
	f, err := fsys.Open(cleanPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, f)
}

// tryServeFile 检查文件系统中是否存在指定路径的文件
// 不检查扩展名，只检查文件是否存在且不是目录
func tryServeFile(fsys fs.FS, filePath string) bool {
	cleanPath := strings.TrimPrefix(filePath, "/")
	if cleanPath == "" {
		cleanPath = "."
	}
	f, err := fsys.Open(cleanPath)
	if err != nil {
		return false
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return !stat.IsDir()
}
