//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const testVisibilityTimeout = 2 * time.Minute

func TestOutboxRepositoryClaimPendingEventsFiltersAndIncrementsAttempts(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormOutboxRepository{db: db}
	ctx := context.Background()
	now := time.Now().UTC()

	eligiblePending := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   10,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload:       map[string]any{"document_id": 10},
		Attempts:      0,
		AvailableAt:   now.Add(-2 * time.Minute),
	}
	eligibleFailed := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   11,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusFailed,
		Payload:       map[string]any{"document_id": 11},
		Attempts:      2,
		AvailableAt:   now.Add(-time.Minute),
	}
	futurePending := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   12,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload:       map[string]any{"document_id": 12},
		Attempts:      0,
		AvailableAt:   now.Add(time.Minute),
	}

	mustCreateRecord(t, db, &eligiblePending)
	mustCreateRecord(t, db, &eligibleFailed)
	mustCreateRecord(t, db, &futurePending)

	claimed, err := repo.ClaimPendingEvents(ctx, 10, now, testVisibilityTimeout)
	if err != nil {
		t.Fatalf("ClaimPendingEvents returned error: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("expected 2 claimed events, got %d", len(claimed))
	}
	if claimed[0].ID != eligiblePending.ID || claimed[1].ID != eligibleFailed.ID {
		t.Fatalf("expected claims in available_at order, got IDs %d and %d", claimed[0].ID, claimed[1].ID)
	}

	var storedPending models.OutboxEvent
	if err := db.First(&storedPending, eligiblePending.ID).Error; err != nil {
		t.Fatalf("load claimed pending event: %v", err)
	}
	if storedPending.Attempts != 1 {
		t.Fatalf("expected pending attempts 1, got %d", storedPending.Attempts)
	}

	var storedFailed models.OutboxEvent
	if err := db.First(&storedFailed, eligibleFailed.ID).Error; err != nil {
		t.Fatalf("load claimed failed event: %v", err)
	}
	if storedFailed.Attempts != 3 {
		t.Fatalf("expected failed attempts 3, got %d", storedFailed.Attempts)
	}

	var storedFuture models.OutboxEvent
	if err := db.First(&storedFuture, futurePending.ID).Error; err != nil {
		t.Fatalf("load future pending event: %v", err)
	}
	if storedFuture.Attempts != 0 {
		t.Fatalf("expected future event attempts to remain 0, got %d", storedFuture.Attempts)
	}
	if storedFuture.Status != domain.OutboxEventStatusPending {
		t.Fatalf("expected future event to stay PENDING, got %q", storedFuture.Status)
	}
}

// O bug: ClaimPendingEvents so incrementava `attempts`. O status seguia PENDING
// e available_at nao avancava, entao o mesmo evento era reivindicado de novo no
// poll seguinte — publicado e indexado 2x, com custo de LLM dobrado.
//
// O `FOR UPDATE SKIP LOCKED` nao protegia: o lock morre no commit da transacao,
// e a publicacao no broker acontece depois, ja fora dela.
func TestOutboxRepositoryClaimMarksEventsInFlightAndHidesThem(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormOutboxRepository{db: db}
	ctx := context.Background()
	now := time.Now().UTC()

	event := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   20,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload:       map[string]any{"document_id": 20},
		Attempts:      0,
		AvailableAt:   now.Add(-time.Minute),
	}
	mustCreateRecord(t, db, &event)

	claimed, err := repo.ClaimPendingEvents(ctx, 10, now, testVisibilityTimeout)
	if err != nil {
		t.Fatalf("primeiro claim retornou erro: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("esperava 1 evento reivindicado, veio %d", len(claimed))
	}

	// O evento devolvido tem de refletir o que foi gravado: o dispatcher usa
	// Attempts para calcular o backoff.
	if claimed[0].Status != domain.OutboxEventStatusInFlight {
		t.Fatalf("evento devolvido deveria vir como IN_FLIGHT, veio %q", claimed[0].Status)
	}
	if claimed[0].Attempts != 1 {
		t.Fatalf("evento devolvido deveria ter attempts=1, veio %d", claimed[0].Attempts)
	}

	var stored models.OutboxEvent
	if err := db.First(&stored, event.ID).Error; err != nil {
		t.Fatalf("recarregar evento: %v", err)
	}
	if stored.Status != domain.OutboxEventStatusInFlight {
		t.Fatalf("evento persistido deveria estar IN_FLIGHT, esta %q", stored.Status)
	}
	if !stored.AvailableAt.After(now) {
		t.Fatalf("available_at deveria ter avancado para o futuro, esta %s (now=%s)", stored.AvailableAt, now)
	}

	// O ponto central: um segundo dispatcher, no mesmo instante, NAO pode
	// reivindicar o mesmo evento.
	second, err := repo.ClaimPendingEvents(ctx, 10, now, testVisibilityTimeout)
	if err != nil {
		t.Fatalf("segundo claim retornou erro: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("evento em voo foi reivindicado de novo: publicacao e indexacao dupla (%d eventos)", len(second))
	}
}

// Rede de seguranca: se o worker morrer entre reivindicar e publicar, ninguem
// marca o evento como PUBLISHED nem FAILED. Vencido o visibility timeout, ele
// precisa voltar a ficar elegivel sozinho — senao o documento nunca indexa.
func TestOutboxRepositoryReclaimsExpiredInFlightEvents(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormOutboxRepository{db: db}
	ctx := context.Background()
	now := time.Now().UTC()

	abandoned := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   30,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusInFlight,
		Payload:       map[string]any{"document_id": 30},
		Attempts:      1,
		// Visibilidade ja vencida: o worker anterior morreu.
		AvailableAt: now.Add(-time.Minute),
	}
	stillInFlight := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   31,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusInFlight,
		Payload:       map[string]any{"document_id": 31},
		Attempts:      1,
		// Ainda dentro da janela: outro worker esta cuidando dele.
		AvailableAt: now.Add(time.Minute),
	}

	mustCreateRecord(t, db, &abandoned)
	mustCreateRecord(t, db, &stillInFlight)

	claimed, err := repo.ClaimPendingEvents(ctx, 10, now, testVisibilityTimeout)
	if err != nil {
		t.Fatalf("claim retornou erro: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("esperava recuperar so o evento abandonado, veio %d", len(claimed))
	}
	if claimed[0].ID != abandoned.ID {
		t.Fatalf("recuperou o evento errado: esperava %d, veio %d", abandoned.ID, claimed[0].ID)
	}
	if claimed[0].Attempts != 2 {
		t.Fatalf("attempts deveria ter ido para 2 na recuperacao, veio %d", claimed[0].Attempts)
	}

	var stored models.OutboxEvent
	if err := db.First(&stored, stillInFlight.ID).Error; err != nil {
		t.Fatalf("recarregar evento em voo: %v", err)
	}
	if stored.Attempts != 1 {
		t.Fatalf("evento dentro da janela nao podia ser tocado, attempts=%d", stored.Attempts)
	}
}

