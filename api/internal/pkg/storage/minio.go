package storage

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client     *minio.Client
	bucketName string
}

func NewMinioStorage(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*MinioStorage, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	ctx := context.Background()
	exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
	if errBucketExists != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", errBucketExists)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: "us-east-1"})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
		}
		log.Printf("Successfully created bucket %s\n", bucketName)
	}

	return &MinioStorage{
		client:     minioClient,
		bucketName: bucketName,
	}, nil
}

func (m *MinioStorage) UploadFile(ctx context.Context, file io.Reader, objectSize int64, contentType string, objectName string) (string, error) {
	
	info, err := m.client.PutObject(ctx, m.bucketName, objectName, file, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to minio: %w", err)
	}

	fileUrl := fmt.Sprintf("%s/%s", m.bucketName, info.Key)
	return fileUrl, nil
}
