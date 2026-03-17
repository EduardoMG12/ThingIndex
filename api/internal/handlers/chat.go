package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

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

func (h *ChatHandler) CreateConversation(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req dto.CreateChatConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	conversation, err := h.useCase.CreateConversation(c.Request.Context(), userID, req.CollectionID, req.Title)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCollectionRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_collection_required"), nil)
		case errors.Is(err, domain.ErrCollectionNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_collection_not_found"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusCreated, "success", i18n.T(lang, "chat.success_conversation_create"), dto.ToChatConversationResponse(conversation))
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	collectionID, err := parseChatUint(c.Query("collection_id"))
	if err != nil || collectionID == 0 {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.invalid_collection_id"), nil)
		return
	}

	conversations, listErr := h.useCase.ListConversations(c.Request.Context(), userID, collectionID)
	if listErr != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "chat.success_conversations_list"), dto.ToChatConversationResponses(conversations))
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	conversationID, err := parseChatUint(c.Param("id"))
	if err != nil || conversationID == 0 {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "chat.error_invalid_conversation_id"), nil)
		return
	}

	conversation, messages, getErr := h.useCase.GetConversationMessages(c.Request.Context(), userID, conversationID)
	if getErr != nil {
		switch {
		case errors.Is(getErr, domain.ErrChatConversationNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "chat.error_conversation_not_found"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "chat.success_conversation_get"), dto.ChatConversationDetailResponse{
		Conversation: dto.ToChatConversationResponse(conversation),
		Messages:     dto.ToChatConversationMessageResponses(messages),
	})
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	conversationID, err := parseChatUint(c.Param("id"))
	if err != nil || conversationID == 0 {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "chat.error_invalid_conversation_id"), nil)
		return
	}

	deleteErr := h.useCase.DeleteConversation(c.Request.Context(), userID, conversationID)
	if deleteErr != nil {
		switch {
		case errors.Is(deleteErr, domain.ErrChatConversationNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "chat.error_conversation_not_found"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "chat.success_conversation_delete"), nil)
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
		ConversationID: req.ConversationID,
		CollectionID:   req.CollectionID,
		Prompt:         req.Prompt,
		Language:       lang,
		History:        history,
	}, func(event domain.ChatStreamEvent) error {
		return writeChatEvent(c, event.Type, event)
	}); err != nil {
		switch {
		case errors.Is(err, domain.ErrChatConversationNotFound):
			_ = writeChatEvent(c, "error", gin.H{"message": i18n.T(lang, "chat.error_conversation_not_found")})
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

func parseChatUint(value string) (uint, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(parsed), nil
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
