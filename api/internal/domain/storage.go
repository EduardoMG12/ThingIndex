package domain

import (
	"context"
	"io"
)

type StorageService interface {
	UploadFile(ctx context.Context, file io.Reader, objectSize int64, contentType string, objectName string) (string, error)
	OpenFile(ctx context.Context, objectPath string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, objectPath string) error
	FileExists(ctx context.Context, objectPath string) (bool, error)
}

type UserStorageResolver interface {
	ResolveStorage(ctx context.Context, userID uint) (StorageService, error)
}
