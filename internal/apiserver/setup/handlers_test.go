package setup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agents-admin/internal/config"
)

func TestGenerateToken(t *testing.T) {
	token := generateToken()
	if len(token) != 64 {
		t.Errorf("expected token length 64, got %d", len(token))
	}
	token2 := generateToken()
	if token == token2 {
		t.Error("expected different tokens")
	}
}

func TestGenerateRandomString(t *testing.T) {
	s := generateRandomString(32)
	if len(s) != 32 {
		t.Errorf("expected length 32, got %d", len(s))
	}
	s2 := generateRandomString(32)
	if s == s2 {
		t.Error("expected different strings")
	}
}

func TestValidateAuth(t *testing.T) {
	tests := []struct {
		email    string
		password string
		wantOK   bool
	}{
		{"", "password", false},
		{"noemail", "password", false},
		{"admin@test.com", "short", false},
		{"admin@test.com", "longpassword", true},
	}
	for _, tt := range tests {
		r := validateAuth(AuthConfig{AdminEmail: tt.email, AdminPassword: tt.password})
		if r.OK != tt.wantOK {
			t.Errorf("validateAuth(%q, %q) = %v, want %v: %s", tt.email, tt.password, r.OK, tt.wantOK, r.Message)
		}
	}
}

func TestHandleInfo(t *testing.T) {
	srv := &Server{configDir: "/tmp/test", token: "test"}
	req := httptest.NewRequest(http.MethodGet, "/setup/api/info", nil)
	w := httptest.NewRecorder()
	srv.handleInfo(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp InfoResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Hostname == "" {
		t.Error("hostname should not be empty")
	}
}

func TestHandleInfoMethodNotAllowed(t *testing.T) {
	srv := &Server{token: "test"}
	req := httptest.NewRequest(http.MethodPost, "/setup/api/info", nil)
	w := httptest.NewRecorder()
	srv.handleInfo(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleValidateBadJSON(t *testing.T) {
	srv := &Server{token: "test"}
	req := httptest.NewRequest(http.MethodPost, "/setup/api/validate", strings.NewReader("invalid"))
	w := httptest.NewRecorder()
	srv.handleValidate(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleValidateDBFail(t *testing.T) {
	srv := &Server{token: "test"}
	body := `{
		"database": {"type":"postgres","host":"127.0.0.1","port":1,"user":"x","password":"x","dbname":"x","sslmode":"disable"},
		"redis": {"host":"127.0.0.1","port":1},
		"auth": {"admin_email":"a@b.com","admin_password":"12345678"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/setup/api/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleValidate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp ValidateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Valid {
		t.Error("should not be valid with unreachable DB")
	}
	if resp.Checks["database"].OK {
		t.Error("database check should fail")
	}
}

func TestWriteYAMLConfigPostgres(t *testing.T) {
	dir := t.TempDir()
	req := ValidateRequest{
		Database: DatabaseConfig{Type: "postgres", Host: "db.local", Port: 5432, User: "admin", Password: "secret", DBName: "mydb", SSLMode: "disable"},
		Redis:    RedisConfig{Host: "redis.local", Port: 6379},
		TLS:      TLSConfig{Enabled: true, Mode: "auto_generate", Hosts: "my.server"},
		Auth:     AuthConfig{AdminEmail: "admin@test.com", AdminPassword: "password123"},
		Server:   ServerConfig{Port: "8080"},
	}

	if err := writeYAMLConfig(dir, req); err != nil {
		t.Fatalf("writeYAMLConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, config.ConfigFileName()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	content := string(data)
	for _, want := range []string{"driver: postgres", "db.local", "5432", "admin", "mydb", "redis.local", "6379", "auto_generate", "my.server", "8080"} {
		if !strings.Contains(content, want) {
			t.Errorf("yaml should contain %q", want)
		}
	}
	// Password should NOT be in yaml
	if strings.Contains(content, "secret") {
		t.Error("yaml should not contain password")
	}
}

func TestWriteYAMLConfigSQLite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	req := ValidateRequest{
		Database: DatabaseConfig{Type: "sqlite", Path: dbPath},
		Redis:    RedisConfig{Host: "localhost", Port: 6379},
		Server:   ServerConfig{Port: "8080"},
	}

	if err := writeYAMLConfig(dir, req); err != nil {
		t.Fatalf("writeYAMLConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, config.ConfigFileName()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	content := string(data)
	for _, want := range []string{"driver: sqlite", dbPath} {
		if !strings.Contains(content, want) {
			t.Errorf("yaml should contain %q", want)
		}
	}
	// Should NOT contain postgres fields
	if strings.Contains(content, "sslmode") {
		t.Error("sqlite yaml should not contain sslmode")
	}
}

func TestWriteEnvConfigPostgres(t *testing.T) {
	dir := t.TempDir()
	req := ValidateRequest{
		Database: DatabaseConfig{Type: "postgres", Host: "db.local", Port: 5432, User: "admin", Password: "secret", DBName: "mydb", SSLMode: "disable"},
		Redis:    RedisConfig{Host: "redis.local", Port: 6379},
		Auth:     AuthConfig{AdminEmail: "admin@test.com", AdminPassword: "password123"},
	}

	if err := writeEnvConfig(dir, req, "jwt-test-secret"); err != nil {
		t.Fatalf("writeEnvConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, config.EnvFileName()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	content := string(data)
	for _, want := range []string{"DB_PASSWORD=secret", "jwt-test-secret", "admin@test.com", "password123"} {
		if !strings.Contains(content, want) {
			t.Errorf("env should contain %q", want)
		}
	}
	// Should NOT contain DATABASE_URL or REDIS_URL (not sensitive)
	if strings.Contains(content, "DATABASE_URL") {
		t.Error("env should not contain DATABASE_URL")
	}
	if strings.Contains(content, "REDIS_URL") {
		t.Error("env should not contain REDIS_URL")
	}

	// Check file permissions (0600)
	info, _ := os.Stat(filepath.Join(dir, config.EnvFileName()))
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected permission 0600, got %o", info.Mode().Perm())
	}
}

func TestWriteEnvConfigSQLite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	req := ValidateRequest{
		Database: DatabaseConfig{Type: "sqlite", Path: dbPath},
		Redis:    RedisConfig{Host: "localhost", Port: 6379},
		Auth:     AuthConfig{AdminEmail: "admin@test.com", AdminPassword: "password123"},
	}

	if err := writeEnvConfig(dir, req, "jwt-test-secret"); err != nil {
		t.Fatalf("writeEnvConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, config.EnvFileName()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	content := string(data)
	// SQLite: no DB_PASSWORD, no DATABASE_URL
	if strings.Contains(content, "DB_PASSWORD") {
		t.Error("sqlite env should not contain DB_PASSWORD")
	}
	if strings.Contains(content, "DATABASE_URL") {
		t.Error("sqlite env should not contain DATABASE_URL")
	}
	// Should contain auth secrets
	if !strings.Contains(content, "jwt-test-secret") {
		t.Error("env should contain JWT_SECRET")
	}
}

func TestTestSQLite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "subdir", "test.db")

	// Should succeed: writable directory
	result := testSQLite(DatabaseConfig{Type: "sqlite", Path: dbPath}, dir)
	if !result.OK {
		t.Errorf("testSQLite should succeed for writable path, got: %s", result.Message)
	}

	// Should fail: unwritable path
	result2 := testSQLite(DatabaseConfig{Type: "sqlite", Path: "/proc/nonexistent/test.db"}, dir)
	if result2.OK {
		t.Error("testSQLite should fail for unwritable path")
	}
}

func TestHandleValidateSQLite(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{configDir: dir, token: "test"}
	dbPath := filepath.Join(dir, "test.db")
	body := `{"database":{"type":"sqlite","path":"` + dbPath + `"},"redis":{"host":"127.0.0.1","port":1},"auth":{"admin_email":"a@b.com","admin_password":"12345678"}}`
	req := httptest.NewRequest(http.MethodPost, "/setup/api/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleValidate(w, req)

	var resp ValidateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Checks["database"].OK {
		t.Errorf("SQLite check should pass for writable dir, got: %s", resp.Checks["database"].Message)
	}
}

func TestIsUnderSystemd(t *testing.T) {
	original := os.Getenv("INVOCATION_ID")
	defer os.Setenv("INVOCATION_ID", original)

	os.Unsetenv("INVOCATION_ID")
	if os.Getppid() != 1 && isUnderSystemd() {
		t.Error("should not detect systemd in test")
	}

	os.Setenv("INVOCATION_ID", "test")
	if !isUnderSystemd() {
		t.Error("should detect systemd when INVOCATION_ID set")
	}
}

func TestGetLocalIPs(t *testing.T) {
	ips := getLocalIPs()
	t.Logf("Local IPs: %v", ips)
}

func TestWriteYAMLConfigMongoDB(t *testing.T) {
	dir := t.TempDir()
	req := ValidateRequest{
		Database: DatabaseConfig{Type: "mongodb", Host: "mongo.local", Port: 27017, User: "admin", Password: "secret", DBName: "mydb"},
		Redis:    RedisConfig{Host: "redis.local", Port: 6379, Password: "redispass"},
		TLS:      TLSConfig{Enabled: true, Mode: "auto_generate"},
		Auth:     AuthConfig{AdminEmail: "admin@test.com", AdminPassword: "password123"},
		Server:   ServerConfig{Port: "8080"},
	}

	if err := writeYAMLConfig(dir, req); err != nil {
		t.Fatalf("writeYAMLConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, config.ConfigFileName()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	content := string(data)
	for _, want := range []string{"driver: mongodb", "mongo.local", "27017", "mydb", "redis.local", "6379"} {
		if !strings.Contains(content, want) {
			t.Errorf("yaml should contain %q", want)
		}
	}
	// Password should NOT be in yaml (goes to .env)
	if strings.Contains(content, "secret") {
		t.Error("yaml should not contain password")
	}
	// MongoDB yaml should NOT contain sslmode
	if strings.Contains(content, "sslmode") {
		t.Error("mongodb yaml should not contain sslmode")
	}
}

func TestWriteEnvConfigMongoDB(t *testing.T) {
	dir := t.TempDir()
	req := ValidateRequest{
		Database: DatabaseConfig{Type: "mongodb", Host: "mongo.local", Port: 27017, User: "admin", Password: "mongosecret", DBName: "mydb"},
		Redis:    RedisConfig{Host: "redis.local", Port: 6379, Password: "redispass"},
		Auth:     AuthConfig{AdminEmail: "admin@test.com", AdminPassword: "password123"},
	}

	if err := writeEnvConfig(dir, req, "jwt-test-secret"); err != nil {
		t.Fatalf("writeEnvConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, config.EnvFileName()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	content := string(data)
	// MongoDB credentials
	for _, want := range []string{"MONGO_ROOT_USERNAME=admin", "MONGO_ROOT_PASSWORD=mongosecret", "REDIS_PASSWORD=redispass", "jwt-test-secret"} {
		if !strings.Contains(content, want) {
			t.Errorf("env should contain %q", want)
		}
	}
	// Should NOT contain DB_PASSWORD (that's for postgres)
	if strings.Contains(content, "DB_PASSWORD") {
		t.Error("mongodb env should not contain DB_PASSWORD")
	}
}

func TestTestMongoDB(t *testing.T) {
	// Should fail for unreachable host
	result := testMongoDB(DatabaseConfig{Type: "mongodb", Host: "127.0.0.1", Port: 1})
	if result.OK {
		t.Error("testMongoDB should fail for unreachable port")
	}

	// Should fail for empty host
	result2 := testMongoDB(DatabaseConfig{Type: "mongodb", Host: ""})
	if result2.OK {
		t.Error("testMongoDB should fail for empty host")
	}
}

func TestHandleGenerateInfra(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{configDir: dir, token: "test"}

	body := `{"mongo_port":27017,"redis_port":6379,"minio_api_port":9000,"minio_console_port":9001}`
	req := httptest.NewRequest(http.MethodPost, "/setup/api/generate-infra", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleGenerateInfra(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp InfraGenerateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success, got: %s", resp.Message)
	}
	if resp.MongoUser == "" || resp.MongoPassword == "" {
		t.Error("expected generated credentials")
	}
	if resp.RedisPassword == "" || resp.MinIOPassword == "" {
		t.Error("expected all passwords generated")
	}

	// Verify files exist
	infraDir := filepath.Join(dir, "infra")
	if _, err := os.Stat(filepath.Join(infraDir, "docker-compose.yml")); err != nil {
		t.Error("docker-compose.yml should exist")
	}
	if _, err := os.Stat(filepath.Join(infraDir, ".env")); err != nil {
		t.Error(".env should exist")
	}

	// Verify .env contains credentials
	envData, _ := os.ReadFile(filepath.Join(infraDir, ".env"))
	envContent := string(envData)
	if !strings.Contains(envContent, resp.MongoPassword) {
		t.Error(".env should contain generated mongo password")
	}
	if !strings.Contains(envContent, resp.RedisPassword) {
		t.Error(".env should contain generated redis password")
	}

	// Verify .env has restricted permissions
	info, _ := os.Stat(filepath.Join(infraDir, ".env"))
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected .env permission 0600, got %o", info.Mode().Perm())
	}
}

func TestHandleGenerateInfra_ReusesExistingCredentials(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{configDir: dir, token: "test"}

	// First call: generate fresh credentials
	body := `{"mongo_port":27017,"redis_port":6379,"minio_api_port":9000,"minio_console_port":9001}`
	req1 := httptest.NewRequest(http.MethodPost, "/setup/api/generate-infra", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	srv.handleGenerateInfra(w1, req1)

	var resp1 InfraGenerateResponse
	json.NewDecoder(w1.Body).Decode(&resp1)
	if !resp1.Success {
		t.Fatalf("first call failed: %s", resp1.Message)
	}

	// Second call: should reuse the same credentials
	req2 := httptest.NewRequest(http.MethodPost, "/setup/api/generate-infra", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.handleGenerateInfra(w2, req2)

	var resp2 InfraGenerateResponse
	json.NewDecoder(w2.Body).Decode(&resp2)
	if !resp2.Success {
		t.Fatalf("second call failed: %s", resp2.Message)
	}

	if resp1.MongoPassword != resp2.MongoPassword {
		t.Errorf("mongo password changed: %q -> %q", resp1.MongoPassword, resp2.MongoPassword)
	}
	if resp1.RedisPassword != resp2.RedisPassword {
		t.Errorf("redis password changed: %q -> %q", resp1.RedisPassword, resp2.RedisPassword)
	}
	if resp1.MinIOPassword != resp2.MinIOPassword {
		t.Errorf("minio password changed: %q -> %q", resp1.MinIOPassword, resp2.MinIOPassword)
	}
}

func TestHandleGenerateInfraMethodNotAllowed(t *testing.T) {
	srv := &Server{configDir: t.TempDir(), token: "test"}
	req := httptest.NewRequest(http.MethodGet, "/setup/api/generate-infra", nil)
	w := httptest.NewRecorder()
	srv.handleGenerateInfra(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleDeployInfraNotGenerated(t *testing.T) {
	srv := &Server{configDir: t.TempDir(), token: "test"}
	req := httptest.NewRequest(http.MethodPost, "/setup/api/deploy-infra", nil)
	w := httptest.NewRecorder()
	srv.handleDeployInfra(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleInfraStatusNotDeployed(t *testing.T) {
	srv := &Server{configDir: t.TempDir(), token: "test"}
	req := httptest.NewRequest(http.MethodGet, "/setup/api/infra-status", nil)
	w := httptest.NewRecorder()
	srv.handleInfraStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp InfraStatusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "not_deployed" {
		t.Errorf("expected not_deployed, got %s", resp.Status)
	}
}

func TestFindDockerCompose(t *testing.T) {
	// Just verify it doesn't panic
	result := findDockerCompose()
	t.Logf("findDockerCompose: %q", result)
}

func TestTokenAuthMiddleware(t *testing.T) {
	srv := &Server{token: "secret-token"}
	handler := srv.tokenAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// No token → 403
	req := httptest.NewRequest(http.MethodGet, "/setup/api/info", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	// Query token → 200
	req = httptest.NewRequest(http.MethodGet, "/setup/api/info?token=secret-token", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with query token, got %d", w.Code)
	}

	// Header token → 200
	req = httptest.NewRequest(http.MethodGet, "/setup/api/info", nil)
	req.Header.Set("Authorization", "token secret-token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with header token, got %d", w.Code)
	}
}
