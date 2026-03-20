package handlers

import (
	"errors"
	"net/http"
	"net/url"
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

type testConnectionRequest struct {
	Family       string            `json:"family" binding:"required"`
	ProviderType string            `json:"provider_type" binding:"required"`
	Credentials  map[string]string `json:"credentials" binding:"required"`
}

type updateProviderStateRequest struct {
	ProviderType       string `json:"provider_type"`
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

	if err := h.useCase.SaveCredentials(withRequestAuditContext(c, userID), userID, req.Family, req.ProviderType, req.Credentials, activation); err != nil {
		switch {
		case errors.Is(err, domain.ErrProviderFamilyInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_family"), nil)
		case errors.Is(err, domain.ErrProviderTypeInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_type"), nil)
		case errors.Is(err, domain.ErrProviderCredentialsInvalid):
			response.Error(c, http.StatusBadRequest, "error", invalidProviderCredentialsMessage(lang, req.ProviderType, req.Credentials), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "provider.success_save"), nil)
}

func (h *ProviderHandler) TestConnection(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req testConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	if err := h.useCase.TestConnection(c.Request.Context(), userID, req.Family, req.ProviderType, req.Credentials); err != nil {
		status, message := providerConnectionFailureResponse(lang, req.ProviderType, req.Credentials, err)
		response.Error(c, status, "error", message, nil)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "provider.success_connection_test"), nil)
}

func (h *ProviderHandler) ListModels(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req testConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	models, err := h.useCase.ListModels(c.Request.Context(), userID, req.Family, req.ProviderType, req.Credentials)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProviderFamilyInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_family"), nil)
		case errors.Is(err, domain.ErrProviderTypeInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_type"), nil)
		case errors.Is(err, domain.ErrProviderCredentialsInvalid):
			response.Error(c, http.StatusBadRequest, "error", invalidProviderModelsMessage(lang, req.ProviderType, req.Credentials), nil)
		case errors.Is(err, domain.ErrProviderModelsUnsupported):
			response.Error(c, http.StatusBadRequest, "error", "A listagem automatica de modelos ainda nao esta disponivel para esse provider.", nil)
		case errors.Is(err, domain.ErrProviderModelsTimeout):
			response.Error(c, http.StatusGatewayTimeout, "error", "A listagem de modelos excedeu o tempo limite. Revise a URL base do provider e tente novamente.", nil)
		case errors.Is(err, domain.ErrProviderModelsUnavailable):
			response.Error(c, http.StatusBadGateway, "error", "Nao foi possivel consultar os modelos no provider. Revise a URL base e a conectividade do backend com esse endpoint.", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", "Modelos carregados com sucesso.", gin.H{
		"models": models,
	})
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

func (h *ProviderHandler) ListProviderAuditEvents(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	limit, ok := parseOptionalLimitQuery(c, 20, "Parametro limit invalido.")
	if !ok {
		return
	}

	events, err := h.useCase.ListProviderAuditEvents(c.Request.Context(), userID, domain.ProviderAuditFilter{
		Limit:        limit,
		Family:       c.Query("family"),
		Status:       c.Query("status"),
		Action:       c.Query("action"),
		ProviderType: c.Query("provider_type"),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(c, http.StatusOK, "success", "Auditoria de providers carregada com sucesso.", dto.ToProviderAuditEventResponses(events))
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

	var req updateProviderStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	providerType := strings.TrimSpace(c.Param("providerType"))
	if providerType == "" || strings.EqualFold(providerType, "undefined") || strings.EqualFold(providerType, "null") {
		providerType = req.ProviderType
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
		err = h.useCase.SetProviderEnabled(withRequestAuditContext(c, userID), userID, domain.FamilyStorage, providerType, enabled)
	case "INDEXING":
		err = h.useCase.SetLLMUsageEnabled(withRequestAuditContext(c, userID), userID, providerType, domain.LLMUsageIndexing, enabled)
	case "CHAT":
		err = h.useCase.SetLLMUsageEnabled(withRequestAuditContext(c, userID), userID, providerType, domain.LLMUsageChat, enabled)
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

func invalidProviderCredentialsMessage(lang string, providerType string, credentials map[string]string) string {
	if isLoopbackOllamaBaseURL(providerType, credentials) {
		return i18n.T(lang, "provider.error_ollama_localhost")
	}

	return i18n.T(lang, "provider.error_invalid_credentials")
}

func invalidProviderModelsMessage(lang string, providerType string, credentials map[string]string) string {
	if isLoopbackOllamaBaseURL(providerType, credentials) {
		return i18n.T(lang, "provider.error_models_ollama_localhost")
	}

	return "Nao foi possivel listar modelos para esse provider com a configuracao atual."
}

func providerConnectionFailureResponse(lang string, providerType string, credentials map[string]string, err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrProviderFamilyInvalid):
		return http.StatusBadRequest, i18n.T(lang, "provider.error_invalid_family")
	case errors.Is(err, domain.ErrProviderTypeInvalid):
		return http.StatusBadRequest, i18n.T(lang, "provider.error_invalid_type")
	case errors.Is(err, domain.ErrProviderCredentialsInvalid):
		return http.StatusBadRequest, invalidProviderCredentialsMessage(lang, providerType, credentials)
	case errors.Is(err, domain.ErrProviderAuthenticationFailed):
		return http.StatusUnauthorized, i18n.T(lang, "provider.error_auth_failed")
	case errors.Is(err, domain.ErrProviderModelInvalid):
		return http.StatusBadRequest, i18n.T(lang, "provider.error_model_invalid")
	case errors.Is(err, domain.ErrProviderConnectionTimeout):
		return http.StatusGatewayTimeout, i18n.T(lang, "provider.error_connection_timeout")
	case errors.Is(err, domain.ErrProviderEndpointUnavailable):
		if isLoopbackOllamaBaseURL(providerType, credentials) {
			return http.StatusBadGateway, i18n.T(lang, "provider.error_ollama_localhost")
		}
		return http.StatusBadGateway, i18n.T(lang, "provider.error_endpoint_unreachable")
	case errors.Is(err, domain.ErrProviderConnectionFailed):
		return http.StatusBadGateway, i18n.T(lang, "provider.error_connection_failed")
	default:
		return http.StatusInternalServerError, i18n.T(lang, "error.internal")
	}
}

func isLoopbackOllamaBaseURL(providerType string, credentials map[string]string) bool {
	if !strings.EqualFold(strings.TrimSpace(providerType), domain.ProviderOllama) {
		return false
	}

	baseURL := strings.TrimSpace(credentials["base_url"])
	if baseURL == "" {
		return false
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
