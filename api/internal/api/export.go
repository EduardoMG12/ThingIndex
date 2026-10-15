package api

import "thing-index/internal/domain"

// ResolveDefaultStorageForIndexer aplica ao indexer o mesmo retry de storage da API.
func ResolveDefaultStorageForIndexer(build func() (domain.StorageService, error)) domain.StorageService {
	return resolveDefaultStorage(build)
}
