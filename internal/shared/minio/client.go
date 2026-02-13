// Package objstore 封装 MinIO 对象存储客户端
package objstore

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"agents-admin/internal/config"
)

// Client MinIO 客户端封装
type Client struct {
	mc     *minio.Client
	bucket string
}

// NewClient 创建 MinIO 客户端
func NewClient(cfg config.MinIOConfig) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("minio endpoint is required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("minio access_key and secret_key are required")
	}

	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	bucket := cfg.Bucket
	if bucket == "" {
		bucket = "agents-admin"
	}

	return &Client{mc: mc, bucket: bucket}, nil
}

// EnsureBucket 确保 bucket 存在
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
		log.Printf("[minio] Created bucket: %s", c.bucket)
	}
	return nil
}

// Upload 上传对象
func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.mc.PutObject(ctx, c.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	return nil
}

// Download 下载对象，调用方负责关闭返回的 ReadCloser
func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", key, err)
	}
	// 验证对象存在（GetObject 不会立即返回错误）
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		return nil, fmt.Errorf("stat %s: %w", key, err)
	}
	return obj, nil
}

// Exists 检查对象是否存在
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.mc.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Delete 删除对象
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}
