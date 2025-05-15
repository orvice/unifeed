package dao

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.orx.me/apps/unifeed/internal/conf"
)

type S3Client struct {
	client     *minio.Client
	bucketName string
}

// NewS3Client 创建一个新的 S3 客户端实例
func NewS3Client() (*S3Client, error) {

	config := conf.Conf.S3
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		// log
		slog.Error("failed to create S3 client",
			"endpoint", config.Endpoint,
			"use_ssl", config.UseSSL,
			"bucket_name", config.BucketName,
			"error", err)
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	s3Client := &S3Client{
		client:     client,
		bucketName: config.BucketName,
	}

	// 确保 bucket 存在
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, config.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, config.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return s3Client, nil
}

// PutObject 上传对象到 S3
func (s *S3Client) PutObject(ctx context.Context, objectName string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucketName, objectName, io.Reader(bytes.NewReader(data)), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

// GetObject 从 S3 获取对象
func (s *S3Client) GetObject(ctx context.Context, objectName string) (io.Reader, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return obj, nil
}

// ListObjects 列出指定前缀的对象
func (s *S3Client) ListObjects(ctx context.Context, prefix string) ([]minio.ObjectInfo, error) {
	var objects []minio.ObjectInfo
	objectCh := s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		objects = append(objects, object)
	}

	return objects, nil
}

// RemoveObject 删除对象
func (s *S3Client) RemoveObject(ctx context.Context, objectName string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object: %w", err)
	}
	return nil
}

// GetPresignedURL 获取预签名 URL
func (s *S3Client) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return url.String(), nil
}
