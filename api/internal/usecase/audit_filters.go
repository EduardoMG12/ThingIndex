package usecase

import (
	"strings"

	"thing-index/internal/domain"
)

func normalizeAuditPageFilter(filter *domain.AuditPageFilter) error {
	if filter == nil {
		return nil
	}

	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.ToUpper(strings.TrimSpace(filter.Status))
	filter.ActorType = strings.ToUpper(strings.TrimSpace(filter.ActorType))
	filter.CorrelationID = strings.TrimSpace(filter.CorrelationID)
	filter.Cursor = strings.TrimSpace(filter.Cursor)

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Cursor != "" {
		if _, err := domain.DecodeAuditCursor(filter.Cursor); err != nil {
			return domain.ErrAuditCursorInvalid
		}
	}

	if filter.CreatedFrom != nil {
		createdFrom := filter.CreatedFrom.UTC()
		filter.CreatedFrom = &createdFrom
	}
	if filter.CreatedTo != nil {
		createdTo := filter.CreatedTo.UTC()
		filter.CreatedTo = &createdTo
	}

	if filter.CreatedFrom != nil && filter.CreatedTo != nil && filter.CreatedFrom.After(*filter.CreatedTo) {
		return domain.ErrAuditTimeRangeInvalid
	}

	return nil
}

func normalizeDocumentAuditFilter(filter domain.DocumentAuditFilter) (domain.DocumentAuditFilter, error) {
	if err := normalizeAuditPageFilter(&filter.AuditPageFilter); err != nil {
		return domain.DocumentAuditFilter{}, err
	}

	filter.Action = strings.ToUpper(strings.TrimSpace(filter.Action))
	return filter, nil
}

func normalizeProviderAuditFilter(filter domain.ProviderAuditFilter) (domain.ProviderAuditFilter, error) {
	if err := normalizeAuditPageFilter(&filter.AuditPageFilter); err != nil {
		return domain.ProviderAuditFilter{}, err
	}

	filter.Family = strings.ToUpper(strings.TrimSpace(filter.Family))
	filter.ProviderType = normalizeProviderType(filter.ProviderType)
	filter.Action = strings.ToUpper(strings.TrimSpace(filter.Action))

	return filter, nil
}
