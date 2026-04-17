package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type minioProvider struct {
	client *minio.Client
	bucket string
}

// NewMinioProvider constructs a MinIO-backed StorageProvider.
func NewMinioProvider(endpoint, accessKey, secretKey, bucket string, useSSL bool) (StorageProvider, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return &minioProvider{client: client, bucket: bucket}, nil
}

func (p *minioProvider) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := p.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := p.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket %s: %w", bucket, err)
		}
	}
	return nil
}

func (p *minioProvider) Upload(ctx context.Context, input UploadInput) (string, error) {
	_, err := p.client.PutObject(ctx, p.bucket, input.Key, input.Reader, input.Size,
		minio.PutObjectOptions{ContentType: input.ContentType},
	)
	if err != nil {
		return "", fmt.Errorf("upload object %s: %w", input.Key, err)
	}
	// Return a presigned URL valid for 7 days
	url, err := p.client.PresignedGetObject(ctx, p.bucket, input.Key, 7*24*time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("presign url: %w", err)
	}
	return url.String(), nil
}

func (p *minioProvider) Delete(ctx context.Context, key string) error {
	err := p.client.RemoveObject(ctx, p.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("delete object %s: %w", key, err)
	}
	return nil
}

func (p *minioProvider) GetURL(ctx context.Context, key string) (string, error) {
	url, err := p.client.PresignedGetObject(ctx, p.bucket, key, 7*24*time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("get presigned url: %w", err)
	}
	return url.String(), nil
}
