// Package tlsutil 提供 TLS 证书自动生成能力
//
// 支持在程序启动时自动生成自签名 CA 和服务端证书，
// 实现内网环境零配置 HTTPS。
package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CertFiles 证书文件路径
type CertFiles struct {
	CAFile   string // CA 证书
	CertFile string // 服务端证书
	KeyFile  string // 服务端私钥
}

// DefaultCertDir 默认证书目录
const DefaultCertDir = "/etc/agents-admin/certs"

// DefaultCertFiles 返回默认证书文件路径
func DefaultCertFiles(dir string) CertFiles {
	if dir == "" {
		dir = DefaultCertDir
	}
	return CertFiles{
		CAFile:   filepath.Join(dir, "ca.pem"),
		CertFile: filepath.Join(dir, "server.pem"),
		KeyFile:  filepath.Join(dir, "server-key.pem"),
	}
}

// CertsExist 检查证书文件是否全部存在
func (c CertFiles) CertsExist() bool {
	for _, f := range []string{c.CAFile, c.CertFile, c.KeyFile} {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// GenerateOptions 证书生成选项
type GenerateOptions struct {
	// Hosts 证书的 SANs（IP 和域名，逗号分隔）
	// 自动包含 localhost 和 127.0.0.1
	Hosts string

	// Organization CA 组织名
	Organization string

	// ValidFor 证书有效期
	ValidFor time.Duration

	// CertDir 证书输出目录
	CertDir string

	// Force 是否覆盖已有证书
	Force bool
}

// DefaultGenerateOptions 返回默认选项
func DefaultGenerateOptions() GenerateOptions {
	return GenerateOptions{
		Hosts:        "localhost,127.0.0.1",
		Organization: "Agents Admin",
		ValidFor:     365 * 24 * time.Hour,
		CertDir:      DefaultCertDir,
		Force:        false,
	}
}

// EnsureCerts 确保证书存在：如果不存在则自动生成
// 返回证书文件路径
func EnsureCerts(opts GenerateOptions) (*CertFiles, error) {
	files := DefaultCertFiles(opts.CertDir)

	if !opts.Force && files.CertsExist() {
		log.Printf("[tls] Certificates already exist in %s", opts.CertDir)
		return &files, nil
	}

	log.Printf("[tls] Auto-generating TLS certificates in %s ...", opts.CertDir)
	if err := GenerateCerts(opts); err != nil {
		return nil, err
	}
	log.Printf("[tls] TLS certificates generated successfully")
	return &files, nil
}

// GenerateCerts 生成 CA 证书 + 服务端证书
func GenerateCerts(opts GenerateOptions) error {
	if opts.CertDir == "" {
		opts.CertDir = DefaultCertDir
	}
	if opts.Organization == "" {
		opts.Organization = "Agents Admin"
	}
	if opts.ValidFor == 0 {
		opts.ValidFor = 365 * 24 * time.Hour
	}

	// 创建目录
	if err := os.MkdirAll(opts.CertDir, 0755); err != nil {
		return fmt.Errorf("create cert dir: %w", err)
	}

	// 收集 SANs
	hosts := collectHosts(opts.Hosts)

	// ============================================================
	// 1. 生成 CA
	// ============================================================
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate CA key: %w", err)
	}

	caSerial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	caTemplate := &x509.Certificate{
		SerialNumber: caSerial,
		Subject: pkix.Name{
			Organization: []string{opts.Organization},
			CommonName:   opts.Organization + " CA",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // CA 10 年
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create CA cert: %w", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return fmt.Errorf("parse CA cert: %w", err)
	}

	// ============================================================
	// 2. 生成服务端证书（由 CA 签发）
	// ============================================================
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate server key: %w", err)
	}

	serverSerial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	serverTemplate := &x509.Certificate{
		SerialNumber: serverSerial,
		Subject: pkix.Name{
			Organization: []string{opts.Organization},
			CommonName:   "Agents Admin Server",
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(opts.ValidFor),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
	}

	// 设置 SANs
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			serverTemplate.IPAddresses = append(serverTemplate.IPAddresses, ip)
		} else {
			serverTemplate.DNSNames = append(serverTemplate.DNSNames, h)
		}
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create server cert: %w", err)
	}

	// ============================================================
	// 3. 写入文件
	// ============================================================
	files := DefaultCertFiles(opts.CertDir)

	// CA 证书（公开，644）
	if err := writePEM(files.CAFile, "CERTIFICATE", caCertDER, 0644); err != nil {
		return fmt.Errorf("write CA cert: %w", err)
	}

	// 服务端证书（公开，644）
	if err := writePEM(files.CertFile, "CERTIFICATE", serverCertDER, 0644); err != nil {
		return fmt.Errorf("write server cert: %w", err)
	}

	// 服务端私钥（敏感，600）
	keyBytes, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		return fmt.Errorf("marshal server key: %w", err)
	}
	if err := writePEM(files.KeyFile, "EC PRIVATE KEY", keyBytes, 0600); err != nil {
		return fmt.Errorf("write server key: %w", err)
	}

	log.Printf("[tls] Generated files:")
	log.Printf("[tls]   CA cert:     %s", files.CAFile)
	log.Printf("[tls]   Server cert: %s (SANs: %s)", files.CertFile, strings.Join(hosts, ", "))
	log.Printf("[tls]   Server key:  %s", files.KeyFile)
	log.Printf("[tls]   Valid for:   %s", opts.ValidFor)

	return nil
}

// collectHosts 收集并去重 hosts，确保包含 localhost 和 127.0.0.1
func collectHosts(hostsStr string) []string {
	seen := make(map[string]bool)
	var result []string

	// 确保基本 hosts 始终包含
	defaults := []string{"localhost", "127.0.0.1", "::1"}
	for _, h := range defaults {
		if !seen[h] {
			seen[h] = true
			result = append(result, h)
		}
	}

	// 添加用户指定的 hosts
	if hostsStr != "" {
		for _, h := range strings.Split(hostsStr, ",") {
			h = strings.TrimSpace(h)
			if h != "" && !seen[h] {
				seen[h] = true
				result = append(result, h)
			}
		}
	}

	// 尝试获取本机 hostname
	if hostname, err := os.Hostname(); err == nil && !seen[hostname] {
		seen[hostname] = true
		result = append(result, hostname)
	}

	// 尝试获取本机 IP
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				ip := ipnet.IP.String()
				if !seen[ip] {
					seen[ip] = true
					result = append(result, ip)
				}
			}
		}
	}

	return result
}

func writePEM(path, blockType string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}
