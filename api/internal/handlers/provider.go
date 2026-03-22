package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	providerpkg "thing-index/internal/pkg/provider"
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

// @Summary      List provider support matrix
// @Description  Returns the current support classification for provider integrations, including support tier, transport mode and operational notes.
// @Tags         providers
// @Produce      json
// @Param        family query string false "Optional provider family filter. The current public matrix is available for LLM."
// @Success      200 {object} response.Response{data=[]dto.ProviderSupportResponse}
// @Failure      400 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /providers/support [get]
func (h *ProviderHandler) ListSupportMatrix(c *gin.Context) {
	lang := c.GetString("language")
	family := strings.ToUpper(strings.TrimSpace(c.Query("family")))
	if family != "" && family != domain.FamilyLLM {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "provider.error_invalid_family"), nil)
		return
	}

	items, err := h.useCase.ListProviderSupportMatrix(c.Request.Context(), family)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(c, http.StatusOK, "success", "Matriz de suporte de providers carregada com sucesso.", dto.ToProviderSupportResponses(items))
}

// @Summary      Test provider readiness
// @Description  Validates the configured provider credentials and returns a readiness report with connectivity checks and optional model discovery details.
// @Tags         providers
// @Accept       json
// @Produce      json
// @Param        request body dto.ProviderConnectionRequest true "Provider test payload"
// @Success      200 {object} response.Response{data=dto.ProviderReadinessReportResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      502 {object} response.Response
// @Failure      504 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/providers/test [post]
func (h *ProviderHandler) TestConnection(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req testConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	report, err := h.useCase.TestConnection(c.Request.Context(), userID, req.Family, req.ProviderType, req.Credentials)
	if err != nil {
		status, message := providerConnectionFailureResponse(lang, req.ProviderType, req.Credentials, err)
		response.Error(c, status, "error", message, nil)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "provider.success_connection_test"), dto.ToProviderReadinessReportResponse(report))
}

// @Summary      List provider models
// @Description  Tries to discover available models for the selected LLM provider, using live discovery and cached fallbacks when applicable.
// @Tags         providers
// @Accept       json
// @Produce      json
// @Param        request body dto.ProviderConnectionRequest true "Provider model-discovery payload"
// @Success      200 {object} response.Response{data=dto.ProviderModelsResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      502 {object} response.Response
// @Failure      504 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/providers/models [post]
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

	response.Success(c, http.StatusOK, "success", "Modelos carregados com sucesso.", dto.ProviderModelsResponse{
		Models: models,
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

		if publicMetadata := providerpkg.PublicMetadata(providers[i].Family, credentials); len(publicMetadata) > 0 {
			publicMetadataByProvider[providers[i].ProviderType] = publicMetadata
		}
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "provider.success_list"), dto.ToProviderConfigResponses(providers, publicMetadataByProvider))
}

// @Summary      Get provider runtime overview
// @Description  Returns the currently active storage, indexing, and chat providers for the authenticated user.
// @Tags         providers
// @Produce      json
// @Success      200 {object} response.Response{data=dto.ProviderRuntimeOverviewResponse}
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/providers/runtime [get]
func (h *ProviderHandler) ListRuntimeOverview(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	overview, err := h.useCase.GetProviderRuntimeOverview(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(c, http.StatusOK, "success", "Runtime ativo de providers carregado com sucesso.", dto.ToProviderRuntimeOverviewResponse(overview))
}

// @Summary      Query provider audit events
// @Description  Query provider audit events with operational filters, summary and cursor pagination.
// @Tags         providers
// @Produce      json
// @Param        limit query int false "Items per page (max 100)"
// @Param        cursor query string false "Opaque cursor returned by the previous audit page"
// @Param        q query string false "Search over provider type, detail, origin and request correlation"
// @Param        family query string false "Provider family filter"
// @Param        provider_type query string false "Provider type filter"
// @Param        action query string false "Audit action filter"
// @Param        status query string false "Audit status filter"
// @Param        actor_type query string false "Actor type filter"
// @Param        actor_id query int false "Actor ID filter"
// @Param        correlation_id query string false "Request correlation ID filter"
// @Param        created_from query string false "Inclusive start timestamp in RFC3339"
// @Param        created_to query string false "Inclusive end timestamp in RFC3339"
// @Success      200 {object} response.Response{data=dto.ProviderAuditQueryResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/providers/audit [get]
func (h *ProviderHandler) ListProviderAuditEvents(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	limit, ok := parseOptionalLimitQuery(c, 20, "Parametro limit invalido.")
	if !ok {
		return
	}
	actorID, ok := parseOptionalUintQuery(c, "actor_id", i18n.T(lang, "error.invalid_actor_id"))
	if !ok {
		return
	}
	createdFrom, ok := parseOptionalTimeQuery(c, "created_from", i18n.T(lang, "error.invalid_created_from"))
	if !ok {
		return
	}
	createdTo, ok := parseOptionalTimeQuery(c, "created_to", i18n.T(lang, "error.invalid_created_to"))
	if !ok {
		return
	}

	result, err := h.useCase.QueryProviderAuditEvents(c.Request.Context(), userID, domain.ProviderAuditFilter{
		AuditPageFilter: domain.AuditPageFilter{
			Limit:         limit,
			Cursor:        c.Query("cursor"),
			Query:         c.Query("q"),
			Status:        c.Query("status"),
			ActorType:     c.Query("actor_type"),
			ActorID:       actorID,
			CorrelationID: c.Query("correlation_id"),
			CreatedFrom:   createdFrom,
			CreatedTo:     createdTo,
		},
		Family:       c.Query("family"),
		Action:       c.Query("action"),
		ProviderType: c.Query("provider_type"),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAuditCursorInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.invalid_audit_cursor"), nil)
		case errors.Is(err, domain.ErrAuditTimeRangeInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.invalid_audit_range"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "provider.success_audit_query"), dto.ToProviderAuditQueryResponse(result))
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
