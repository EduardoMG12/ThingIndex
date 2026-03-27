package domain

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"thing-index/internal/models"
)

type AuditContext struct {
	ActorType     string
	ActorID       *uint
	CorrelationID string
	Origin        string
	RequestMethod string
	RoutePath     string
	RemoteAddr    string
}

type AuditPageFilter struct {
	Limit         int
	Cursor        string
	Query         string
	Status        string
	ActorType     string
	ActorID       *uint
	CorrelationID string
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
}

type DocumentAuditFilter struct {
	AuditPageFilter
	Action       string
	DocumentID   *uint
	CollectionID *uint
}

type ProviderAuditFilter struct {
	AuditPageFilter
	Family       string
	Action       string
	ProviderType string
}

type AuditCursor struct {
	CreatedAt time.Time
	ID        uint
}

type AuditPageInfo struct {
	Limit      int
	Returned   int
	HasNext    bool
	NextCursor string
}

type AuditCountBucket struct {
	Key   string
	Count int64
}

type DocumentAuditSummary struct {
	TotalEvents       int64
	UniqueDocuments   int64
	UniqueCollections int64
	UniqueRequests    int64
	LastEventAt       *time.Time
	ActionBreakdown   []AuditCountBucket
	StatusBreakdown   []AuditCountBucket
}

type ProviderAuditSummary struct {
	TotalEvents     int64
	UniqueProviders int64
	UniqueRequests  int64
	LastEventAt     *time.Time
	ActionBreakdown []AuditCountBucket
	StatusBreakdown []AuditCountBucket
}

type DocumentAuditQueryResult struct {
	Items    []models.DocumentAuditEvent
	Summary  DocumentAuditSummary
	PageInfo AuditPageInfo
}

type ProviderAuditQueryResult struct {
	Items    []models.ProviderAuditEvent
	Summary  ProviderAuditSummary
	PageInfo AuditPageInfo
}

type auditContextKey struct{}

func WithAuditContext(ctx context.Context, auditCtx AuditContext) context.Context {
	return context.WithValue(ctx, auditContextKey{}, auditCtx)
}

func ExtractAuditContext(ctx context.Context) (AuditContext, bool) {
	if ctx == nil {
		return AuditContext{}, false
	}

	auditCtx, ok := ctx.Value(auditContextKey{}).(AuditContext)
	return auditCtx, ok
}

func EncodeAuditCursor(cursor AuditCursor) string {
	payload := cursor.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + strconv.FormatUint(uint64(cursor.ID), 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeAuditCursor(value string) (AuditCursor, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return AuditCursor{}, fmt.Errorf("empty cursor")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return AuditCursor{}, err
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return AuditCursor{}, fmt.Errorf("invalid cursor format")
	}

	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return AuditCursor{}, err
	}

	parsedID, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return AuditCursor{}, err
	}

	return AuditCursor{
		CreatedAt: createdAt.UTC(),
		ID:        uint(parsedID),
	}, nil
}
