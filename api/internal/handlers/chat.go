package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/validation"

	"github.com/gin-gonic/gin"
)

// sseWriteDeadline e a janela de OCIOSIDADE do stream de chat: o prazo e
// renovado a cada evento escrito, entao o que ele limita e o silencio entre
// dois eventos, e nao a duracao total da resposta. Um LLM que para de produzir
// derruba a conexao em 2 minutos; um que continua produzindo pode levar o tempo
// que precisar.
const sseWriteDeadline = 2 * time.Minute

type ChatHandler struct {
	useCase domain.ChatUseCase
}

func NewChatHandler(useCase domain.ChatUseCase) *ChatHandler {
	return &ChatHandler{useCase: useCase}
}

// @Summary      Create chat conversation
// @Description  Create a persisted chat conversation scoped to one collection before starting the grounded response stream.
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateChatConversationRequest true "Conversation payload"
// @Success      201 {object} response.Response{data=dto.ChatConversationResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /chat/conversations [post]
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

// @Summary      List chat conversations
// @Description  List saved chat conversations for the authenticated user inside one collection.
// @Tags         chat
// @Produce      json
// @Param        collection_id query int true "Collection ID"
// @Success      200 {object} response.Response{data=[]dto.ChatConversationResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /chat/conversations [get]
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

// @Summary      Get chat conversation
// @Description  Fetch one persisted chat conversation and the grounded messages already stored for it.
// @Tags         chat
// @Produce      json
// @Param        id path int true "Conversation ID"
// @Success      200 {object} response.Response{data=dto.ChatConversationDetailResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /chat/conversations/{id} [get]
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

// @Summary      Update chat conversation title
// @Description  Rename one persisted chat conversation.
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        id path int true "Conversation ID"
// @Param        request body dto.UpdateChatConversationRequest true "Conversation update payload"
// @Success      200 {object} response.Response{data=dto.ChatConversationResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /chat/conversations/{id} [patch]
func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	conversationID, err := parseChatUint(c.Param("id"))
	if err != nil || conversationID == 0 {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "chat.error_invalid_conversation_id"), nil)
		return
	}

	var req dto.UpdateChatConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	conversation, updateErr := h.useCase.UpdateConversationTitle(c.Request.Context(), userID, conversationID, req.Title)
	if updateErr != nil {
		switch {
		case errors.Is(updateErr, domain.ErrTitleRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_title"), nil)
		case errors.Is(updateErr, domain.ErrChatConversationNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "chat.error_conversation_not_found"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "common.saved"), dto.ToChatConversationResponse(conversation))
}

// @Summary      Delete chat conversation
// @Description  Delete one persisted chat conversation and its grounded message history.
// @Tags         chat
// @Produce      json
// @Param        id path int true "Conversation ID"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /chat/conversations/{id} [delete]
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

// @Summary      Stream grounded chat response
// @Description  Start a grounded chat response for one persisted conversation and stream stage, source, delta, done, and error events over SSE.
// @Tags         chat
// @Accept       json
// @Produce      text/event-stream
// @Param        request body dto.ChatStreamRequest true "Chat stream payload"
// @Success      200 {string} string "Server-sent events stream"
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /chat/stream [post]
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
		log.Printf("chat stream event type=%s stage=%s delta_len=%d sources=%d", event.Type, event.Stage, len(event.Delta), len(event.Sources))
		return writeChatEvent(c, event.Type, event)
	}); err != nil {
		switch {
		case errors.Is(err, domain.ErrChatConversationNotFound):
			_ = writeChatEvent(c, "error", gin.H{"message": i18n.T(lang, "chat.error_conversation_not_found")})
		case errors.Is(err, domain.ErrCollectionNotFound):
			_ = writeChatEvent(c, "error", gin.H{"message": i18n.T(lang, "doc.error_collection_not_found")})
		case errors.Is(err, domain.ErrLLMProviderNotConfigured):
			_ = writeChatEvent(c, "error", gin.H{"message": i18n.T(lang, "provider.error_llm_not_configured")})
		default:
			// "Falha ao gerar resposta" sozinho esconde a causa que o usuario
			// precisa saber ("monthly usage limit", "context length"). O
			// detalhe vai junto, curto e numa linha.
			_ = writeChatEvent(c, "error", gin.H{"message": i18n.T(lang, "chat.error_stream"), "detail": chatErrorDetail(err)})
		}
	}
}

// chatErrorDetail devolve a causa do erro do provedor em uma linha de ate 200
// caracteres, para o usuario entender por que a resposta nao veio.
func chatErrorDetail(err error) string {
	if err == nil {
		return ""
	}
	detail := strings.Join(strings.Fields(err.Error()), " ")
	if runes := []rune(detail); len(runes) > 200 {
		detail = string(runes[:200]) + "…"
	}
	return detail
}

func parseChatUint(value string) (uint, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(parsed), nil
}

// writeChatEvent escreve um evento SSE e empurra o prazo de escrita.
//
// O ServerWriteTimeout do http.Server e um deadline ABSOLUTO contado do inicio
// da requisicao, e nao um limite de ociosidade. Com 30s configurados, qualquer
// resposta de chat que demorasse mais que isso era cortada no meio -- o usuario
// via o texto parar sem erro e sem explicacao. Aumentar o WriteTimeout global
// nao serve: afrouxaria o prazo de todas as outras rotas, que e justamente a
// protecao contra cliente lento.
//
// http.ResponseController.SetWriteDeadline empurra o prazo apenas desta
// conexao, a cada evento. Enquanto o LLM estiver produzindo texto o stream
// segue vivo; se ele parar de produzir, o deadline vence e a conexao morre --
// que e o comportamento desejado (limite de ociosidade, nao de duracao total).
//
// Servidores sem suporte a SetWriteDeadline devolvem ErrNotSupported. Nesse
// caso seguimos escrevendo normalmente em vez de abortar: perder o streaming
// longo e ruim, perder o chat inteiro e pior.
func writeChatEvent(c *gin.Context, event string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	rc := http.NewResponseController(c.Writer)
	if err := rc.SetWriteDeadline(time.Now().Add(sseWriteDeadline)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("estender o prazo de escrita do SSE: %w", err)
	}

	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, encoded); err != nil {
		return err
	}

	c.Writer.Flush()
	return nil
}
