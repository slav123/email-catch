package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/slav123/email-catch/internal/config"
)

type Backend interface {
	StoreLocal(path string, data []byte) error
	StoreS3(path string, data []byte) error
	StoreS3WithContentType(path string, data []byte, contentType string) error
	StoreS3WithOptions(path string, data []byte, contentType string, contentEncoding string) error
}

type StorageBackend struct {
	config      *config.Config
	minioClient *minio.Client
}

func NewStorageBackend(cfg *config.Config) (Backend, error) {
	backend := &StorageBackend{
		config: cfg,
	}

	if cfg.Storage.S3Compatible.Enabled {
		client, err := minio.New(cfg.Storage.S3Compatible.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.Storage.S3Compatible.AccessKey, cfg.Storage.S3Compatible.SecretKey, ""),
			Secure: cfg.Storage.S3Compatible.UseSSL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create minio client: %w", err)
		}

		backend.minioClient = client

		ctx := context.Background()
		exists, err := client.BucketExists(ctx, cfg.Storage.S3Compatible.Bucket)
		if err != nil {
			return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
		}

		if !exists {
			err = client.MakeBucket(ctx, cfg.Storage.S3Compatible.Bucket, minio.MakeBucketOptions{
				Region: cfg.Storage.S3Compatible.Region,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create bucket: %w", err)
			}
		}
	}

	if cfg.Storage.Local.Enabled {
		if err := os.MkdirAll(cfg.Storage.Local.Directory, 0755); err != nil {
			return nil, fmt.Errorf("failed to create local directory: %w", err)
		}
	}

	return backend, nil
}

func (b *StorageBackend) StoreLocal(path string, data []byte) error {
	if !b.config.Storage.Local.Enabled {
		return fmt.Errorf("local storage is not enabled")
	}

	fullPath := filepath.Join(b.config.Storage.Local.Directory, path)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

func (b *StorageBackend) StoreS3(path string, data []byte) error {
	return b.StoreS3WithContentType(path, data, "message/rfc822")
}

func (b *StorageBackend) StoreS3WithContentType(path string, data []byte, contentType string) error {
	return b.StoreS3WithOptions(path, data, contentType, "")
}

func (b *StorageBackend) StoreS3WithOptions(path string, data []byte, contentType string, contentEncoding string) error {
	if !b.config.Storage.S3Compatible.Enabled {
		return fmt.Errorf("S3 storage is not enabled")
	}

	if b.minioClient == nil {
		return fmt.Errorf("minio client is not initialized")
	}

	objectName := path
	if b.config.Storage.S3Compatible.PathPrefix != "" {
		objectName = fmt.Sprintf("%s/%s", b.config.Storage.S3Compatible.PathPrefix, path)
	}

	// Compress data if it's a compressible file type
	finalData := data
	finalContentEncoding := contentEncoding
	
	if b.shouldCompress(path, contentType) {
		compressed, err := b.compressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress data: %w", err)
		}
		finalData = compressed
		finalContentEncoding = "gzip"
	}

	reader := bytes.NewReader(finalData)

	options := minio.PutObjectOptions{
		ContentType: contentType,
	}
	
	if finalContentEncoding != "" {
		options.ContentEncoding = finalContentEncoding
	}

	_, err := b.minioClient.PutObject(
		context.Background(),
		b.config.Storage.S3Compatible.Bucket,
		objectName,
		reader,
		int64(len(finalData)),
		options,
	)

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// ShouldCompress determines if a file should be compressed based on its path and content type
func (b *StorageBackend) ShouldCompress(path string, contentType string) bool {
	// Compress .eml and .json files
	if strings.HasSuffix(path, ".eml") || strings.HasSuffix(path, ".json") {
		return true
	}
	
	// Also compress based on content type
	if contentType == "message/rfc822" || contentType == "application/json" {
		return true
	}
	
	return false
}

// CompressData compresses data using gzip
func (b *StorageBackend) CompressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	
	if _, err := gzipWriter.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write to gzip writer: %w", err)
	}
	
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	
	return buf.Bytes(), nil
}

// shouldCompress is a private method that calls the public ShouldCompress
func (b *StorageBackend) shouldCompress(path string, contentType string) bool {
	return b.ShouldCompress(path, contentType)
}

// compressData is a private method that calls the public CompressData
func (b *StorageBackend) compressData(data []byte) ([]byte, error) {
	return b.CompressData(data)
}

func (b *StorageBackend) ListS3Objects(prefix string) ([]string, error) {
	if !b.config.Storage.S3Compatible.Enabled {
		return nil, fmt.Errorf("S3 storage is not enabled")
	}

	if b.minioClient == nil {
		return nil, fmt.Errorf("minio client is not initialized")
	}

	objectPrefix := prefix
	if b.config.Storage.S3Compatible.PathPrefix != "" {
		objectPrefix = fmt.Sprintf("%s/%s", b.config.Storage.S3Compatible.PathPrefix, prefix)
	}

	ctx := context.Background()
	objects := b.minioClient.ListObjects(ctx, b.config.Storage.S3Compatible.Bucket, minio.ListObjectsOptions{
		Prefix:    objectPrefix,
		Recursive: true,
	})

	var objectNames []string
	for object := range objects {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		objectNames = append(objectNames, object.Key)
	}

	return objectNames, nil
}

func (b *StorageBackend) GetS3Object(objectName string) ([]byte, error) {
	if !b.config.Storage.S3Compatible.Enabled {
		return nil, fmt.Errorf("S3 storage is not enabled")
	}

	if b.minioClient == nil {
		return nil, fmt.Errorf("minio client is not initialized")
	}

	fullObjectName := objectName
	if b.config.Storage.S3Compatible.PathPrefix != "" {
		fullObjectName = fmt.Sprintf("%s/%s", b.config.Storage.S3Compatible.PathPrefix, objectName)
	}

	ctx := context.Background()
	object, err := b.minioClient.GetObject(ctx, b.config.Storage.S3Compatible.Bucket, fullObjectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	data := bytes.NewBuffer(nil)
	if _, err := data.ReadFrom(object); err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	return data.Bytes(), nil
}

func (b *StorageBackend) DeleteS3Object(objectName string) error {
	if !b.config.Storage.S3Compatible.Enabled {
		return fmt.Errorf("S3 storage is not enabled")
	}

	if b.minioClient == nil {
		return fmt.Errorf("minio client is not initialized")
	}

	fullObjectName := objectName
	if b.config.Storage.S3Compatible.PathPrefix != "" {
		fullObjectName = fmt.Sprintf("%s/%s", b.config.Storage.S3Compatible.PathPrefix, objectName)
	}

	ctx := context.Background()
	err := b.minioClient.RemoveObject(ctx, b.config.Storage.S3Compatible.Bucket, fullObjectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}