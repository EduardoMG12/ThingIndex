package handlers

import (
	"context"

	"thing-index/internal/domain"

	"github.com/gin-gonic/gin"
)

func withRequestAuditContext(c *gin.Context, userID uint) context.Context {
	actorID := userID
	requestID := c.GetString("request_id")
	routePath := c.FullPath()
	origin := c.Request.Method
	if routePath != "" {
		origin = c.Request.Method + " " + routePath
	}

	return domain.WithAuditContext(c.Request.Context(), domain.AuditContext{
		ActorType:     "USER",
		ActorID:       &actorID,
		CorrelationID: requestID,
		Origin:        origin,
		RequestMethod: c.Request.Method,
		RoutePath:     routePath,
		RemoteAddr:    c.ClientIP(),
	})
}
