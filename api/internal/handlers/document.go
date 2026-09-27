package handlers

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/indexio"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/validation"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	useCase            domain.DocumentCatalogUseCase
	maxUploadBodyBytes int64
}

func NewDocumentHandler(useCase domain.DocumentCatalogUseCase, maxUploadBodyBytes int64) *DocumentHandler {
	return &DocumentHandler{
		useCase:            useCase,
		maxUploadBodyBytes: maxUploadBodyBytes,
	}
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBodyBytes)
	title := c.PostForm("title")
	categoryName := strings.TrimSpace(c.PostForm("category"))
	categoryDesc := ""
	if categoryName != "" {
		categoryDesc = i18n.T(lang, "doc.auto_category_desc")
	}
	collectionName := strings.TrimSpace(c.PostForm("collection"))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		if isUploadTooLargeError(err) {
			response.Error(c, http.StatusRequestEntityTooLarge, "error", i18n.T(lang, "doc.error_file_too_large"), nil)
			return
		}
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_file"), nil)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		h.respondInternalError(c, lang, err)
		return
	}
	defer file.Close()

	doc, err := h.useCase.UploadDocument(
		withRequestAuditContext(c, userID),
		userID,
		title,
		file,
		fileHeader.Size,
		fileHeader.Header.Get("Content-Type"),
		categoryName,
		categoryDesc,
		collectionName,
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTitleRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_title"), nil)
		case errors.Is(err, domain.ErrFileRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_file"), nil)
		case errors.Is(err, domain.ErrFileTooLarge):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_file_too_large"), nil)
		case errors.Is(err, domain.ErrDocumentInvalidMime):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_invalid_mime"), nil)
		case errors.Is(err, domain.ErrCollectionRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_collection_required"), nil)
		case errors.Is(err, domain.ErrCollectionCategoryConflict):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_collection_category_conflict"), nil)
		case errors.Is(err, domain.ErrStorageProviderUnavailable):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_storage_provider"), nil)
		case errors.Is(err, domain.ErrUploadFailed):
			// Este 500 tambem precisa registrar a causa: e justamente o
			// caminho de storage indisponivel, o mais dificil de diagnosticar
			// sem log.
			log.Printf("erro interno em %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "doc.error_upload"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusCreated, "success", i18n.T(lang, "doc.success_upload"), dto.ToDocumentResponse(doc))
}

func (h *DocumentHandler) CreateCollection(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req dto.CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	collection, err := h.useCase.CreateCollection(c.Request.Context(), userID, req.Name, req.Category, req.Metadata)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCollectionRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_collection_required"), nil)
		case errors.Is(err, domain.ErrCollectionDuplicate):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_collection_duplicate"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusCreated, "success", i18n.T(lang, "doc.success_collection_create"), dto.ToCollectionResponse(collection))
}

func (h *DocumentHandler) ListCollections(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	collections, err := h.useCase.ListCollections(c.Request.Context(), userID)
	if err != nil {
		h.respondInternalError(c, lang, err)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_collections_list"), dto.ToCollectionsResponse(collections))
}

func (h *DocumentHandler) GetCollection(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	collectionID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_collection_id"))
	if !ok {
		return
	}

	collection, err := h.useCase.GetCollection(c.Request.Context(), userID, collectionID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCollectionNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_collection_not_found"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_collection_get"), dto.ToCollectionResponse(collection))
}

func (h *DocumentHandler) GetCollectionContext(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	collectionID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_collection_id"))
	if !ok {
		return
	}

	collectionContext, err := h.useCase.GetCollectionContext(c.Request.Context(), userID, collectionID, c.Query("q"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCollectionNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_collection_not_found"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_collection_context_get"), dto.ToCollectionContextResponse(collectionContext))
}

// defaultNodeConnectionMinScore e o limiar padrao das conexoes derivadas.
//
// O valor 2 exige que ao menos dois termos da origem apareçam no destino. Um
// unico termo em comum e ruido: "contrato" liga qualquer par de documentos
// juridicos sem que exista relacao de conteudo. E o mesmo raciocinio da Secao
// de Limiar do artigo, onde a taxa de falso positivo de 1,000 mostrou que
// devolver "o menos ruim" e pior do que devolver nada.
const defaultNodeConnectionMinScore = 2

// GetNodeConnections devolve as conexoes de um no do sumario com nos de outros
// documentos da mesma colecao.
//
// As conexoes sao DERIVADAS a cada chamada e nunca persistidas: nao existe
// tabela de arestas no esquema, e essa ausencia e deliberada (principio II da
// constituicao -- grafo jamais como camada primaria de recuperacao).
//
// @Summary      List derived connections for an outline node
// @Description  Returns outline nodes from OTHER documents in the same collection whose text overlaps the given node. Connections are recomputed on every request and never stored; each one carries the terms that justify it.
// @Tags         documents
// @Produce      json
// @Param        id path int true "Collection ID"
// @Param        node_id query string true "Outline node identifier"
// @Param        min_score query int false "Minimum overlap score (default 2)"
// @Success      200 {object} dto.NodeConnectionsResponse
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /documents/collections/{id}/connections [get]
func (h *DocumentHandler) GetNodeConnections(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	collectionID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_collection_id"))
	if !ok {
		return
	}

	nodeID := strings.TrimSpace(c.Query("node_id"))
	if nodeID == "" {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_outline_node_required"), nil)
		return
	}

	minScore := defaultNodeConnectionMinScore
	if raw := strings.TrimSpace(c.Query("min_score")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_invalid_min_score"), nil)
			return
		}
		minScore = parsed
	}

	connections, err := h.useCase.GetNodeConnections(c.Request.Context(), userID, collectionID, nodeID, minScore)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCollectionNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_collection_not_found"), nil)
		case errors.Is(err, domain.ErrOutlineNodeNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_outline_node_not_found"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_node_connections_get"),
		dto.ToNodeConnectionsResponse(nodeID, minScore, connections))
}

func (h *DocumentHandler) RetryIndexing(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	documentID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}

	document, err := h.useCase.RetryDocumentIndexing(withRequestAuditContext(c, userID), userID, documentID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDocumentNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_not_found"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", "Documento reenfileirado para indexacao.", dto.ToDocumentResponse(document))
}

