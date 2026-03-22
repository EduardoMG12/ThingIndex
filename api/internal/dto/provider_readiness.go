package dto

import (
	"time"

	"thing-index/internal/domain"
)

type ProviderReadinessCheckResponse struct {
	Code     string            `json:"code"`
	Status   string            `json:"status"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ProviderReadinessReportResponse struct {
	Family           string                           `json:"family"`
	ProviderType     string                           `json:"provider_type"`
	Status           string                           `json:"status"`
	CheckedAt        time.Time                        `json:"checked_at"`
	Checks           []ProviderReadinessCheckResponse `json:"checks"`
	DiscoveredModels []string                         `json:"discovered_models,omitempty"`
}

func ToProviderReadinessReportResponse(report *domain.ProviderReadinessReport) *ProviderReadinessReportResponse {
	if report == nil {
		return nil
	}

	checks := make([]ProviderReadinessCheckResponse, 0, len(report.Checks))
	for _, check := range report.Checks {
		checks = append(checks, ProviderReadinessCheckResponse{
			Code:     check.Code,
			Status:   check.Status,
			Metadata: check.Metadata,
		})
	}

	discoveredModels := make([]string, 0, len(report.DiscoveredModels))
	for _, model := range report.DiscoveredModels {
		if model == "" {
			continue
		}
		discoveredModels = append(discoveredModels, model)
	}

	return &ProviderReadinessReportResponse{
		Family:           report.Family,
		ProviderType:     report.ProviderType,
		Status:           report.Status,
		CheckedAt:        report.CheckedAt,
		Checks:           checks,
		DiscoveredModels: discoveredModels,
	}
}
