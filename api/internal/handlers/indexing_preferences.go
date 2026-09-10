package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/validation"
)

// IndexingPreferencesHandler expoe os toggles do enriquecimento por usuario.
// Tres booleanos sem regra de negocio: o handler fala direto com o
// repositorio, sem caso de uso no meio.
type IndexingPreferencesHandler struct {
	repo domain.IndexingPreferencesRepository
}

func NewIndexingPreferencesHandler(repo domain.IndexingPreferencesRepository) *IndexingPreferencesHandler {
	return &IndexingPreferencesHandler{repo: repo}
}

// Get godoc
// @Summary      Escopo do enriquecimento por LLM na indexacao (por usuario)
// @Tags         indexing
// @Produce      json
// @Success      200 {object} dto.IndexingPreferencesResponse
// @Router       /user/indexing-preferences [get]
func (h *IndexingPreferencesHandler) Get(c *gin.Context) {
	lang := c.GetString("language")
	prefs, err := h.repo.GetIndexingPreferences(c.Request.Context(), c.GetUint("user_id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "indexing.error_preferences"), nil)
		return
	}
	response.Success(c, http.StatusOK, "success", i18n.T(lang, "indexing.success_preferences_get"), toIndexingPreferencesResponse(prefs))
}

// Put godoc
// @Summary      Define o escopo do enriquecimento por LLM na indexacao
// @Tags         indexing
// @Accept       json
// @Produce      json
// @Param        request body dto.IndexingPreferencesRequest true "Escopo"
// @Success      200 {object} dto.IndexingPreferencesResponse
// @Router       /user/indexing-preferences [put]
func (h *IndexingPreferencesHandler) Put(c *gin.Context) {
	lang := c.GetString("language")
	var req dto.IndexingPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}
	prefs := domain.IndexingPreferences{DocumentSynopsis: req.DocumentSynopsis, PageSummaries: req.PageSummaries, SectionSummaries: req.SectionSummaries}
	if err := h.repo.SaveIndexingPreferences(c.Request.Context(), c.GetUint("user_id"), prefs); err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "indexing.error_preferences"), nil)
		return
	}
	response.Success(c, http.StatusOK, "success", i18n.T(lang, "indexing.success_preferences_set"), toIndexingPreferencesResponse(prefs))
}

func toIndexingPreferencesResponse(p domain.IndexingPreferences) dto.IndexingPreferencesResponse {
	return dto.IndexingPreferencesResponse{DocumentSynopsis: p.DocumentSynopsis, PageSummaries: p.PageSummaries, SectionSummaries: p.SectionSummaries}
}
