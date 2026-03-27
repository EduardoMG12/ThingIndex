package usecase

import (
	"testing"
	"time"

	"thing-index/internal/domain"
)

func TestNormalizeDocumentAuditFilterRejectsInvalidCursor(t *testing.T) {
	t.Parallel()

	_, err := normalizeDocumentAuditFilter(domain.DocumentAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			Cursor: "invalid",
		},
	})
	if err != domain.ErrAuditCursorInvalid {
		t.Fatalf("expected ErrAuditCursorInvalid, got %v", err)
	}
}

func TestNormalizeProviderAuditFilterRejectsInvalidTimeRange(t *testing.T) {
	t.Parallel()

	createdFrom := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	createdTo := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)

	_, err := normalizeProviderAuditFilter(domain.ProviderAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			CreatedFrom: &createdFrom,
			CreatedTo:   &createdTo,
		},
	})
	if err != domain.ErrAuditTimeRangeInvalid {
		t.Fatalf("expected ErrAuditTimeRangeInvalid, got %v", err)
	}
}

func TestNormalizeProviderAuditFilterNormalizesProviderTypeAndDefaults(t *testing.T) {
	t.Parallel()

	filter, err := normalizeProviderAuditFilter(domain.ProviderAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			Limit:     999,
			Status:    " success ",
			ActorType: " user ",
		},
		Family:       " llm ",
		ProviderType: " open_router ",
		Action:       " provider_config_save ",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if filter.Limit != 100 {
		t.Fatalf("expected limit 100, got %d", filter.Limit)
	}
	if filter.Status != "SUCCESS" {
		t.Fatalf("expected normalized status, got %q", filter.Status)
	}
	if filter.ActorType != "USER" {
		t.Fatalf("expected normalized actor type, got %q", filter.ActorType)
	}
	if filter.Family != "LLM" {
		t.Fatalf("expected normalized family, got %q", filter.Family)
	}
	if filter.ProviderType != domain.ProviderOpenRouter {
		t.Fatalf("expected normalized provider type %q, got %q", domain.ProviderOpenRouter, filter.ProviderType)
	}
	if filter.Action != "PROVIDER_CONFIG_SAVE" {
		t.Fatalf("expected normalized action, got %q", filter.Action)
	}
}
