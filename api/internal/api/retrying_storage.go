package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"thing-index/internal/domain"
)

const defaultStorageRetryInterval = 5 * time.Second

// retryingStorage e o storage padrao quando a construcao falhou no boot: sem ele,
// com o MinIO fora na subida, tudo dava 500 ate reiniciar a API. Tenta construir
// de novo no maximo uma vez por `interval`.
type retryingStorage struct {
	build    func() (domain.StorageService, error)
	interval time.Duration
	now      func() time.Time

	mu          sync.Mutex
	real        domain.StorageService
	lastErr     error
	lastAttempt time.Time
}

func newRetryingStorage(build func() (domain.StorageService, error), bootErr error, interval time.Duration, now func() time.Time) *retryingStorage {
	if interval <= 0 {
		interval = defaultStorageRetryInterval
	}
	if now == nil {
		now = time.Now
	}
	return &retryingStorage{
		build:       build,
		interval:    interval,
		now:         now,
		lastErr:     bootErr,
		lastAttempt: now(),
	}
}

// resolve constroi sob o mutex: chamadas concorrentes esperam a tentativa em curso.
func (s *retryingStorage) resolve() (domain.StorageService, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.real != nil {
		return s.real, nil
	}
	if s.now().Sub(s.lastAttempt) < s.interval {
		return nil, fmt.Errorf("default storage is unavailable: %w", s.lastErr)
	}

	s.lastAttempt = s.now()
	built, err := s.build()
	if err != nil {
		s.lastErr = err
		return nil, fmt.Errorf("default storage is unavailable: %w", err)
	}
	log.Printf("default storage provider initialized after boot failure")
	s.real = built
	s.lastErr = nil
	return built, nil
}

func (s *retryingStorage) UploadFile(ctx context.Context, file io.Reader, objectSize int64, contentType string, objectName string) (string, error) {
	real, err := s.resolve()
	if err != nil {
		return "", err
	}
	return real.UploadFile(ctx, file, objectSize, contentType, objectName)
}

func (s *retryingStorage) OpenFile(ctx context.Context, objectPath string) (io.ReadCloser, error) {
	real, err := s.resolve()
	if err != nil {
		return nil, err
	}
	return real.OpenFile(ctx, objectPath)
}

func (s *retryingStorage) DeleteFile(ctx context.Context, objectPath string) error {
	real, err := s.resolve()
	if err != nil {
		return err
	}
	return real.DeleteFile(ctx, objectPath)
}

func (s *retryingStorage) FileExists(ctx context.Context, objectPath string) (bool, error) {
	real, err := s.resolve()
	if err != nil {
		return false, err
	}
	return real.FileExists(ctx, objectPath)
}

func resolveDefaultStorage(build func() (domain.StorageService, error)) domain.StorageService {
	storage, err := build()
	if err == nil {
		return storage
	}
	log.Printf("warning: failed to initialize default storage provider, retrying on demand every %s: %v", defaultStorageRetryInterval, err)
	return newRetryingStorage(build, err, defaultStorageRetryInterval, time.Now)
}
