package api

import "thing-index/internal/domain"

// ResolveDefaultStorageForIndexer aplica ao indexer a mesma politica da API:
// se o storage padrao falhar no boot, tenta de novo sob demanda em vez de
// congelar o erro ate o processo reiniciar.
func ResolveDefaultStorageForIndexer(build func() (domain.StorageService, error)) domain.StorageService {
	return resolveDefaultStorage(build)
}
