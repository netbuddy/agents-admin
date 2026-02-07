package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateCerts(t *testing.T) {
	tmpDir := t.TempDir()

	opts := GenerateOptions{
		Hosts:        "10.0.1.50,myserver.local",
		Organization: "Test Org",
		CertDir:      tmpDir,
	}

	err := GenerateCerts(opts)
	if err != nil {
		t.Fatalf("GenerateCerts failed: %v", err)
	}

	// 验证文件存在
	files := DefaultCertFiles(tmpDir)
	for _, f := range []string{files.CAFile, files.CertFile, files.KeyFile} {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// 验证证书可以被解析和加载
	cert, err := tls.LoadX509KeyPair(files.CertFile, files.KeyFile)
	if err != nil {
		t.Fatalf("LoadX509KeyPair failed: %v", err)
	}

	// 验证 CA 可以验证服务端证书
	caPEM, err := os.ReadFile(files.CAFile)
	if err != nil {
		t.Fatalf("read CA file: %v", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		t.Fatal("failed to parse CA cert")
	}

	serverCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse server cert: %v", err)
	}

	// 验证 SANs
	if len(serverCert.IPAddresses) == 0 {
		t.Error("expected IP SANs")
	}
	if len(serverCert.DNSNames) == 0 {
		t.Error("expected DNS SANs")
	}

	t.Logf("DNS SANs: %v", serverCert.DNSNames)
	t.Logf("IP SANs: %v", serverCert.IPAddresses)

	// 验证证书链
	_, err = serverCert.Verify(x509.VerifyOptions{
		Roots: caPool,
	})
	if err != nil {
		t.Fatalf("certificate verification failed: %v", err)
	}

	t.Log("Certificate chain verified successfully")
}

func TestEnsureCerts_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	opts := GenerateOptions{
		CertDir: tmpDir,
	}

	// 第一次生成
	certs1, err := EnsureCerts(opts)
	if err != nil {
		t.Fatalf("first EnsureCerts failed: %v", err)
	}

	// 记录文件修改时间
	info1, _ := os.Stat(filepath.Join(tmpDir, "server.pem"))

	// 第二次应跳过（幂等）
	certs2, err := EnsureCerts(opts)
	if err != nil {
		t.Fatalf("second EnsureCerts failed: %v", err)
	}

	info2, _ := os.Stat(filepath.Join(tmpDir, "server.pem"))

	if info1.ModTime() != info2.ModTime() {
		t.Error("expected certificates to not be regenerated")
	}

	if certs1.CertFile != certs2.CertFile {
		t.Error("expected same cert file path")
	}
}
