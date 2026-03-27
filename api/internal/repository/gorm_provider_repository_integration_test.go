//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestProviderRepositoryListProviderAuditEventsAppliesFilters(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormProviderRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "list-provider-audit")
	config := models.UserProviderConfig{
		UserID:             user.ID,
		Family:             domain.FamilyLLM,
		ProviderType:       domain.ProviderOpenAI,
		EncryptedData:      "ciphertext",
		EnabledForIndexing: true,
	}
	mustCreateRecord(t, db, &config)

	mustCreateRecord(t, db, &models.ProviderAuditEvent{
		UserID:           user.ID,
		ProviderConfigID: &config.ID,
		Family:           domain.FamilyLLM,
		ProviderType:     domain.ProviderOpenAI,
		ActorType:        "USER",
		Action:           "PROVIDER_CONFIG_SAVE",
		Status:           "SUCCESS",
		Detail:           "saved",
	})
	mustCreateRecord(t, db, &models.ProviderAuditEvent{
		UserID:           user.ID,
		ProviderConfigID: &config.ID,
		Family:           domain.FamilyLLM,
		ProviderType:     domain.ProviderOpenAI,
		ActorType:        "USER",
		Action:           "PROVIDER_ACTIVATION_CHANGE",
		Status:           "FAILED",
		Detail:           "toggle failed",
	})

	events, err := repo.ListProviderAuditEvents(ctx, user.ID, domain.ProviderAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			Limit:  10,
			Status: "success",
		},
		Family:       "llm",
		ProviderType: "openai",
		Action:       "provider_config_save",
	})
	if err != nil {
		t.Fatalf("ListProviderAuditEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 filtered provider event, got %d", len(events))
	}
	if events[0].Action != "PROVIDER_CONFIG_SAVE" || events[0].Status != "SUCCESS" {
		t.Fatalf("unexpected filtered provider event: %#v", events[0])
	}
}

func TestProviderRepositoryQueryProviderAuditEventsReturnsSummaryAndCursor(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &gormProviderRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "query-provider-audit")
	config := models.UserProviderConfig{
		UserID:             user.ID,
		Family:             domain.FamilyLLM,
		ProviderType:       domain.ProviderOpenAI,
		EncryptedData:      "ciphertext",
		EnabledForIndexing: true,
	}
	mustCreateRecord(t, db, &config)

	actorID := user.ID
	now := time.Now().UTC()
	for index, item := range []struct {
		action string
		status string
		detail string
	}{
		{action: "PROVIDER_CONFIG_SAVE", status: "SUCCESS", detail: "saved config"},
		{action: "PROVIDER_ACTIVATION_CHANGE", status: "FAILED", detail: "activation failed"},
		{action: "PROVIDER_ACTIVATION_CHANGE", status: "SUCCESS", detail: "activation enabled"},
	} {
		createdAt := now.Add(time.Duration(index) * time.Minute)
		mustCreateRecord(t, db, &models.ProviderAuditEvent{
			UserID:           user.ID,
			ProviderConfigID: &config.ID,
			Family:           domain.FamilyLLM,
			ProviderType:     domain.ProviderOpenAI,
			ActorType:        "USER",
			ActorID:          &actorID,
			Action:           item.action,
			Status:           item.status,
			CorrelationID:    "req-provider-query-77",
			Origin:           "PATCH /api/v1/user/providers/OPENAI",
			Detail:           item.detail,
			CreatedAt:        createdAt,
			UpdatedAt:        createdAt,
		})
	}

	result, err := repo.QueryProviderAuditEvents(ctx, user.ID, domain.ProviderAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			Limit:         1,
			Query:         "activation",
			ActorType:     "user",
			ActorID:       &actorID,
			CorrelationID: "req-provider-query-77",
		},
		Family:       "llm",
		ProviderType: "openai",
	})
	if err != nil {
		t.Fatalf("QueryProviderAuditEvents returned error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item on first page, got %d", len(result.Items))
	}
	if !result.PageInfo.HasNext || result.PageInfo.NextCursor == "" {
		t.Fatalf("expected next cursor on first page, got %#v", result.PageInfo)
	}
	if result.Summary.TotalEvents != 2 {
		t.Fatalf("expected summary total 2 for filtered search, got %d", result.Summary.TotalEvents)
	}
	if result.Summary.UniqueProviders != 1 {
		t.Fatalf("expected 1 unique provider, got %d", result.Summary.UniqueProviders)
	}
	if len(result.Summary.ActionBreakdown) == 0 || len(result.Summary.StatusBreakdown) == 0 {
		t.Fatalf("expected action and status breakdowns, got %#v", result.Summary)
	}

	nextPage, err := repo.QueryProviderAuditEvents(ctx, user.ID, domain.ProviderAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			Limit:         1,
			Cursor:        result.PageInfo.NextCursor,
			Query:         "activation",
			ActorType:     "user",
			ActorID:       &actorID,
			CorrelationID: "req-provider-query-77",
		},
		Family:       "llm",
		ProviderType: "openai",
	})
	if err != nil {
		t.Fatalf("QueryProviderAuditEvents second page returned error: %v", err)
	}
	if len(nextPage.Items) != 1 {
		t.Fatalf("expected 1 remaining provider audit item for filtered search, got %d", len(nextPage.Items))
	}
}