// Evento ja publicado nunca mais pode ser reivindicado, mesmo com available_at
// no passado — senao o documento reindexaria a cada poll.
func TestOutboxRepositoryNeverReclaimsPublishedEvents(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormOutboxRepository{db: db}
	ctx := context.Background()
	now := time.Now().UTC()

	event := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   40,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload:       map[string]any{"document_id": 40},
		Attempts:      0,
		AvailableAt:   now.Add(-time.Minute),
	}
	mustCreateRecord(t, db, &event)

	if _, err := repo.ClaimPendingEvents(ctx, 10, now, testVisibilityTimeout); err != nil {
		t.Fatalf("claim retornou erro: %v", err)
	}
	if err := repo.MarkEventPublished(ctx, event.ID, now); err != nil {
		t.Fatalf("MarkEventPublished retornou erro: %v", err)
	}

	// Bem depois de qualquer visibility timeout plausivel.
	future := now.Add(24 * time.Hour)
	claimed, err := repo.ClaimPendingEvents(ctx, 10, future, testVisibilityTimeout)
	if err != nil {
		t.Fatalf("claim posterior retornou erro: %v", err)
	}
	if len(claimed) != 0 {
		t.Fatalf("evento PUBLISHED foi reivindicado de novo: reindexacao duplicada")
	}
}

// Um evento marcado FAILED volta a ser elegivel quando o backoff vence — e o
// contador de tentativas continua crescendo, para o backoff progredir.
func TestOutboxRepositoryReclaimsFailedEventAfterBackoff(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormOutboxRepository{db: db}
	ctx := context.Background()
	now := time.Now().UTC()

	event := models.OutboxEvent{
		AggregateType: domain.OutboxAggregateDocument,
		AggregateID:   50,
		EventType:     domain.OutboxEventDocumentIndexRequested,
		Status:        domain.OutboxEventStatusPending,
		Payload:       map[string]any{"document_id": 50},
		Attempts:      0,
		AvailableAt:   now.Add(-time.Minute),
	}
	mustCreateRecord(t, db, &event)

	if _, err := repo.ClaimPendingEvents(ctx, 10, now, testVisibilityTimeout); err != nil {
		t.Fatalf("claim retornou erro: %v", err)
	}

	backoffUntil := now.Add(30 * time.Second)
	if err := repo.MarkEventFailed(ctx, event.ID, "broker indisponivel", backoffUntil); err != nil {
		t.Fatalf("MarkEventFailed retornou erro: %v", err)
	}

	// Antes do backoff vencer: invisivel.
	early, err := repo.ClaimPendingEvents(ctx, 10, now.Add(10*time.Second), testVisibilityTimeout)
	if err != nil {
		t.Fatalf("claim antecipado retornou erro: %v", err)
	}
	if len(early) != 0 {
		t.Fatalf("evento em backoff foi reivindicado cedo demais")
	}

	// Depois: elegivel de novo, com attempts crescendo.
	late, err := repo.ClaimPendingEvents(ctx, 10, backoffUntil.Add(time.Second), testVisibilityTimeout)
	if err != nil {
		t.Fatalf("claim posterior retornou erro: %v", err)
	}
	if len(late) != 1 {
		t.Fatalf("evento deveria voltar apos o backoff, veio %d", len(late))
	}
	if late[0].Attempts != 2 {
		t.Fatalf("attempts deveria ser 2 na segunda tentativa, veio %d", late[0].Attempts)
	}
}
