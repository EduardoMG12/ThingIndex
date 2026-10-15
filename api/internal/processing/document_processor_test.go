package processing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/queue"
)

type processingRepoStub struct {
	document      *models.Document
	updatedStates []domain.DocumentProcessingState
	attempts      []models.DocumentProcessingAttempt
	replaceCalls  int
	replaceErr    error
	results       []domain.IndexingResult

	reapCutoffs []time.Time
	reapCount   int64
	reapErr     error

	requeueCutoffs []time.Time
	requeueErr     error
}

func (r *processingRepoStub) RequeueStuckQueuedDocuments(_ context.Context, olderThan time.Time) (int64, error) {
	r.requeueCutoffs = append(r.requeueCutoffs, olderThan)
	return 0, r.requeueErr
}

func (r *processingRepoStub) ReapStuckProcessingDocuments(_ context.Context, olderThan time.Time) (int64, error) {
	r.reapCutoffs = append(r.reapCutoffs, olderThan)
	if r.reapErr != nil {
		return 0, r.reapErr
	}
	return r.reapCount, nil
}

func (r *processingRepoStub) GetDocumentByIDForProcessing(context.Context, uint) (*models.Document, error) {
	if r.document == nil {
		return nil, domain.ErrDocumentNotFound
	}
	return r.document, nil
}

func (r *processingRepoStub) UpdateDocumentProcessingState(_ context.Context, _ uint, state domain.DocumentProcessingState) error {
	r.updatedStates = append(r.updatedStates, state)
	return nil
}

func (r *processingRepoStub) CreateProcessingAttempt(_ context.Context, attempt *models.DocumentProcessingAttempt) error {
	if attempt == nil {
		return nil
	}

	r.attempts = append(r.attempts, *attempt)
	return nil
}

func (r *processingRepoStub) ReplaceIndexingResult(context.Context, uint, domain.IndexingResult) error {
	r.replaceCalls++
	if r.replaceErr != nil {
		return r.replaceErr
	}
	return nil
}

type processingStorageStub struct {
	openBodyByPath map[string]string
	openErrByPath  map[string]error
	existsByPath   map[string]bool
}

func (s *processingStorageStub) UploadFile(context.Context, io.Reader, int64, string, string) (string, error) {
	panic("unexpected UploadFile call")
}

func (s *processingStorageStub) OpenFile(_ context.Context, objectPath string) (io.ReadCloser, error) {
	if err := s.openErrByPath[objectPath]; err != nil {
		return nil, err
	}
	if body, ok := s.openBodyByPath[objectPath]; ok {
		return io.NopCloser(strings.NewReader(body)), nil
	}
	return nil, errors.New("missing object")
}

func (s *processingStorageStub) DeleteFile(context.Context, string) error {
	panic("unexpected DeleteFile call")
}

func (s *processingStorageStub) FileExists(_ context.Context, objectPath string) (bool, error) {
	return s.existsByPath[objectPath], nil
}

type processingStorageResolverStub struct {
	storage domain.StorageService
	err     error
}

func (r *processingStorageResolverStub) ResolveStorage(context.Context, uint) (domain.StorageService, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.storage, nil
}

type processingIndexerStub struct {
	calls    int
	lastBody string
	result   domain.IndexingResult
	err      error
}

func (s *processingIndexerStub) Index(_ context.Context, _ *models.Document, file io.Reader) (domain.IndexingResult, error) {
	s.calls++
	body, readErr := io.ReadAll(file)
	if readErr != nil {
		return domain.IndexingResult{}, readErr
	}
	s.lastBody = string(body)
	if s.err != nil {
		return domain.IndexingResult{}, s.err
	}
	return s.result, nil
}

