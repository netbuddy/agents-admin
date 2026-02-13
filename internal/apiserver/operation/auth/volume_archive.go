package auth

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

// UploadVolumeArchive 上传 Volume 归档到 MinIO
//
// PUT /api/v1/accounts/{id}/volume-archive
// Body: tar.gz 二进制流
func (h *Handler) UploadVolumeArchive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if h.minio == nil {
		writeError(w, http.StatusServiceUnavailable, "object storage not configured")
		return
	}

	// 验证账号存在
	account, err := h.store.GetAccount(ctx, id)
	if err != nil {
		log.Printf("[auth] GetAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get account")
		return
	}
	if account == nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	// 上传到 MinIO
	archiveKey := fmt.Sprintf("volumes/%s.tar.gz", id)
	if err := h.minio.Upload(ctx, archiveKey, r.Body, r.ContentLength, "application/gzip"); err != nil {
		log.Printf("[auth] Upload volume archive error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to upload volume archive")
		return
	}

	// 更新账号记录
	if err := h.store.UpdateAccountVolumeArchive(ctx, id, archiveKey); err != nil {
		log.Printf("[auth] UpdateAccountVolumeArchive error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update account")
		return
	}

	log.Printf("[auth] Volume archive uploaded: %s -> %s", id, archiveKey)
	writeJSON(w, http.StatusOK, map[string]string{
		"archive_key": archiveKey,
	})
}

// DownloadVolumeArchive 从 MinIO 下载 Volume 归档
//
// GET /api/v1/accounts/{id}/volume-archive
func (h *Handler) DownloadVolumeArchive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if h.minio == nil {
		writeError(w, http.StatusServiceUnavailable, "object storage not configured")
		return
	}

	// 验证账号存在且有归档
	account, err := h.store.GetAccount(ctx, id)
	if err != nil {
		log.Printf("[auth] GetAccount error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get account")
		return
	}
	if account == nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if account.VolumeArchiveKey == nil || *account.VolumeArchiveKey == "" {
		writeError(w, http.StatusNotFound, "no volume archive available")
		return
	}

	// 从 MinIO 下载
	reader, err := h.minio.Download(ctx, *account.VolumeArchiveKey)
	if err != nil {
		log.Printf("[auth] Download volume archive error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to download volume archive")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.tar.gz", id))
	if _, err := io.Copy(w, reader); err != nil {
		log.Printf("[auth] Stream volume archive error: %v", err)
	}
}
