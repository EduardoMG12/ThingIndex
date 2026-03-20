package dto

import (
	"time"

	"thing-index/internal/models"
)

type DocumentAuditEventResponse struct {
	ID            uint           `json:"id"`
	UserID        uint           `json:"user_id"`
	DocumentID    *uint          `json:"document_id,omitempty"`
	CollectionID  *uint          `json:"collection_id,omitempty"`
	ActorType     string         `json:"actor_type"`
	ActorID       *uint          `json:"actor_id,omitempty"`
	Action        string         `json:"action"`
	Status        string         `json:"status"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Origin        string         `json:"origin,omitempty"`
	Title         string         `json:"title,omitempty"`
	Detail        string         `json:"detail,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type ProviderAuditEventResponse struct {
	ID               uint           `json:"id"`
	UserID           uint           `json:"user_id"`
	ProviderConfigID *uint          `json:"provider_config_id,omitempty"`
	Family           string         `json:"family"`
	ProviderType     string         `json:"provider_type"`
	ActorType        string         `json:"actor_type"`
	ActorID          *uint          `json:"actor_id,omitempty"`
	Action           string         `json:"action"`
	Status           string         `json:"status"`
	CorrelationID    string         `json:"correlation_id,omitempty"`
	Origin           string         `json:"origin,omitempty"`
	Detail           string         `json:"detail,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

func ToDocumentAuditEventResponse(event *models.DocumentAuditEvent) *DocumentAuditEventResponse {
	if event == nil {
		return nil
	}

	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	return &DocumentAuditEventResponse{
		ID:            event.ID,
		UserID:        event.UserID,
		DocumentID:    event.DocumentID,
		CollectionID:  event.CollectionID,
		ActorType:     event.ActorType,
		ActorID:       event.ActorID,
		Action:        event.Action,
		Status:        event.Status,
		CorrelationID: event.CorrelationID,
		Origin:        event.Origin,
		Title:         event.Title,
		Detail:        event.Detail,
		Metadata:      metadata,
		CreatedAt:     event.CreatedAt,
	}
}

func ToDocumentAuditEventResponses(events []models.DocumentAuditEvent) []DocumentAuditEventResponse {
	if len(events) == 0 {
		return nil
	}

	items := make([]DocumentAuditEventResponse, 0, len(events))
	for i := range events {
		if item := ToDocumentAuditEventResponse(&events[i]); item != nil {
			items = append(items, *item)
		}
	}

	return items
}

func ToProviderAuditEventResponse(event *models.ProviderAuditEvent) *ProviderAuditEventResponse {
	if event == nil {
		return nil
	}

	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	return &ProviderAuditEventResponse{
		ID:               event.ID,
		UserID:           event.UserID,
		ProviderConfigID: event.ProviderConfigID,
		Family:           event.Family,
		ProviderType:     event.ProviderType,
		ActorType:        event.ActorType,
		ActorID:          event.ActorID,
		Action:           event.Action,
		Status:           event.Status,
		CorrelationID:    event.CorrelationID,
		Origin:           event.Origin,
		Detail:           event.Detail,
		Metadata:         metadata,
		CreatedAt:        event.CreatedAt,
	}
}

func ToProviderAuditEventResponses(events []models.ProviderAuditEvent) []ProviderAuditEventResponse {
	if len(events) == 0 {
		return nil
	}

	items := make([]ProviderAuditEventResponse, 0, len(events))
	for i := range events {
		if item := ToProviderAuditEventResponse(&events[i]); item != nil {
			items = append(items, *item)
		}
	}

	return items
}
