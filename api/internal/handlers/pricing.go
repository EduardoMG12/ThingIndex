package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/validation"

	"github.com/gin-gonic/gin"
)

type PricingHandler struct {
	useCase domain.PricingUseCase
}

func NewPricingHandler(useCase domain.PricingUseCase) *PricingHandler {
	return &PricingHandler{useCase: useCase}
}

// @Summary      Save pricing source
// @Description  Register or update an official pricing source URL so the pricing worker can sync model prices for the authenticated user.
// @Tags         pricing
// @Accept       json
// @Produce      json
// @Param        request body dto.SavePricingSourceRequest true "Pricing source payload"
// @Success      201 {object} response.Response{data=dto.PricingSourceResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/pricing-sources [post]
func (h *PricingHandler) SaveSource(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req dto.SavePricingSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	source, err := h.useCase.SaveSource(c.Request.Context(), userID, domain.PricingSourceInput{
		ProviderType: req.ProviderType,
		Name:         req.Name,
		SourceURL:    req.SourceURL,
		IsEnabled:    isEnabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProviderTypeInvalid):
			response.Error(c, http.StatusBadRequest, "error", "provider_type invalido", nil)
		case errors.Is(err, domain.ErrPricingSourceInvalid):
			response.Error(c, http.StatusBadRequest, "error", "source_url invalida", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusCreated, "success", "Fonte de pricing salva.", dto.ToPricingSourceResponse(source))
}

// @Summary      List pricing sources
// @Description  Return the saved pricing source URLs and the latest sync status for the authenticated user.
// @Tags         pricing
// @Produce      json
// @Success      200 {object} response.Response{data=[]dto.PricingSourceResponse}
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/pricing-sources [get]
func (h *PricingHandler) ListSources(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	sources, err := h.useCase.ListSources(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(c, http.StatusOK, "success", "Fontes de pricing carregadas.", dto.ToPricingSourceResponses(sources))
}

// @Summary      List current model prices
// @Description  Return the latest synced pricing snapshot for the authenticated user across all configured pricing sources.
// @Tags         pricing
// @Produce      json
// @Success      200 {object} response.Response{data=[]dto.ModelPriceResponse}
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/prices [get]
func (h *PricingHandler) ListPrices(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	prices, err := h.useCase.ListPrices(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(c, http.StatusOK, "success", "Precos carregados.", dto.ToModelPriceResponses(prices))
}

// @Summary      List model price history
// @Description  Return the recent price history captured from synced pricing sources for the authenticated user.
// @Tags         pricing
// @Produce      json
// @Param        limit query int false "Maximum number of history rows to return" minimum(1)
// @Success      200 {object} response.Response{data=[]dto.ModelPriceHistoryResponse}
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/prices/history [get]
func (h *PricingHandler) ListPriceHistory(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	limit := 100
	if rawLimit := c.Query("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.useCase.ListPriceHistory(c.Request.Context(), userID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(c, http.StatusOK, "success", "Historico de precos carregado.", dto.ToModelPriceHistoryResponses(history))
}
