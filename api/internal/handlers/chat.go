package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/validation"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	useCase domain.ChatUseCase
}

func NewChatHandler(useCase domain.ChatUseCase) *ChatHandler {
	return &ChatHandler{useCase: useCase}
}

func (h *ChatHandler) StreamCollectionChat(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req dto.ChatStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	history := make([]domain.ChatMessage, 0, len(req.History))
	for _, item := range req.History {
		history = append(history, domain.ChatMessage{
			Role:    item.Role,
			Content: item.Content,
		})
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-store")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	if err := h.useCase.StreamCollectionChat(c.Request.Context(), userID, domain.ChatRequest{
		CollectionID: req.CollectionID,
		Prompt:       req.Prompt,
		Language:     lang,
		History:      history,
	}, func(event domain.ChatStreamEvent) error {
		return writeChatEvent(c, event.Type, event)
	}); err != nil {
		switch {
		case errors.Is(err, domain.ErrCollectionNotFound):
			_ = writeChatEvent(c, "error", gin.H{"message": i18n.T(lang, "doc.error_collection_not_found")})
		case errors.Is(err, domain.ErrLLMProviderNotConfigured):
			_ = writeChatEvent(c, "error", gin.H{"message": i18n.T(lang, "provider.error_llm_not_configured")})
		case errors.Is(err, domain.ErrLLMProviderUnavailable):
			_ = writeChatEvent(c, "error", gin.H{"message": i18n.T(lang, "chat.error_stream")})
		default:
			_ = writeChatEvent(c, "error", gin.H{"message": i18n.T(lang, "chat.error_stream")})
		}
	}
}

func writeChatEvent(c *gin.Context, event string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, encoded); err != nil {
		return err
	}

	c.Writer.Flush()
	return nil
}
