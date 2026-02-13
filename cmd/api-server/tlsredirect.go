package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// httpOnTLSListener 在 TLS 端口上检测纯 HTTP 请求并自动重定向到 HTTPS。
// TLS ClientHello（首字节 0x16）正常传递给上层 TLS 处理；
// 纯 HTTP 请求则返回 301 重定向到 https:// 并关闭连接。
type httpOnTLSListener struct {
	net.Listener
}

func (l *httpOnTLSListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		// 读取首字节以检测 TLS vs HTTP
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		firstByte := make([]byte, 1)
		n, err := io.ReadFull(conn, firstByte)
		conn.SetReadDeadline(time.Time{})

		if err != nil || n == 0 {
			conn.Close()
			continue
		}

		if firstByte[0] == 0x16 {
			// TLS ClientHello → 传递给 TLS 层
			return &prefixConn{Conn: conn, prefix: firstByte}, nil
		}

		// 纯 HTTP → 后台重定向到 HTTPS
		go handleHTTPRedirect(conn, firstByte)
	}
}

// prefixConn 在底层连接前追加已读取的字节
type prefixConn struct {
	net.Conn
	prefix []byte
}

func (c *prefixConn) Read(b []byte) (int, error) {
	if len(c.prefix) > 0 {
		n := copy(b, c.prefix)
		c.prefix = c.prefix[n:]
		return n, nil
	}
	return c.Conn.Read(b)
}

// handleHTTPRedirect 处理误发到 TLS 端口的 HTTP 请求，返回 301 重定向到 HTTPS
func handleHTTPRedirect(conn net.Conn, firstByte []byte) {
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	reader := bufio.NewReader(&prefixConn{Conn: conn, prefix: firstByte})
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	host := req.Host
	if host == "" {
		host = conn.LocalAddr().String()
	}

	// 确保 host 包含端口（如果不是标准 HTTPS 端口 443）
	_, port, _ := net.SplitHostPort(conn.LocalAddr().String())
	if port != "443" {
		if _, _, err := net.SplitHostPort(host); err != nil {
			// host 没有端口，追加实际监听端口
			host = net.JoinHostPort(host, port)
		}
	}

	target := "https://" + host + req.URL.RequestURI()

	resp := fmt.Sprintf(
		"HTTP/1.1 301 Moved Permanently\r\nLocation: %s\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
		target,
	)
	conn.Write([]byte(resp))
}
