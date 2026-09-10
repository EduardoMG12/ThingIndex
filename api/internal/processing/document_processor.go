package processing

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/indexing"
	"thing-index/internal/models"
	"thing-index/internal/queue"
)

type DocumentProcessor struct {
	repo            domain.DocumentProcessingRepository
	baseStorage     domain.StorageService
	storageResolver domain.UserStorageResolver
	indexer         domain.DocumentIndexer
	llmResolver     domain.UserLLMResolver
	enricher        *indexing.LLMEnricher
	// preferences (opcional): o que a LLM gera para os documentos do usuario.
	preferences domain.IndexingPreferencesRepository
}

// WithIndexingPreferences liga o leitor de preferencias; sem ele, tudo ligado.
func (p *DocumentProcessor) WithIndexingPreferences(repo domain.IndexingPreferencesRepository) *DocumentProcessor {
	p.preferences = repo
	return p
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
		if err := p.repo.UpdateDocumentProcessingState(ctx, event.DocumentID, domain.DocumentProcessingState{
			Status:       status,
			Stage:        stage,
			StatusDetail: detail,
			LastError:    lastError,
			StartedAt:    &startedAt,
			CompletedAt:  completedAt,
		}); err != nil {
			// Antes este erro era descartado com `_ =`, inclusive nas
			// transicoes que marcam FAILED. O efeito era um documento
			// congelado em PROCESSING sem nenhum sintoma em log -- o pior
			// modo de falha possivel para diagnosticar.
			log.Printf("document processor: falha ao atualizar estado do documento %d para %s/%s: %v",
				event.DocumentID, status, stage, err)
		}
	}

	// A carga do documento vem ANTES de marcar PROCESSING de proposito. O
	// claim do outbox impede publicacao dupla, mas nao impede reentrega do
	// RabbitMQ (worker que morre entre processar e dar o ack). Marcar
	// PROCESSING primeiro sobrescreveria o estado INDEXED de um documento
	// pronto, e so entao descobririamos que era duplicata.
	doc, err := p.repo.GetDocumentByIDForProcessing(ctx, event.DocumentID)
	if err != nil {
		completedAt := time.Now().UTC()
		updateState(domain.DocumentStatusFailed, "FAILED_LOADING_DOCUMENT", "Falha ao carregar o documento para indexacao.", err.Error(), &completedAt)
		return fmt.Errorf("%w: load document: %v", domain.ErrDocumentIndexing, err)
	}

	if isAlreadyIndexedFor(doc, event) {
		log.Printf("document processor: descartando reentrega do documento %d (indexado em %s, evento pedido em %s)",
			event.DocumentID, doc.IndexedAt.UTC().Format(time.RFC3339), event.RequestedAt.UTC().Format(time.RFC3339))
		return nil
	}

	if err := p.repo.UpdateDocumentProcessingState(ctx, event.DocumentID, domain.DocumentProcessingState{
		Status:       domain.DocumentStatusProcessing,
		Stage:        "RESOLVING_DOCUMENT",
		StatusDetail: "Carregando documento e preparando pipeline de indexacao.",
		StartedAt:    &startedAt,
	}); err != nil {
		return fmt.Errorf("%w: update status to processing: %v", domain.ErrDocumentIndexing, err)
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
			prefs := domain.DefaultIndexingPreferences()
			if p.preferences != nil {
				if got, prefErr := p.preferences.GetIndexingPreferences(ctx, doc.UserID); prefErr == nil {
					prefs = got
				} else {
					log.Printf("indexing preferences for user %d unavailable, using defaults: %v", doc.UserID, prefErr)
				}
			}
			if enrichedResult, enrichErr := p.enricher.EnrichWith(ctx, observedLLM, doc, result, prefs); enrichErr == nil {
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

	// O enricher cai em fallback deterministico sempre que uma chamada ao LLM
	// falha, e devolve sucesso. Sem esta correcao o pipeline gravava
	// enrichment_status "applied" mesmo quando nenhuma requisicao completou —
	// o metadado afirmava enriquecimento por IA num indice 100% deterministico,
	// e a telemetria de custo e qualidade herdava a mentira.
	if enrichmentStatus == "applied" && observedLLM != nil {
		if usage := observedLLM.Snapshot(); usage.RequestCount == 0 {
			enrichmentStatus = "degraded"
			if usage.FailedRequestCount == 0 {
				enrichmentStatus = "skipped"
			}
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

// isAlreadyIndexedFor decide se o evento recebido ja foi atendido.
//
// A regra e uma so: o documento esta INDEXED e foi indexado DEPOIS de o evento
// ter sido pedido. Qualquer coisa fora disso processa normalmente.
//
// As tres guardas conservadoras importam mais que a regra:
//
//  1. RequestedAt zero -> processa. Eventos gravados antes desta mudanca nao
//     carregam o carimbo, e descartar por ausencia de informacao perderia
//     indexacao de verdade.
//  2. IndexedAt nil -> processa. E o caso da reindexacao manual, que zera o
//     campo justamente para pedir trabalho novo.
//  3. Status != INDEXED -> processa. Documento em FAILED ou ORPHANED precisa da
//     nova tentativa mesmo que tenha um indexed_at antigo.
//
// Na duvida o sistema reprocessa. Reprocessar custa uma rodada de LLM;
// descartar por engano deixa o documento sem indice e o usuario sem sintoma.
func isAlreadyIndexedFor(doc *models.Document, event domain.DocumentIndexRequestedEvent) bool {
	if doc == nil || doc.IndexedAt == nil {
		return false
	}
	if event.RequestedAt.IsZero() {
		return false
	}
	if doc.Status != domain.DocumentStatusIndexed {
		return false
	}

	return !doc.IndexedAt.Before(event.RequestedAt)
}

func (p *DocumentProcessor) MarkRetryScheduled(ctx context.Context, event domain.DocumentIndexRequestedEvent, disposition queue.DocumentIndexFailureDisposition) {
	doc, err := p.repo.GetDocumentByIDForProcessing(ctx, event.DocumentID)
	if err != nil {
		log.Printf("document processor: failed to load document %d for retry scheduling state update: %v", event.DocumentID, err)
		return
	}

	if err := p.repo.UpdateDocumentProcessingState(ctx, event.DocumentID, domain.DocumentProcessingState{
		Status:       domain.DocumentStatusQueued,
		Stage:        "RETRY_SCHEDULED",
		StatusDetail: buildRetryScheduledDetail(disposition.NextRetryAttempt, disposition.MaxRetries, disposition.RetryDelay),
		LastError:    coalesceProcessingError(doc.LastProcessingError, disposition.Err),
	}); err != nil {
		log.Printf("document processor: failed to persist retry scheduled state for document %d: %v", event.DocumentID, err)
	}

	if err := p.repo.CreateProcessingAttempt(ctx, buildProcessingAttemptRecord(doc, "RETRY_SCHEDULED", disposition)); err != nil {
		log.Printf("document processor: failed to persist retry attempt for document %d: %v", event.DocumentID, err)
	}
}

func (p *DocumentProcessor) MarkMovedToDLQ(ctx context.Context, event domain.DocumentIndexRequestedEvent, disposition queue.DocumentIndexFailureDisposition) {
	doc, err := p.repo.GetDocumentByIDForProcessing(ctx, event.DocumentID)
	if err != nil {
		log.Printf("document processor: failed to load document %d for dlq state update: %v", event.DocumentID, err)
		return
	}

	if err := p.repo.UpdateDocumentProcessingState(ctx, event.DocumentID, domain.DocumentProcessingState{
		Status:       domain.DocumentStatusFailed,
		Stage:        "MOVED_TO_DLQ",
		StatusDetail: buildMovedToDLQDetail(disposition.RetryCount),
		LastError:    coalesceProcessingError(doc.LastProcessingError, disposition.Err),
	}); err != nil {
		log.Printf("document processor: failed to persist dlq state for document %d: %v", event.DocumentID, err)
	}

	if err := p.repo.CreateProcessingAttempt(ctx, buildProcessingAttemptRecord(doc, "MOVED_TO_DLQ", disposition)); err != nil {
		log.Printf("document processor: failed to persist dlq attempt for document %d: %v", event.DocumentID, err)
	}
}

func (p *DocumentProcessor) MarkPermanentFailure(ctx context.Context, event domain.DocumentIndexRequestedEvent, disposition queue.DocumentIndexFailureDisposition) {
	doc, err := p.repo.GetDocumentByIDForProcessing(ctx, event.DocumentID)
	if err != nil {
		log.Printf("document processor: failed to load document %d for permanent failure update: %v", event.DocumentID, err)
		return
	}

	status := doc.Status
	stage := doc.ProcessingStage
	if status == "" || status == domain.DocumentStatusQueued || status == domain.DocumentStatusProcessing {
		status = domain.DocumentStatusFailed
	}
	if stage == "" {
		stage = "FAILED_PERMANENT"
	}

	if errors.Is(disposition.Err, domain.ErrDocumentSourceMissing) {
		status = domain.DocumentStatusOrphaned
		if stage == "" || stage == "FAILED_PERMANENT" {
			stage = "SOURCE_MISSING"
		}
	}

	if err := p.repo.UpdateDocumentProcessingState(ctx, event.DocumentID, domain.DocumentProcessingState{
		Status:       status,
		Stage:        stage,
		StatusDetail: buildPermanentFailureDetail(disposition.Err),
		LastError:    coalesceProcessingError(doc.LastProcessingError, disposition.Err),
	}); err != nil {
		log.Printf("document processor: failed to persist permanent failure state for document %d: %v", event.DocumentID, err)
	}

	if err := p.repo.CreateProcessingAttempt(ctx, buildProcessingAttemptRecord(doc, "FAILED_PERMANENT", disposition)); err != nil {
		log.Printf("document processor: failed to persist permanent attempt for document %d: %v", event.DocumentID, err)
	}
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

func buildRetryScheduledDetail(nextRetryAttempt, maxRetries int, retryDelay time.Duration) string {
	delayText := retryDelay.Round(time.Second).String()
	if delayText == "0s" {
		delayText = "instantes"
	}

	return fmt.Sprintf("A tentativa automatica falhou. Nova tentativa %d de %d agendada para %s.", nextRetryAttempt, maxRetries, delayText)
}

func buildMovedToDLQDetail(retryCount int) string {
	if retryCount <= 0 {
		return "A indexacao falhou e o evento foi movido para a DLQ. Agora e necessaria intervencao manual."
	}

	return fmt.Sprintf("As tentativas automaticas foram esgotadas apos %d retry(s). O evento foi movido para a DLQ e agora exige intervencao manual.", retryCount)
}

func buildPermanentFailureDetail(err error) string {
	if errors.Is(err, domain.ErrDocumentSourceMissing) {
		return "O arquivo original nao foi encontrado. Nenhuma nova tentativa automatica sera agendada ate que o original seja restaurado."
	}

	if errors.Is(err, domain.ErrDocumentParsingFailed) {
		return "A falha foi classificada como permanente durante a leitura do arquivo. Revise o documento original antes de reenfileirar."
	}

	return "A falha foi classificada como permanente e o documento nao sera reenfileirado automaticamente."
}

func coalesceProcessingError(current string, cause error) string {
	trimmed := strings.TrimSpace(current)
	if trimmed != "" {
		return trimmed
	}
	if cause == nil {
		return ""
	}
	return cause.Error()
}

func buildProcessingAttemptRecord(
	doc *models.Document,
	dispositionLabel string,
	disposition queue.DocumentIndexFailureDisposition,
) *models.DocumentProcessingAttempt {
	if doc == nil {
		return nil
	}

	var detail string
	switch dispositionLabel {
	case "RETRY_SCHEDULED":
		detail = buildRetryScheduledDetail(disposition.NextRetryAttempt, disposition.MaxRetries, disposition.RetryDelay)
	case "MOVED_TO_DLQ":
		detail = buildMovedToDLQDetail(disposition.RetryCount)
	default:
		detail = buildPermanentFailureDetail(disposition.Err)
	}

	return &models.DocumentProcessingAttempt{
		DocumentID:        doc.ID,
		UserID:            doc.UserID,
		Stage:             doc.ProcessingStage,
		FailureKind:       classifyProcessingFailureKind(dispositionLabel, disposition.Err),
		Disposition:       dispositionLabel,
		Detail:            detail,
		ErrorMessage:      coalesceProcessingError(doc.LastProcessingError, disposition.Err),
		RetryCount:        disposition.RetryCount,
		NextRetryAttempt:  disposition.NextRetryAttempt,
		MaxRetryAttempts:  disposition.MaxRetries,
		RetryDelaySeconds: int(disposition.RetryDelay / time.Second),
	}
}

func classifyProcessingFailureKind(dispositionLabel string, err error) string {
	if errors.Is(err, domain.ErrDocumentSourceMissing) {
		return "SOURCE_MISSING"
	}

	switch dispositionLabel {
	case "RETRY_SCHEDULED", "MOVED_TO_DLQ":
		return "TRANSIENT"
	default:
		return "PERMANENT"
	}
}
