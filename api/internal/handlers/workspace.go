package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/validation"
)

// WorkspaceHandler expoe as anotacoes do workspace: marcas coloridas por
// paragrafo e a legenda da colecao. Tudo escopado pelo usuario do token.
type WorkspaceHandler struct {
	useCase domain.WorkspaceUseCase
}

func NewWorkspaceHandler(useCase domain.WorkspaceUseCase) *WorkspaceHandler {
	return &WorkspaceHandler{useCase: useCase}
}

// ListHighlights godoc
// @Summary      Lista as marcas coloridas do usuario numa collection
// @Tags         workspace
// @Produce      json
// @Param        id path int true "Collection ID"
// @Success      200 {object} dto.HighlightsResponse
// @Router       /documents/collections/{id}/highlights [get]
func (h *WorkspaceHandler) ListHighlights(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	collectionID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_collection_id"))
	if !ok {
		return
	}
	items, err := h.useCase.ListHighlights(c.Request.Context(), userID, collectionID)
	if err != nil {
		h.respondError(c, lang, err)
		return
	}
	response.Success(c, http.StatusOK, "success", i18n.T(lang, "workspace.success_highlights_list"), dto.ToHighlightsResponse(items))
}

// SetHighlight godoc
// @Summary      Marca um paragrafo com uma cor (cria ou troca)
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        id path int true "Collection ID"
// @Param        paragraphNodeId path int true "Paragraph node ID"
// @Param        request body dto.SetHighlightRequest true "Cor: amarelo, verde, azul, vermelho ou roxo"
// @Success      200 {object} dto.HighlightResponse
// @Router       /documents/collections/{id}/highlights/{paragraphNodeId} [put]
func (h *WorkspaceHandler) SetHighlight(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	collectionID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_collection_id"))
	if !ok {
		return
	}
	paragraphNodeID, ok := parseUintParam(c, "paragraphNodeId", i18n.T(lang, "workspace.error_invalid_paragraph_id"))
	if !ok {
		return
	}
	var req dto.SetHighlightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}
	highlight, err := h.useCase.SetHighlight(c.Request.Context(), userID, collectionID, paragraphNodeID, req.Color)
	if err != nil {
		h.respondError(c, lang, err)
		return
	}
	response.Success(c, http.StatusOK, "success", i18n.T(lang, "workspace.success_highlight_set"), dto.ToHighlightResponse(*highlight))
}

// RemoveHighlight godoc
// @Summary      Remove a marca de um paragrafo
// @Tags         workspace
// @Param        id path int true "Collection ID"
// @Param        paragraphNodeId path int true "Paragraph node ID"
// @Success      200
// @Router       /documents/collections/{id}/highlights/{paragraphNodeId} [delete]
func (h *WorkspaceHandler) RemoveHighlight(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	collectionID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_collection_id"))
	if !ok {
		return
	}
	paragraphNodeID, ok := parseUintParam(c, "paragraphNodeId", i18n.T(lang, "workspace.error_invalid_paragraph_id"))
	if !ok {
		return
	}
	if err := h.useCase.RemoveHighlight(c.Request.Context(), userID, collectionID, paragraphNodeID); err != nil {
		h.respondError(c, lang, err)
		return
	}
	response.Success(c, http.StatusOK, "success", i18n.T(lang, "workspace.success_highlight_removed"), nil)
}

// GetLegend godoc
// @Summary      Legenda das cores da collection
// @Tags         workspace
// @Produce      json
// @Param        id path int true "Collection ID"
// @Success      200 {object} dto.LegendResponse
// @Router       /documents/collections/{id}/legend [get]
func (h *WorkspaceHandler) GetLegend(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	collectionID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_collection_id"))
	if !ok {
		return
	}
	legend, err := h.useCase.GetLegend(c.Request.Context(), userID, collectionID)
	if err != nil {
		h.respondError(c, lang, err)
		return
	}
	response.Success(c, http.StatusOK, "success", i18n.T(lang, "workspace.success_legend_get"), dto.ToLegendResponse(legend))
}

// SetLegend godoc
// @Summary      Substitui a legenda das cores da collection
// @Tags         workspace
// @Accept       json
// @Produce      json
// @Param        id path int true "Collection ID"
// @Param        request body dto.LegendRequest true "Rotulos por cor"
// @Success      200 {object} dto.LegendResponse
// @Router       /documents/collections/{id}/legend [put]
func (h *WorkspaceHandler) SetLegend(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	collectionID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_collection_id"))
	if !ok {
		return
	}
	var req dto.LegendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}
	legend, err := h.useCase.SetLegend(c.Request.Context(), userID, collectionID, domain.Legend(req.Labels))
	if err != nil {
		h.respondError(c, lang, err)
		return
	}
	response.Success(c, http.StatusOK, "success", i18n.T(lang, "workspace.success_legend_set"), dto.ToLegendResponse(legend))
}

func (h *WorkspaceHandler) respondError(c *gin.Context, lang string, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidHighlightColor):
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "workspace.error_invalid_color"), nil)
	case errors.Is(err, domain.ErrInvalidLegend):
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "workspace.error_invalid_legend"), nil)
	case errors.Is(err, domain.ErrParagraphNotInCollection):
		response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "workspace.error_paragraph_not_in_collection"), nil)
	case errors.Is(err, domain.ErrCollectionNotFound):
		response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_collection_not_found"), nil)
	default:
		log.Printf("erro interno em %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
	}
}
