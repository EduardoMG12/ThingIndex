package dto

import (
	"time"

	"thing-index/internal/domain"
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

type AuditCountBucketResponse struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type AuditPageInfoResponse struct {
	Limit      int    `json:"limit"`
	Returned   int    `json:"returned"`
	HasNext    bool   `json:"has_next"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type DocumentAuditSummaryResponse struct {
	TotalEvents       int64                      `json:"total_events"`
	UniqueDocuments   int64                      `json:"unique_documents"`
	UniqueCollections int64                      `json:"unique_collections"`
	UniqueRequests    int64                      `json:"unique_requests"`
	LastEventAt       *time.Time                 `json:"last_event_at,omitempty"`
	ActionBreakdown   []AuditCountBucketResponse `json:"action_breakdown,omitempty"`
	StatusBreakdown   []AuditCountBucketResponse `json:"status_breakdown,omitempty"`
}

type ProviderAuditSummaryResponse struct {
	TotalEvents     int64                      `json:"total_events"`
	UniqueProviders int64                      `json:"unique_providers"`
	UniqueRequests  int64                      `json:"unique_requests"`
	LastEventAt     *time.Time                 `json:"last_event_at,omitempty"`
	ActionBreakdown []AuditCountBucketResponse `json:"action_breakdown,omitempty"`
	StatusBreakdown []AuditCountBucketResponse `json:"status_breakdown,omitempty"`
}

type DocumentAuditQueryResponse struct {
	Items    []DocumentAuditEventResponse `json:"items"`
	Summary  DocumentAuditSummaryResponse `json:"summary"`
	PageInfo AuditPageInfoResponse        `json:"page_info"`
}

type ProviderAuditQueryResponse struct {
	Items    []ProviderAuditEventResponse `json:"items"`
	Summary  ProviderAuditSummaryResponse `json:"summary"`
	PageInfo AuditPageInfoResponse        `json:"page_info"`
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
	// Lista vazia tem de serializar como [], nao null.
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
	// Lista vazia tem de serializar como [], nao null.
	items := make([]ProviderAuditEventResponse, 0, len(events))
	for i := range events {
		if item := ToProviderAuditEventResponse(&events[i]); item != nil {
			items = append(items, *item)
		}
	}

	return items
}

func toAuditCountBucketResponses(items []domain.AuditCountBucket) []AuditCountBucketResponse {
	if len(items) == 0 {
		return nil
	}

	result := make([]AuditCountBucketResponse, 0, len(items))
	for i := range items {
		result = append(result, AuditCountBucketResponse{
			Key:   items[i].Key,
			Count: items[i].Count,
		})
	}

	return result
}

func ToDocumentAuditQueryResponse(result *domain.DocumentAuditQueryResult) *DocumentAuditQueryResponse {
	if result == nil {
		return nil
	}

	return &DocumentAuditQueryResponse{
		Items: ToDocumentAuditEventResponses(result.Items),
		Summary: DocumentAuditSummaryResponse{
			TotalEvents:       result.Summary.TotalEvents,
			UniqueDocuments:   result.Summary.UniqueDocuments,
			UniqueCollections: result.Summary.UniqueCollections,
			UniqueRequests:    result.Summary.UniqueRequests,
			LastEventAt:       result.Summary.LastEventAt,
			ActionBreakdown:   toAuditCountBucketResponses(result.Summary.ActionBreakdown),
			StatusBreakdown:   toAuditCountBucketResponses(result.Summary.StatusBreakdown),
		},
		PageInfo: AuditPageInfoResponse{
			Limit:      result.PageInfo.Limit,
			Returned:   result.PageInfo.Returned,
			HasNext:    result.PageInfo.HasNext,
			NextCursor: result.PageInfo.NextCursor,
		},
	}
}

func ToProviderAuditQueryResponse(result *domain.ProviderAuditQueryResult) *ProviderAuditQueryResponse {
	if result == nil {
		return nil
	}

	return &ProviderAuditQueryResponse{
		Items: ToProviderAuditEventResponses(result.Items),
		Summary: ProviderAuditSummaryResponse{
			TotalEvents:     result.Summary.TotalEvents,
			UniqueProviders: result.Summary.UniqueProviders,
			UniqueRequests:  result.Summary.UniqueRequests,
			LastEventAt:     result.Summary.LastEventAt,
			ActionBreakdown: toAuditCountBucketResponses(result.Summary.ActionBreakdown),
			StatusBreakdown: toAuditCountBucketResponses(result.Summary.StatusBreakdown),
		},
		PageInfo: AuditPageInfoResponse{
			Limit:      result.PageInfo.Limit,
			Returned:   result.PageInfo.Returned,
			HasNext:    result.PageInfo.HasNext,
			NextCursor: result.PageInfo.NextCursor,
		},
	}
}
