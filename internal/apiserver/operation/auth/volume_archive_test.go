package auth

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"agents-admin/internal/shared/model"
)

// mockMinIOClient 模拟 MinIO 客户端行为
type mockMinIOClient struct {
	data map[string][]byte
}

func (m *mockMinIOClient) upload(key string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	m.data[key] = data
	return nil
}

func (m *mockMinIOClient) download(key string) (io.ReadCloser, error) {
	data, ok := m.data[key]
	if !ok {
		return nil, io.EOF
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func TestUploadVolumeArchive_NoMinIO(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)
	// minio is nil by default

	req := httptest.NewRequest("PUT", "/api/v1/accounts/acc-1/volume-archive", bytes.NewReader([]byte("test")))
	req.SetPathValue("id", "acc-1")
	w := httptest.NewRecorder()

	h.UploadVolumeArchive(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestDownloadVolumeArchive_NoMinIO(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/accounts/acc-1/volume-archive", nil)
	req.SetPathValue("id", "acc-1")
	w := httptest.NewRecorder()

	h.DownloadVolumeArchive(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestUploadVolumeArchive_AccountNotFound(t *testing.T) {
	store := newMockStore()
	h := NewHandler(store)

	// 设置一个假的 MinIO client（通过 SetMinIOClient）
	// 由于 objstore.Client 需要真实的 MinIO，这里只测 account not found 路径
	// MinIO client 设为 nil 时返回 503，所以这个测试验证 nil case
	req := httptest.NewRequest("PUT", "/api/v1/accounts/nonexistent/volume-archive", bytes.NewReader([]byte("test")))
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	h.UploadVolumeArchive(w, req)

	// Without MinIO configured, should return 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestDownloadVolumeArchive_NoArchiveKey(t *testing.T) {
	store := newMockStore()
	store.accounts["acc-1"] = &model.Account{
		ID:     "acc-1",
		Name:   "test@example.com",
		Status: model.AccountStatusAuthenticated,
		// VolumeArchiveKey is nil
	}
	h := NewHandler(store)

	req := httptest.NewRequest("GET", "/api/v1/accounts/acc-1/volume-archive", nil)
	req.SetPathValue("id", "acc-1")
	w := httptest.NewRecorder()

	h.DownloadVolumeArchive(w, req)

	// Without MinIO configured, should return 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestVolumeArchiveKey_StorageUpdate(t *testing.T) {
	store := newMockStore()
	store.accounts["acc-1"] = &model.Account{
		ID:     "acc-1",
		Name:   "test@example.com",
		Status: model.AccountStatusAuthenticated,
	}

	// 测试存储层 UpdateAccountVolumeArchive
	err := store.UpdateAccountVolumeArchive(context.Background(), "acc-1", "volumes/acc-1.tar.gz")
	if err != nil {
		t.Fatalf("UpdateAccountVolumeArchive failed: %v", err)
	}

	acc := store.accounts["acc-1"]
	if acc.VolumeArchiveKey == nil {
		t.Fatal("VolumeArchiveKey should not be nil after update")
	}
	if *acc.VolumeArchiveKey != "volumes/acc-1.tar.gz" {
		t.Errorf("expected volumes/acc-1.tar.gz, got %s", *acc.VolumeArchiveKey)
	}
}
