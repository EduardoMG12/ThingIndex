package handlers

import (
	"errors"
	"net/http"

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