func TestMarkRetryScheduledPersistsQueuedOperationalState(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{
		document: &models.Document{
			ID:                  7,
			Status:              domain.DocumentStatusFailed,
			ProcessingStage:     "FAILED_PARSING_STRUCTURE",
			LastProcessingError: "parse pdf: broken xref table",
		},
	}
	processor := NewDocumentProcessor(repo, nil, nil, nil, nil, nil)

	processor.MarkRetryScheduled(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 7}, queue.DocumentIndexFailureDisposition{
		NextRetryAttempt: 1,
		MaxRetries:       3,
		RetryDelay:       30 * time.Second,
		Err:              fmt.Errorf("%w: parse pdf", domain.ErrDocumentIndexing),
	})

	if len(repo.updatedStates) != 1 {
		t.Fatalf("expected one state update, got %d", len(repo.updatedStates))
	}

	state := repo.updatedStates[0]
	if state.Status != domain.DocumentStatusQueued || state.Stage != "RETRY_SCHEDULED" {
		t.Fatalf("unexpected retry scheduled state: %#v", state)
	}
	if !strings.Contains(state.StatusDetail, "tentativa 1 de 3") {
		t.Fatalf("expected retry detail to mention attempt count, got %#v", state)
	}
	if state.LastError != "parse pdf: broken xref table" {
		t.Fatalf("expected original processing error to be preserved, got %#v", state)
	}
	if len(repo.attempts) != 1 {
		t.Fatalf("expected one processing attempt, got %d", len(repo.attempts))
	}
	if repo.attempts[0].FailureKind != "TRANSIENT" || repo.attempts[0].Disposition != "RETRY_SCHEDULED" {
		t.Fatalf("unexpected retry attempt metadata: %#v", repo.attempts[0])
	}
}

func TestMarkMovedToDLQPersistsFailedOperationalState(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{
		document: &models.Document{
			ID:                  8,
			Status:              domain.DocumentStatusFailed,
			ProcessingStage:     "FAILED_PERSISTING_INDEX",
			LastProcessingError: "database unavailable",
		},
	}
	processor := NewDocumentProcessor(repo, nil, nil, nil, nil, nil)

	processor.MarkMovedToDLQ(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 8}, queue.DocumentIndexFailureDisposition{
		RetryCount: 3,
		MaxRetries: 3,
		Err:        fmt.Errorf("%w: persist indexing result", domain.ErrDocumentIndexing),
	})

	if len(repo.updatedStates) != 1 {
		t.Fatalf("expected one state update, got %d", len(repo.updatedStates))
	}

	state := repo.updatedStates[0]
	if state.Status != domain.DocumentStatusFailed || state.Stage != "MOVED_TO_DLQ" {
		t.Fatalf("unexpected dlq state: %#v", state)
	}
	if !strings.Contains(state.StatusDetail, "3 retry(s)") {
		t.Fatalf("expected dlq detail to mention retry exhaustion, got %#v", state)
	}
	if len(repo.attempts) != 1 {
		t.Fatalf("expected one processing attempt, got %d", len(repo.attempts))
	}
	if repo.attempts[0].FailureKind != "TRANSIENT" || repo.attempts[0].Disposition != "MOVED_TO_DLQ" {
		t.Fatalf("unexpected dlq attempt metadata: %#v", repo.attempts[0])
	}
}

func TestMarkPermanentFailureKeepsOrphanedStateWhenSourceIsMissing(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{
		document: &models.Document{
			ID:                  9,
			Status:              domain.DocumentStatusOrphaned,
			ProcessingStage:     "SOURCE_MISSING",
			LastProcessingError: "source file missing in configured storage",
		},
	}
	processor := NewDocumentProcessor(repo, nil, nil, nil, nil, nil)

	processor.MarkPermanentFailure(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 9}, queue.DocumentIndexFailureDisposition{
		Err: fmt.Errorf("%w: source file missing", domain.ErrDocumentSourceMissing),
	})

	if len(repo.updatedStates) != 1 {
		t.Fatalf("expected one state update, got %d", len(repo.updatedStates))
	}

	state := repo.updatedStates[0]
	if state.Status != domain.DocumentStatusOrphaned || state.Stage != "SOURCE_MISSING" {
		t.Fatalf("unexpected permanent failure state: %#v", state)
	}
	if !strings.Contains(state.StatusDetail, "Nenhuma nova tentativa automatica") {
		t.Fatalf("expected permanent failure detail to mention no automatic retry, got %#v", state)
	}
	if len(repo.attempts) != 1 {
		t.Fatalf("expected one processing attempt, got %d", len(repo.attempts))
	}
	if repo.attempts[0].FailureKind != "SOURCE_MISSING" || repo.attempts[0].Disposition != "FAILED_PERMANENT" {
		t.Fatalf("unexpected permanent attempt metadata: %#v", repo.attempts[0])
	}
}