func (h *DocumentHandler) ListCategories(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	categories, err := h.useCase.ListCategories(c.Request.Context(), userID)
	if err != nil {
		h.respondInternalError(c, lang, err)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_categories_list"), dto.ToCategoriesResponse(categories))
}

func (h *DocumentHandler) CreateCategory(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	category, err := h.useCase.CreateCategory(c.Request.Context(), userID, req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCategoryRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_category_required"), nil)
		case errors.Is(err, domain.ErrCategoryDuplicate):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_category_duplicate"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusCreated, "success", i18n.T(lang, "doc.success_category_create"), dto.ToCategoryResponse(category))
}

func (h *DocumentHandler) UpdateCategory(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	categoryID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_category_id"))
	if !ok {
		return
	}

	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	if req.Name == nil && req.Description == nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_category_patch_empty"), nil)
		return
	}

	category, err := h.useCase.UpdateCategory(c.Request.Context(), userID, categoryID, req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCategoryNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_category_not_found"), nil)
		case errors.Is(err, domain.ErrCategoryRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_category_required"), nil)
		case errors.Is(err, domain.ErrCategoryDuplicate):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_category_duplicate"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_category_update"), dto.ToCategoryResponse(category))
}

func (h *DocumentHandler) UpdateCollectionCategory(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	collectionIdentifier := c.Param("collection")

	var req dto.UpdateCollectionCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	count, collection, err := h.useCase.UpdateCollectionCategory(
		c.Request.Context(),
		userID,
		collectionIdentifier,
		req.Category,
		i18n.T(lang, "doc.auto_category_desc"),
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCollectionRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_collection_required"), nil)
		case errors.Is(err, domain.ErrCollectionNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_collection_not_found"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_collection_category_update"), gin.H{
		"collection":        dto.ToCollectionResponse(collection),
		"updated_documents": count,
	})
}

func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	docs, err := h.useCase.ListDocuments(c.Request.Context(), userID)
	if err != nil {
		h.respondInternalError(c, lang, err)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_documents_list"), dto.ToDocumentsResponse(docs))
}

// @Summary      Query document audit events
// @Description  Query document audit events with operational filters, summary and cursor pagination.
// @Tags         documents
// @Produce      json
// @Param        limit query int false "Items per page (max 100)"
// @Param        cursor query string false "Opaque cursor returned by the previous audit page"
// @Param        q query string false "Search over title, detail, origin and request correlation"
// @Param        action query string false "Audit action filter"
// @Param        status query string false "Audit status filter"
// @Param        actor_type query string false "Actor type filter"
// @Param        actor_id query int false "Actor ID filter"
// @Param        correlation_id query string false "Request correlation ID filter"
// @Param        document_id query int false "Document ID filter"
// @Param        collection_id query int false "Collection ID filter"
// @Param        created_from query string false "Inclusive start timestamp in RFC3339"
// @Param        created_to query string false "Inclusive end timestamp in RFC3339"
// @Success      200 {object} response.Response{data=dto.DocumentAuditQueryResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /documents/audit [get]
func (h *DocumentHandler) ListDocumentAuditEvents(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	limit, ok := parseOptionalLimitQuery(c, 20, "Parametro limit invalido.")
	if !ok {
		return
	}
	documentID, ok := parseOptionalUintQuery(c, "document_id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}
	collectionID, ok := parseOptionalUintQuery(c, "collection_id", i18n.T(lang, "error.invalid_collection_id"))
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

	result, err := h.useCase.QueryDocumentAuditEvents(c.Request.Context(), userID, domain.DocumentAuditFilter{
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
		Action:       c.Query("action"),
		DocumentID:   documentID,
		CollectionID: collectionID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAuditCursorInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.invalid_audit_cursor"), nil)
		case errors.Is(err, domain.ErrAuditTimeRangeInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.invalid_audit_range"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_audit_query"), dto.ToDocumentAuditQueryResponse(result))
}

func (h *DocumentHandler) GetDocument(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	documentID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}

	doc, err := h.useCase.GetDocument(c.Request.Context(), userID, documentID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDocumentNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_document_not_found"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_document_get"), dto.ToDocumentResponse(doc))
}

func (h *DocumentHandler) GetDocumentContent(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	documentID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}

	content, err := h.useCase.GetDocumentContent(c.Request.Context(), userID, documentID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDocumentNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_document_not_found"), nil)
		case errors.Is(err, domain.ErrDocumentSourceMissing):
			response.Error(c, http.StatusGone, "error", "O arquivo original nao foi encontrado no storage.", nil)
		case errors.Is(err, domain.ErrStorageProviderUnavailable):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_storage_provider"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}
	defer content.Reader.Close()

	disposition := "attachment"
	switch content.Document.MimeType {
	case "application/pdf", "text/plain":
		disposition = "inline"
	}

	c.Header("Content-Type", content.Document.MimeType)
	c.Header("Content-Disposition", buildSafeContentDisposition(disposition, content.Document.Title))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "no-referrer")
	if content.Document.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(content.Document.Size, 10))
	}
	c.Status(http.StatusOK)

	if _, copyErr := io.Copy(c.Writer, content.Reader); copyErr != nil {
		_ = c.Error(copyErr)
	}
}

