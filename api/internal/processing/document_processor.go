package processing

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	enrichmentStatus := "not_configured"
	startedAt := time.Now().UTC()
	updateState := func(status, stage, detail, lastError string, completedAt *time.Time) {
		_ = p.repo.UpdateDocumentProcessingState(ctx, event.DocumentID, domain.DocumentProcessingState{
			Status:       status,
			Stage:        stage,
			StatusDetail: detail,
			LastError:    lastError,
			StartedAt:    &startedAt,
			CompletedAt:  completedAt,
		})
	}

	if err := p.repo.UpdateDocumentProcessingState(ctx, event.DocumentID, domain.DocumentProcessingState{
		Status:       domain.DocumentStatusProcessing,
		Stage:        "RESOLVING_DOCUMENT",
		StatusDetail: "Carregando documento e preparando pipeline de indexacao.",
		StartedAt:    &startedAt,
	}); err != nil {
		return fmt.Errorf("%w: update status to processing: %v", domain.ErrDocumentIndexing, err)
	}

	doc, err := p.repo.GetDocumentByIDForProcessing(ctx, event.DocumentID)
	if err != nil {
		completedAt := time.Now().UTC()
		updateState(domain.DocumentStatusFailed, "FAILED_LOADING_DOCUMENT", "Falha ao carregar o documento para indexacao.", err.Error(), &completedAt)
		return fmt.Errorf("%w: load document: %v", domain.ErrDocumentIndexing, err)
	}

	storageClient := p.baseStorage
	usedCustomStorage := false
	updateState(domain.DocumentStatusProcessing, "RESOLVING_STORAGE", "Resolvendo storage de origem para leitura do arquivo.", "", nil)
	userStorage, resolveErr := p.storageResolver.ResolveStorage(ctx, doc.UserID)
	switch {
	case resolveErr == nil:
		storageClient = userStorage
		usedCustomStorage = true
	case errors.Is(resolveErr, domain.ErrStorageProviderNotConfigured):
	default:
		completedAt := time.Now().UTC()
		updateState(domain.DocumentStatusFailed, "FAILED_RESOLVING_STORAGE", "Falha ao resolver o storage configurado para o documento.", resolveErr.Error(), &completedAt)
		return fmt.Errorf("%w: resolve storage: %v", domain.ErrDocumentIndexing, resolveErr)
	}

	updateState(domain.DocumentStatusProcessing, "READING_SOURCE", "Abrindo o arquivo bruto para extracao estrutural.", "", nil)
	reader, err := storageClient.OpenFile(ctx, doc.FileURL)
	if err != nil {
		if usedCustomStorage {
			fallbackReader, fallbackErr := p.baseStorage.OpenFile(ctx, doc.FileURL)
			if fallbackErr == nil {
				reader = fallbackReader
			} else {
				completedAt := time.Now().UTC()
				if shouldMarkAsOrphaned(ctx, doc.FileURL, storageClient, p.baseStorage) {
					updateState(domain.DocumentStatusOrphaned, "SOURCE_MISSING", "O arquivo original nao foi encontrado no storage configurado.", "source file missing in configured storage", &completedAt)
					return fmt.Errorf("%w: source file missing", domain.ErrDocumentSourceMissing)
				}
				updateState(domain.DocumentStatusFailed, "FAILED_READING_SOURCE", "Nao foi possivel abrir o arquivo nem no storage customizado nem no fallback.", fmt.Sprintf("custom storage: %v; fallback storage: %v", err, fallbackErr), &completedAt)
				return fmt.Errorf("%w: open file: custom storage: %v; fallback storage: %v", domain.ErrDocumentIndexing, err, fallbackErr)
			}
		} else {
			completedAt := time.Now().UTC()
			if shouldMarkAsOrphaned(ctx, doc.FileURL, storageClient, nil) {
				updateState(domain.DocumentStatusOrphaned, "SOURCE_MISSING", "O arquivo original nao foi encontrado no storage configurado.", "source file missing in configured storage", &completedAt)
				return fmt.Errorf("%w: source file missing", domain.ErrDocumentSourceMissing)
			}
			updateState(domain.DocumentStatusFailed, "FAILED_READING_SOURCE", "Nao foi possivel abrir o arquivo no storage configurado.", err.Error(), &completedAt)
			return fmt.Errorf("%w: open file: %v", domain.ErrDocumentIndexing, err)
		}
	}
	defer reader.Close()

	updateState(domain.DocumentStatusProcessing, "PARSING_STRUCTURE", "Extraindo paginas, secoes e estrutura base do documento.", "", nil)
	result, err := p.indexer.Index(ctx, doc, reader)
	if err != nil {
		completedAt := time.Now().UTC()
		updateState(domain.DocumentStatusFailed, "FAILED_PARSING_STRUCTURE", "Falha durante a extracao estrutural do documento.", err.Error(), &completedAt)
		return err
	}

	updateState(domain.DocumentStatusProcessing, "BUILDING_CHUNKS", "Montando representacoes auxiliares para retrieval e persistencia.", "", nil)
	result.Chunks = buildDocumentChunks(doc.ID, result)

	var observedLLM *observedLLMService
	if p.llmResolver != nil && p.enricher != nil {
		updateState(domain.DocumentStatusProcessing, "RESOLVING_LLM", "Resolvendo provider de IA para enriquecimento do indice.", "", nil)
		llmClient, resolveErr := p.llmResolver.ResolveLLMForIndexing(ctx, doc.UserID)
		switch {
		case resolveErr == nil:
			enrichmentStatus = "configured"
			observedLLM = newObservedLLMService(llmClient)
			updateState(domain.DocumentStatusProcessing, "ENRICHING_INDEX", "Executando construcao de arvore, resumos e sinais de retrieval com LLM.", "", nil)
			if enrichedResult, enrichErr := p.enricher.Enrich(ctx, observedLLM, doc, result); enrichErr == nil {
				result = enrichedResult
				enrichmentStatus = "applied"
			} else {
				enrichmentStatus = "failed"
			}
		case errors.Is(resolveErr, domain.ErrLLMProviderNotConfigured):
			enrichmentStatus = "not_configured"
		default:
			enrichmentStatus = "unavailable"
		}
	}

	updateState(domain.DocumentStatusProcessing, "FINALIZING_INDEX", "Consolidando metadados, telemetria e chunks finais.", "", nil)
	result.Chunks = buildDocumentChunks(doc.ID, result)
	appendPipelineMetadata(&result, doc.ID, enrichmentStatus)
	if observedLLM != nil {
		appendLLMUsageMetadata(&result, doc.ID, observedLLM.Snapshot())
	}
	completedAt := time.Now().UTC()
	appendProcessingTimingMetadata(&result, doc.ID, startedAt, completedAt)
	sanitizeIndexingResult(&result)

	updateState(domain.DocumentStatusProcessing, "PERSISTING_INDEX", "Persistindo paginas, arvore, metadados e monitoramento no banco.", "", nil)
	if err := p.repo.ReplaceIndexingResult(ctx, doc.ID, result); err != nil {
		failedAt := time.Now().UTC()
		updateState(domain.DocumentStatusFailed, "FAILED_PERSISTING_INDEX", "Falha ao persistir o indice estruturado do documento.", err.Error(), &failedAt)
		return fmt.Errorf("%w: persist indexing result: %v", domain.ErrDocumentIndexing, err)
	}

	return nil
}

func shouldMarkAsOrphaned(ctx context.Context, fileURL string, primary domain.StorageService, secondary domain.StorageService) bool {
	existsInPrimary, err := primary.FileExists(ctx, fileURL)
	if err != nil || existsInPrimary {
		return false
	}

	if secondary == nil {
		return true
	}

	existsInSecondary, secondaryErr := secondary.FileExists(ctx, fileURL)
	if secondaryErr != nil {
		return false
	}

	return !existsInSecondary
}
