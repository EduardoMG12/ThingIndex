//go:build integration

package repository

import (
	"context"
	"testing"

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
		Limit:        10,
		Family:       "llm",
		ProviderType: "openai",
		Action:       "provider_config_save",
		Status:       "success",
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