// @Summary      Get document index
// @Description  Return the persisted structural index for a document owned by the authenticated user, including pages, paragraphs, tables, metadata, chunks, outline, evaluation, and monitoring.
// @Tags         documents
// @Produce      json
// @Param        id path int true "Document ID"
// @Param        q query string false "Optional retrieval query evaluated over the persisted outline and page summaries"
// @Success      200 {object} response.Response{data=dto.DocumentIndexResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /documents/{id}/index [get]
func (h *DocumentHandler) GetDocumentIndex(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	documentID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}

	index, err := h.useCase.GetDocumentIndex(c.Request.Context(), userID, documentID, c.Query("q"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDocumentNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_document_not_found"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_document_get"), dto.ToDocumentIndexResponse(index))
}

// @Summary      Export document index
// @Description  Download the versioned `document_index_export` JSON artifact for a document owned by the authenticated user. This artifact is consumed by the index import CLI.
// @Tags         documents
// @Produce      json
// @Param        id path int true "Document ID"
// @Success      200 {object} indexio.DocumentIndexExportEnvelope
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /documents/{id}/index/export [get]
func (h *DocumentHandler) ExportDocumentIndex(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	documentID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}

	index, err := h.useCase.GetDocumentIndex(c.Request.Context(), userID, documentID, "")
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDocumentNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_document_not_found"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	indexResponse := dto.ToDocumentIndexResponse(index)
	exported := indexio.NewDocumentIndexExport(resolveDocumentIndexExportOrigin(c), indexResponse, time.Now().UTC())
	if exported == nil {
		h.respondInternalError(c, lang, err)
		return
	}

	filename := buildDocumentIndexExportFilename(indexResponse.Document.Title)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", buildSafeContentDisposition("attachment", filename))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "no-referrer")
	c.JSON(http.StatusOK, exported)
}

