package dto

import (
	"time"

	"thing-index/internal/domain"
)

type ProviderRuntimeAssignmentResponse struct {
	Family           string            `json:"family"`
	ActivationTarget string            `json:"activation_target"`
	ProviderType     string            `json:"provider_type"`
	PublicMetadata   map[string]string `json:"public_metadata,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type ProviderRuntimeOverviewResponse struct {
	Storage  *ProviderRuntimeAssignmentResponse `json:"storage,omitempty"`
	Indexing *ProviderRuntimeAssignmentResponse `json:"indexing,omitempty"`
	Chat     *ProviderRuntimeAssignmentResponse `json:"chat,omitempty"`
}

func ToProviderRuntimeAssignmentResponse(assignment *domain.ProviderRuntimeAssignment) *ProviderRuntimeAssignmentResponse {
	if assignment == nil {
		return nil
	}

	return &ProviderRuntimeAssignmentResponse{
		Family:           assignment.Family,
		ActivationTarget: assignment.ActivationTarget,
		ProviderType:     assignment.ProviderType,
		PublicMetadata:   assignment.PublicMetadata,
		UpdatedAt:        assignment.UpdatedAt,
	}
}

func ToProviderRuntimeOverviewResponse(overview *domain.ProviderRuntimeOverview) *ProviderRuntimeOverviewResponse {
	if overview == nil {
		return &ProviderRuntimeOverviewResponse{}
	}

	return &ProviderRuntimeOverviewResponse{
		Storage:  ToProviderRuntimeAssignmentResponse(overview.Storage),
		Indexing: ToProviderRuntimeAssignmentResponse(overview.Indexing),
		Chat:     ToProviderRuntimeAssignmentResponse(overview.Chat),
	}
}
