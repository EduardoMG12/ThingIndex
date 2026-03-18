package domain

import "context"

type AuditContext struct {
	ActorType     string
	ActorID       *uint
	CorrelationID string
	Origin        string
	RequestMethod string
	RoutePath     string
	RemoteAddr    string
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