func TestProcessFailsWhenResolvingStorageReturnsUnexpectedError(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{
		document: &models.Document{
			ID:      11,
			UserID:  3,
			FileURL: "documents/failure.pdf",
		},
	}
	processor := NewDocumentProcessor(
		repo,
		&processingStorageStub{},
		&processingStorageResolverStub{err: errors.New("vault unavailable")},
		&processingIndexerStub{},
		nil,
		nil,
	)

	err := processor.Process(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 11, UserID: 3})
	if !errors.Is(err, domain.ErrDocumentIndexing) {
		t.Fatalf("expected indexing error when storage resolution fails, got %v", err)
	}

	lastState := repo.updatedStates[len(repo.updatedStates)-1]
	if lastState.Status != domain.DocumentStatusFailed || lastState.Stage != "FAILED_RESOLVING_STORAGE" {
		t.Fatalf("unexpected final state after storage resolution failure: %#v", lastState)
	}
	if !strings.Contains(lastState.LastError, "vault unavailable") {
		t.Fatalf("expected resolution detail to be preserved, got %#v", lastState)
	}
}

func TestProcessMarksDocumentOrphanedWhenSourceIsMissingInBothStorages(t *testing.T) {
	t.Parallel()

	baseStorage := &processingStorageStub{
		openErrByPath: map[string]error{
			"documents/orphaned.pdf": errors.New("base storage missing"),
		},
		existsByPath: map[string]bool{
			"documents/orphaned.pdf": false,
		},
	}
	customStorage := &processingStorageStub{
		openErrByPath: map[string]error{
			"documents/orphaned.pdf": errors.New("custom storage missing"),
		},
		existsByPath: map[string]bool{
			"documents/orphaned.pdf": false,
		},
	}
	repo := &processingRepoStub{
		document: &models.Document{
			ID:      12,
			UserID:  4,
			FileURL: "documents/orphaned.pdf",
		},
	}
	processor := NewDocumentProcessor(
		repo,
		baseStorage,
		&processingStorageResolverStub{storage: customStorage},
		&processingIndexerStub{},
		nil,
		nil,
	)

	err := processor.Process(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 12, UserID: 4})
	if !errors.Is(err, domain.ErrDocumentSourceMissing) {
		t.Fatalf("expected orphaned source classification, got %v", err)
	}

	lastState := repo.updatedStates[len(repo.updatedStates)-1]
	if lastState.Status != domain.DocumentStatusOrphaned || lastState.Stage != "SOURCE_MISSING" {
		t.Fatalf("unexpected final orphaned state: %#v", lastState)
	}
	if !strings.Contains(lastState.LastError, "source file missing") {
		t.Fatalf("expected missing source detail, got %#v", lastState)
	}
}

func TestProcessFallsBackToBaseStorageWhenCustomStorageReadFails(t *testing.T) {
	t.Parallel()

	baseStorage := &processingStorageStub{
		openBodyByPath: map[string]string{
			"documents/fallback.html": "<html>fallback body</html>",
		},
	}
	customStorage := &processingStorageStub{
		openErrByPath: map[string]error{
			"documents/fallback.html": errors.New("custom storage unavailable"),
		},
		existsByPath: map[string]bool{
			"documents/fallback.html": false,
		},
	}
	repo := &processingRepoStub{
		document: &models.Document{
			ID:      13,
			UserID:  5,
			FileURL: "documents/fallback.html",
		},
	}
	indexer := &processingIndexerStub{}
	processor := NewDocumentProcessor(
		repo,
		baseStorage,
		&processingStorageResolverStub{storage: customStorage},
		indexer,
		nil,
		nil,
	)

	if err := processor.Process(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 13, UserID: 5}); err != nil {
		t.Fatalf("expected fallback processing to succeed, got %v", err)
	}
	if indexer.calls != 1 {
		t.Fatalf("expected indexer to run once after fallback, got %d", indexer.calls)
	}
	if indexer.lastBody != "<html>fallback body</html>" {
		t.Fatalf("expected fallback storage body to reach the indexer, got %q", indexer.lastBody)
	}
	if repo.replaceCalls != 1 {
		t.Fatalf("expected indexing result to be persisted once, got %d", repo.replaceCalls)
	}
}

