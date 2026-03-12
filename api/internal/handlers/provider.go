package handlers

import (
	"net/http"
	"thing-index/internal/domain"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type ProviderHandler struct {
	useCase domain.ProviderUseCase
}

func NewProviderHandler(useCase domain.ProviderUseCase) *ProviderHandler {
	return &ProviderHandler{useCase: useCase}
}

type saveCredentialsRequest struct {
	ProviderType string            `json:"provider_type" binding:"required"`
	Credentials  map[string]string `json:"credentials" binding:"required"`
}

func (h *ProviderHandler) SaveCredentials(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req saveCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), err.Error())
		return
	}

	if err := h.useCase.SaveCredentials(c.Request.Context(), userID, req.ProviderType, req.Credentials); err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
		return
	}

	response.Success(c, http.StatusOK, "success", "Credentials saved successfully", nil)
}

func (h *ProviderHandler) ListProviders(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	providers, err := h.useCase.ListUserProviders(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
		return
	}

	response.Success(c, http.StatusOK, "success", "Providers retrieved successfully", providers)
}
