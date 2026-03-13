package handlers

import (
	"errors"
	"net/http"
	"strings"
	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/validation"

	"github.com/gin-gonic/gin"
)

type ProviderHandler struct {
	useCase domain.ProviderUseCase
}

func NewProviderHandler(useCase domain.ProviderUseCase) *ProviderHandler {
	return &ProviderHandler{useCase: useCase}
}

type saveCredentialsRequest struct {
	Family             string            `json:"family" binding:"required"`
	ProviderType       string            `json:"provider_type" binding:"required"`
	Credentials        map[string]string `json:"credentials" binding:"required"`
	IsEnabled          *bool             `json:"is_enabled,omitempty"`
	EnabledForIndexing *bool             `json:"enabled_for_indexing,omitempty"`
	EnabledForChat     *bool             `json:"enabled_for_chat,omitempty"`
}

type updateProviderStateRequest struct {
	Family             string `json:"family"`
	ActivationTarget   string `json:"activation_target"`
	Enabled            *bool  `json:"enabled"`
	IsEnabled          *bool  `json:"is_enabled"`
	EnabledForIndexing *bool  `json:"enabled_for_indexing"`
	EnabledForChat     *bool  `json:"enabled_for_chat"`
}

func (h *ProviderHandler) SaveCredentials(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req saveCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	activation := domain.ProviderActivation{}
	if req.IsEnabled != nil {
		activation.IsEnabled = *req.IsEnabled
	}
	if req.EnabledForIndexing != nil {
		activation.EnabledForIndexing = *req.EnabledForIndexing
	}
	if req.EnabledForChat != nil {
		activation.EnabledForChat = *req.EnabledForChat
	}

	if err := h.useCase.SaveCredentials(c.Request.Context(), userID, req.Family, req.ProviderType, req.Credentials, activation); err != nil {
		switch {
		case errors.Is(err, domain.ErrProviderFamilyInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_family"), nil)
		case errors.Is(err, domain.ErrProviderTypeInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_type"), nil)
		case errors.Is(err, domain.ErrProviderCredentialsInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_credentials"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "provider.success_save"), nil)
}

func (h *ProviderHandler) ListProviders(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	providers, err := h.useCase.ListUserProviders(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	publicMetadataByProvider := make(map[string]map[string]string, len(providers))
	for i := range providers {
		credentials, err := h.useCase.GetDecryptedCredentials(c.Request.Context(), userID, providers[i].ProviderType)
		if err != nil {
			continue
		}

		if publicMetadata := extractPublicMetadata(providers[i].Family, credentials); len(publicMetadata) > 0 {
			publicMetadataByProvider[providers[i].ProviderType] = publicMetadata
		}
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "provider.success_list"), dto.ToProviderConfigResponses(providers, publicMetadataByProvider))
}

func extractPublicMetadata(family string, credentials map[string]string) map[string]string {
	publicMetadata := make(map[string]string)

	switch strings.ToUpper(strings.TrimSpace(family)) {
	case domain.FamilyLLM:
		if model := strings.TrimSpace(credentials["model"]); model != "" {
			publicMetadata["model"] = model
		}
	case domain.FamilyStorage:
		if bucket := strings.TrimSpace(credentials["bucket_name"]); bucket != "" {
			publicMetadata["bucket_name"] = bucket
		}
		if region := strings.TrimSpace(credentials["region"]); region != "" {
			publicMetadata["region"] = region
		}
	}

	return publicMetadata
}

func (h *ProviderHandler) UpdateProviderState(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	providerType := c.Param("providerType")

	var req updateProviderStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	enabled := false
	switch {
	case req.Enabled != nil:
		enabled = *req.Enabled
	case req.IsEnabled != nil:
		enabled = *req.IsEnabled
	case req.EnabledForIndexing != nil:
		enabled = *req.EnabledForIndexing
	case req.EnabledForChat != nil:
		enabled = *req.EnabledForChat
	}

	activationTarget := normalizeActivationTarget(req.ActivationTarget, req.Family, req.EnabledForIndexing, req.EnabledForChat)
	if activationTarget == "" {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_family"), nil)
		return
	}

	var err error
	switch activationTarget {
	case "STORAGE":
		err = h.useCase.SetProviderEnabled(c.Request.Context(), userID, domain.FamilyStorage, providerType, enabled)
	case "INDEXING":
		err = h.useCase.SetLLMUsageEnabled(c.Request.Context(), userID, providerType, domain.LLMUsageIndexing, enabled)
	case "CHAT":
		err = h.useCase.SetLLMUsageEnabled(c.Request.Context(), userID, providerType, domain.LLMUsageChat, enabled)
	default:
		err = domain.ErrProviderFamilyInvalid
	}

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProviderFamilyInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_family"), nil)
		case errors.Is(err, domain.ErrProviderTypeInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_type"), nil)
		case errors.Is(err, domain.ErrProviderCredentialsInvalid):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "provider.error_invalid_credentials"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "provider.success_save"), nil)
}

func normalizeActivationTarget(target string, family string, enabledForIndexing *bool, enabledForChat *bool) string {
	normalizedTarget := strings.ToUpper(strings.TrimSpace(target))
	switch normalizedTarget {
	case "STORAGE", "INDEXING", "CHAT":
		return normalizedTarget
	case "INDEX":
		return "INDEXING"
	case "LLM_INDEXING":
		return "INDEXING"
	case "LLM_CHAT":
		return "CHAT"
	}

	switch {
	case enabledForIndexing != nil:
		return "INDEXING"
	case enabledForChat != nil:
		return "CHAT"
	}

	switch strings.ToUpper(strings.TrimSpace(family)) {
	case domain.FamilyStorage:
		return "STORAGE"
	}

	return ""
}