func TestProcessFailsWhenPersistingIndexingResult(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{
		document: &models.Document{
			ID:      14,
			UserID:  6,
			FileURL: "documents/persist.pdf",
		},
		replaceErr: errors.New("database unavailable"),
	}
	indexer := &processingIndexerStub{}
	processor := NewDocumentProcessor(
		repo,
		&processingStorageStub{
			openBodyByPath: map[string]string{
				"documents/persist.pdf": "%PDF-1.7 sample",
			},
		},
		&processingStorageResolverStub{err: domain.ErrStorageProviderNotConfigured},
		indexer,
		nil,
		nil,
	)

	err := processor.Process(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 14, UserID: 6})
	if !errors.Is(err, domain.ErrDocumentIndexing) {
		t.Fatalf("expected indexing error when persist step fails, got %v", err)
	}
	if indexer.calls != 1 {
		t.Fatalf("expected indexer to run before persistence failure, got %d", indexer.calls)
	}

	lastState := repo.updatedStates[len(repo.updatedStates)-1]
	if lastState.Status != domain.DocumentStatusFailed || lastState.Stage != "FAILED_PERSISTING_INDEX" {
		t.Fatalf("unexpected final persistence failure state: %#v", lastState)
	}
	if !strings.Contains(lastState.LastError, "database unavailable") {
		t.Fatalf("expected persistence failure detail, got %#v", lastState)
	}
}

func TestProcessFailsWhenIndexerReturnsParsingError(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{
		document: &models.Document{
			ID:      15,
			UserID:  7,
			FileURL: "documents/broken.pdf",
		},
	}
	indexer := &processingIndexerStub{
		err: fmt.Errorf("%w: corrupted cross-reference table", domain.ErrDocumentParsingFailed),
	}
	processor := NewDocumentProcessor(
		repo,
		&processingStorageStub{
			openBodyByPath: map[string]string{
				"documents/broken.pdf": "%PDF-1.7 broken",
			},
		},
		&processingStorageResolverStub{err: domain.ErrStorageProviderNotConfigured},
		indexer,
		nil,
		nil,
	)

	err := processor.Process(context.Background(), domain.DocumentIndexRequestedEvent{DocumentID: 15, UserID: 7})
	if !errors.Is(err, domain.ErrDocumentParsingFailed) {
		t.Fatalf("expected parser failure to bubble up, got %v", err)
	}

	lastState := repo.updatedStates[len(repo.updatedStates)-1]
	if lastState.Status != domain.DocumentStatusFailed || lastState.Stage != "FAILED_PARSING_STRUCTURE" {
		t.Fatalf("unexpected final parsing failure state: %#v", lastState)
	}
	if !strings.Contains(lastState.LastError, "corrupted cross-reference table") {
		t.Fatalf("expected parser failure detail, got %#v", lastState)
	}
}

// Idempotencia do consumidor: o RabbitMQ reentrega quando o worker morre
// antes do ack, e o outbox nao impede isso.

func TestProcessDescartaReentregaDeDocumentoJaIndexado(t *testing.T) {
	t.Parallel()

	requestedAt := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	indexedAt := requestedAt.Add(2 * time.Minute)

	repo := &processingRepoStub{
		document: &models.Document{
			ID:        21,
			UserID:    5,
			FileURL:   "documents/ja-indexado.html",
			Status:    domain.DocumentStatusIndexed,
			IndexedAt: &indexedAt,
		},
	}
	indexer := &processingIndexerStub{}
	storage := &processingStorageStub{
		openBodyByPath: map[string]string{"documents/ja-indexado.html": "<html>corpo</html>"},
	}
	processor := NewDocumentProcessor(repo, storage, &processingStorageResolverStub{err: domain.ErrStorageProviderNotConfigured}, indexer, nil, nil)

	err := processor.Process(context.Background(), domain.DocumentIndexRequestedEvent{
		DocumentID:  21,
		UserID:      5,
		RequestedAt: requestedAt,
	})
	if err != nil {
		t.Fatalf("reentrega deveria ser descartada em silencio, veio erro: %v", err)
	}

	if indexer.calls != 0 {
		t.Fatalf("o indexador nao deveria rodar numa reentrega, rodou %d vez(es)", indexer.calls)
	}
	if repo.replaceCalls != 0 {
		t.Fatalf("o indice nao deveria ser reescrito numa reentrega, foi %d vez(es)", repo.replaceCalls)
	}
	// O guard tem que rodar antes de marcar PROCESSING.
	if len(repo.updatedStates) != 0 {
		t.Fatalf("nenhum estado deveria ser escrito numa reentrega, foram %d: %+v", len(repo.updatedStates), repo.updatedStates)
	}
}

