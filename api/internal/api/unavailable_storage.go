package api

import (
	"context"
	"fmt"
	"io"

	"thing-index/internal/domain"
)

type unavailableStorage struct {
	cause error
}

func newUnavailableStorage(cause error) domain.StorageService {
	return unavailableStorage{cause: cause}
}

func (s unavailableStorage) UploadFile(context.Context, io.Reader, int64, string, string) (string, error) {
	return "", fmt.Errorf("default storage is unavailable: %w", s.cause)
}

func (s unavailableStorage) OpenFile(context.Context, string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("default storage is unavailable: %w", s.cause)
}

func (s unavailableStorage) DeleteFile(context.Context, string) error {
	return fmt.Errorf("default storage is unavailable: %w", s.cause)
}

func (s unavailableStorage) FileExists(context.Context, string) (bool, error) {
	return false, fmt.Errorf("default storage is unavailable: %w", s.cause)
}