// @Summary      Get aggregated document monitoring
// @Description  Return the operational monitoring summary for the authenticated user, including queue counters, failure buckets and LLM telemetry aggregates.
// @Tags         documents
// @Produce      json
// @Success      200 {object} response.Response{data=dto.DocumentMonitoringSummaryResponse}
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /documents/monitoring/summary [get]
func (h *DocumentHandler) GetDocumentMonitoringSummary(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	summary, err := h.useCase.GetDocumentMonitoringSummary(c.Request.Context(), userID)
	if err != nil {
		h.respondInternalError(c, lang, err)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_documents_list"), dto.ToDocumentMonitoringSummaryResponse(summary))
}

func resolveDocumentIndexExportOrigin(c *gin.Context) string {
	for _, candidate := range []string{
		c.GetHeader(indexio.DocumentIndexExportOriginHeader),
		c.GetHeader("Origin"),
	} {
		if normalized := indexio.NormalizeExportOrigin(candidate); normalized != "" {
			return normalized
		}
	}

	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		return ""
	}

	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https") {
		scheme = "https"
	}

	return indexio.NormalizeExportOrigin(scheme + "://" + host)
}

func (h *DocumentHandler) ListDocumentsByCategory(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	categoryID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_category_id"))
	if !ok {
		return
	}

	docs, err := h.useCase.ListDocumentsByCategory(c.Request.Context(), categoryID, userID)
	if err != nil {
		h.respondInternalError(c, lang, err)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_documents_list"), dto.ToDocumentsResponse(docs))
}

func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	documentID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}

	if err := h.useCase.DeleteDocument(withDeletionAuditContext(c, userID), userID, documentID); err != nil {
		switch {
		case errors.Is(err, domain.ErrDocumentNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_document_not_found"), nil)
		case errors.Is(err, domain.ErrDocumentSourceDeleteFailed):
			response.Error(c, http.StatusBadGateway, "error", i18n.T(lang, "doc.error_delete_source"), nil)
		case errors.Is(err, domain.ErrStorageProviderUnavailable):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_storage_provider"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", "Documento excluido com sucesso.", nil)
}

func (h *DocumentHandler) DeleteCollectionDocuments(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	collectionID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_collection_id"))
	if !ok {
		return
	}

	deletedCount, collection, err := h.useCase.DeleteCollectionDocuments(withDeletionAuditContext(c, userID), userID, collectionID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCollectionNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_collection_not_found"), nil)
		case errors.Is(err, domain.ErrDocumentSourceDeleteFailed):
			response.Error(c, http.StatusBadGateway, "error", i18n.T(lang, "doc.error_delete_source"), nil)
		case errors.Is(err, domain.ErrStorageProviderUnavailable):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_storage_provider"), nil)
		default:
			h.respondInternalError(c, lang, err)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", "Documentos da collection excluidos com sucesso.", gin.H{
		"collection":        dto.ToCollectionResponse(collection),
		"deleted_documents": deletedCount,
	})
}

func parseUintParam(c *gin.Context, paramName string, invalidMessage string) (uint, bool) {
	value := c.Param(paramName)
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "error", invalidMessage, nil)
		return 0, false
	}

	return uint(parsed), true
}

// respondInternalError devolve 500 ao cliente e REGISTRA a causa no log do
// servidor.
//
// A versao anterior nao recebia o erro: toda falha 5xx virava um 500 generico
// e a causa real morria ali. Na pratica isso significava que um upload
// quebrado por storage fora do ar aparecia no log de acesso apenas como
// `500 | POST /api/v1/documents/upload`, sem uma linha sequer explicando o
// motivo -- o operador ficava cego justamente no momento em que mais precisa
// enxergar. Para um sistema cuja tese e auditabilidade, engolir a causa do
// erro e o comportamento errado.
//
// A mensagem devolvida ao cliente continua generica de proposito: detalhe de
// infraestrutura nao vaza para fora, so vai para o log.
func (h *DocumentHandler) respondInternalError(c *gin.Context, lang string, err error) {
	if err != nil {
		log.Printf("erro interno em %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	}
	response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
}

func isUploadTooLargeError(err error) bool {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}

	errMessage := strings.ToLower(err.Error())
	return strings.Contains(errMessage, "request body too large") || strings.Contains(errMessage, "multipart: message too large")
}

func withDeletionAuditContext(c *gin.Context, userID uint) context.Context {
	return withRequestAuditContext(c, userID)
}
