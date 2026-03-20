package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/models"
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
		h.respondInternalError(c, lang)
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
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "doc.error_upload"), nil)
		default:
			h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
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
		h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_collection_context_get"), dto.ToCollectionContextResponse(collectionContext))
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
			h.respondInternalError(c, lang)
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
		h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
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
		h.respondInternalError(c, lang)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_documents_list"), dto.ToDocumentsResponse(docs))
}

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

	events, err := h.useCase.ListDocumentAuditEvents(c.Request.Context(), userID, domain.DocumentAuditFilter{
		Limit:        limit,
		Action:       c.Query("action"),
		Status:       c.Query("status"),
		DocumentID:   documentID,
		CollectionID: collectionID,
	})
	if err != nil {
		h.respondInternalError(c, lang)
		return
	}

	response.Success(c, http.StatusOK, "success", "Auditoria documental carregada com sucesso.", dto.ToDocumentAuditEventResponses(events))
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
			h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_document_get"), dto.ToDocumentIndexResponse(index))
}

func (h *DocumentHandler) GetDocumentMonitoringSummary(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	summary, err := h.useCase.GetDocumentMonitoringSummary(c.Request.Context(), userID)
	if err != nil {
		h.respondInternalError(c, lang)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_documents_list"), dto.ToDocumentMonitoringSummaryResponse(summary))
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
		h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
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
			h.respondInternalError(c, lang)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", "Documentos da collection excluidos com sucesso.", gin.H{
		"collection":        dto.ToCollectionResponse(collection),
		"deleted_documents": deletedCount,
	})
}

func (h *DocumentHandler) StoreParagraphNodes(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	documentID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}

	var req dto.StoreParagraphNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	nodes := make([]models.ParagraphNode, 0, len(req.Nodes))
	for _, item := range req.Nodes {
		nodes = append(nodes, models.ParagraphNode{
			SequenceOrder: item.SequenceOrder,
			PageNumber:    item.PageNumber,
			Content:       strings.TrimSpace(item.Content),
			Metadata:      item.Metadata,
		})
	}

	stored, err := h.useCase.StoreParagraphNodes(c.Request.Context(), userID, documentID, nodes)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDocumentNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_document_not_found"), nil)
		default:
			h.respondInternalError(c, lang)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_paragraph_nodes_store"), dto.ToParagraphNodeResponses(stored))
}

func (h *DocumentHandler) StoreTableSections(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	documentID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}

	var req dto.StoreTableSectionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	sections := make([]models.TableSection, 0, len(req.Sections))
	for _, item := range req.Sections {
		sections = append(sections, models.TableSection{
			SequenceOrder: item.SequenceOrder,
			Title:         strings.TrimSpace(item.Title),
			Headers:       item.Headers,
			Rows:          item.Rows,
			Metadata:      item.Metadata,
		})
	}

	stored, err := h.useCase.StoreTableSections(c.Request.Context(), userID, documentID, sections)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDocumentNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_document_not_found"), nil)
		default:
			h.respondInternalError(c, lang)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_table_sections_store"), dto.ToTableSectionResponses(stored))
}

func (h *DocumentHandler) StoreMetadataPairs(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	documentID, ok := parseUintParam(c, "id", i18n.T(lang, "error.invalid_document_id"))
	if !ok {
		return
	}

	var req dto.StoreMetadataPairsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	pairs := make([]models.MetadataPair, 0, len(req.Pairs))
	for _, item := range req.Pairs {
		pairs = append(pairs, models.MetadataPair{
			SequenceOrder: item.SequenceOrder,
			Key:           strings.TrimSpace(item.Key),
			Value:         strings.TrimSpace(item.Value),
		})
	}

	stored, err := h.useCase.StoreMetadataPairs(c.Request.Context(), userID, documentID, pairs)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDocumentNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_document_not_found"), nil)
		default:
			h.respondInternalError(c, lang)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_metadata_pairs_store"), dto.ToMetadataPairResponses(stored))
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

func (h *DocumentHandler) respondInternalError(c *gin.Context, lang string) {
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