func TestProcessIndexaQuandoOPedidoEPosteriorAUltimaIndexacao(t *testing.T) {
	t.Parallel()

	// Reindexacao legitima: o usuario pediu de novo depois de o documento ter
	// sido indexado. Aqui o trabalho PRECISA acontecer.
	indexedAt := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	requestedAt := indexedAt.Add(1 * time.Hour)

	repo := &processingRepoStub{
		document: &models.Document{
			ID:        22,
			UserID:    5,
			FileURL:   "documents/reindexar.html",
			Status:    domain.DocumentStatusIndexed,
			IndexedAt: &indexedAt,
		},
	}
	indexer := &processingIndexerStub{}
	storage := &processingStorageStub{
		openBodyByPath: map[string]string{"documents/reindexar.html": "<html>corpo</html>"},
	}
	processor := NewDocumentProcessor(repo, storage, &processingStorageResolverStub{err: domain.ErrStorageProviderNotConfigured}, indexer, nil, nil)

	if err := processor.Process(context.Background(), domain.DocumentIndexRequestedEvent{
		DocumentID:  22,
		UserID:      5,
		RequestedAt: requestedAt,
	}); err != nil {
		t.Fatalf("reindexacao legitima deveria processar, veio erro: %v", err)
	}

	if indexer.calls != 1 {
		t.Fatalf("esperava 1 execucao do indexador, veio %d", indexer.calls)
	}
	if repo.replaceCalls != 1 {
		t.Fatalf("esperava o indice persistido 1 vez, veio %d", repo.replaceCalls)
	}
}

func TestIsAlreadyIndexedForEConservadorEmTodasAsGuardas(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	depois := base.Add(time.Minute)
	antes := base.Add(-time.Minute)

	casos := []struct {
		nome     string
		doc      *models.Document
		event    domain.DocumentIndexRequestedEvent
		esperado bool
		porque   string
	}{
		{
			nome:     "documento nulo",
			doc:      nil,
			event:    domain.DocumentIndexRequestedEvent{RequestedAt: base},
			esperado: false,
		},
		{
			nome:     "evento antigo sem carimbo processa",
			doc:      &models.Document{Status: domain.DocumentStatusIndexed, IndexedAt: &depois},
			event:    domain.DocumentIndexRequestedEvent{},
			esperado: false,
			porque:   "eventos gravados antes do campo existir nao podem ser descartados por ausencia de informacao",
		},
		{
			nome:     "indexed_at zerado pela reindexacao processa",
			doc:      &models.Document{Status: domain.DocumentStatusIndexed, IndexedAt: nil},
			event:    domain.DocumentIndexRequestedEvent{RequestedAt: base},
			esperado: false,
			porque:   "ReindexDocument zera indexed_at justamente para pedir trabalho novo",
		},
		{
			nome:     "documento FAILED processa mesmo com indexed_at futuro",
			doc:      &models.Document{Status: domain.DocumentStatusFailed, IndexedAt: &depois},
			event:    domain.DocumentIndexRequestedEvent{RequestedAt: base},
			esperado: false,
			porque:   "uma tentativa que falhou depois precisa rodar de novo",
		},
		{
			nome:     "indexacao anterior ao pedido processa",
			doc:      &models.Document{Status: domain.DocumentStatusIndexed, IndexedAt: &antes},
			event:    domain.DocumentIndexRequestedEvent{RequestedAt: base},
			esperado: false,
			porque:   "o indice e mais velho que o pedido, entao o pedido ainda nao foi atendido",
		},
		{
			nome:     "indexacao posterior ao pedido descarta",
			doc:      &models.Document{Status: domain.DocumentStatusIndexed, IndexedAt: &depois},
			event:    domain.DocumentIndexRequestedEvent{RequestedAt: base},
			esperado: true,
		},
		{
			nome:     "indexacao no mesmo instante descarta",
			doc:      &models.Document{Status: domain.DocumentStatusIndexed, IndexedAt: &base},
			event:    domain.DocumentIndexRequestedEvent{RequestedAt: base},
			esperado: true,
			porque:   "igualdade conta como atendido; o oposto reprocessaria o evento eternamente",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if got := isAlreadyIndexedFor(caso.doc, caso.event); got != caso.esperado {
				t.Fatalf("isAlreadyIndexedFor = %v, esperava %v. %s", got, caso.esperado, caso.porque)
			}
		})
	}
}
