package processing

import (
	"context"
	"errors"
	"fmt"

	"thing-index/internal/domain"
	"thing-index/internal/indexing"
)

type DocumentProcessor struct {
	repo            domain.DocumentProcessingRepository
	baseStorage     domain.StorageService
	storageResolver domain.UserStorageResolver
	indexer         domain.DocumentIndexer
	llmResolver     domain.UserLLMResolver
	enricher        *indexing.LLMEnricher
}

func NewDocumentProcessor(
	repo domain.DocumentProcessingRepository,
	baseStorage domain.StorageService,
	storageResolver domain.UserStorageResolver,
	indexer domain.DocumentIndexer,
	llmResolver domain.UserLLMResolver,
	enricher *indexing.LLMEnricher,
) *DocumentProcessor {
	return &DocumentProcessor{
		repo:            repo,
		baseStorage:     baseStorage,
		storageResolver: storageResolver,
		indexer:         indexer,
		llmResolver:     llmResolver,
		enricher:        enricher,
	}
}

func (p *DocumentProcessor) Process(ctx context.Context, event domain.DocumentIndexRequestedEvent) error {
	if err := p.repo.UpdateDocumentStatus(ctx, event.DocumentID, domain.DocumentStatusProcessing); err != nil {
		return fmt.Errorf("%w: update status to processing: %v", domain.ErrDocumentIndexing, err)
	}

	doc, err := p.repo.GetDocumentByIDForProcessing(ctx, event.DocumentID)
	if err != nil {
		_ = p.repo.UpdateDocumentStatus(ctx, event.DocumentID, domain.DocumentStatusFailed)
		return fmt.Errorf("%w: load document: %v", domain.ErrDocumentIndexing, err)
	}

	storageClient := p.baseStorage
	userStorage, resolveErr := p.storageResolver.ResolveStorage(ctx, doc.UserID)
	switch {
	case resolveErr == nil:
		storageClient = userStorage
	case errors.Is(resolveErr, domain.ErrStorageProviderNotConfigured):
	default:
		_ = p.repo.UpdateDocumentStatus(ctx, doc.ID, domain.DocumentStatusFailed)
		return fmt.Errorf("%w: resolve storage: %v", domain.ErrDocumentIndexing, resolveErr)
	}

	reader, err := storageClient.OpenFile(ctx, doc.FileURL)
	if err != nil {
		_ = p.repo.UpdateDocumentStatus(ctx, doc.ID, domain.DocumentStatusFailed)
		return fmt.Errorf("%w: open file: %v", domain.ErrDocumentIndexing, err)
	}
	defer reader.Close()

	result, err := p.indexer.Index(ctx, doc, reader)
	if err != nil {
		_ = p.repo.UpdateDocumentStatus(ctx, doc.ID, domain.DocumentStatusFailed)
		return err
	}

	if p.llmResolver != nil && p.enricher != nil {
		llmClient, resolveErr := p.llmResolver.ResolveLLM(ctx, doc.UserID)
		switch {
		case resolveErr == nil:
			if enrichedResult, enrichErr := p.enricher.Enrich(ctx, llmClient, doc, result); enrichErr == nil {
				result = enrichedResult
			}
		case errors.Is(resolveErr, domain.ErrLLMProviderNotConfigured):
		default:
		}
	}

	if err := p.repo.ReplaceIndexingResult(ctx, doc.ID, result); err != nil {
		_ = p.repo.UpdateDocumentStatus(ctx, doc.ID, domain.DocumentStatusFailed)
		return fmt.Errorf("%w: persist indexing result: %v", domain.ErrDocumentIndexing, err)
	}

	return nil
}
