package api

import "thing-index/internal/domain"

func NewUnavailableStorageForIndexer(cause error) domain.StorageService {
	return newUnavailableStorage(cause)
}
