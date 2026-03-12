package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/validation"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	useCase domain.DocumentUseCase
}

func NewDocumentHandler(useCase domain.DocumentUseCase) *DocumentHandler {
	return &DocumentHandler{useCase: useCase}
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	title := c.PostForm("title")
	categoryName := strings.TrimSpace(c.PostForm("category"))
	categoryDesc := ""
	if categoryName != "" {
		categoryDesc = i18n.T(lang, "doc.auto_category_desc")
	}
	collectionName := strings.TrimSpace(c.PostForm("collection"))

	if title == "" {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_title"), nil)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_file"), err.Error())
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
		return
	}
	defer file.Close()

	doc, err := h.useCase.UploadDocument(
		c.Request.Context(),
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
		errorKey := "doc.error_upload"
		status := http.StatusInternalServerError

		if errors.Is(err, domain.ErrCategoryCreate) || errors.Is(err, domain.ErrCategoryFind) {
			errorKey = "doc.error_upload"
		} else if errors.Is(err, domain.ErrUploadFailed) {
			errorKey = "doc.error_upload"
		} else if errors.Is(err, domain.ErrInternal) {
			errorKey = "error.internal"
		}

		response.Error(c, status, "error", i18n.T(lang, errorKey), err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "success", i18n.T(lang, "doc.success_upload"), doc)
}

func (h *DocumentHandler) ListCollections(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	collections, err := h.useCase.ListCollections(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
		return
	}

	response.Success(c, http.StatusOK, "success", "Collections retrieved successfully", collections)
}

func (h *DocumentHandler) ListCategories(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	categories, err := h.useCase.ListCategories(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
		return
	}

	response.Success(c, http.StatusOK, "success", "Categories retrieved successfully", categories)
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
			return
		case errors.Is(err, domain.ErrCategoryDuplicate):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_category_duplicate"), nil)
			return
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
			return
		}
	}

	response.Success(c, http.StatusCreated, "success", i18n.T(lang, "doc.success_category_create"), category)
}

func (h *DocumentHandler) UpdateCategory(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	categoryIDStr := c.Param("id")

	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "error", "Invalid category ID", nil)
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

	category, err := h.useCase.UpdateCategory(c.Request.Context(), userID, uint(categoryID), req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCategoryNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_category_not_found"), nil)
			return
		case errors.Is(err, domain.ErrCategoryRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_category_required"), nil)
			return
		case errors.Is(err, domain.ErrCategoryDuplicate):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "doc.error_category_duplicate"), nil)
			return
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
			return
		}
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_category_update"), category)
}

func (h *DocumentHandler) UpdateCollectionCategory(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	collection := c.Param("collection")

	var req dto.UpdateCollectionCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	rowsAffected, category, err := h.useCase.UpdateCollectionCategory(
		c.Request.Context(),
		userID,
		collection,
		req.Category,
		i18n.T(lang, "doc.auto_category_desc"),
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCollectionRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "doc.error_collection_required"), nil)
			return
		case errors.Is(err, domain.ErrCollectionNotFound):
			response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "doc.error_collection_not_found"), nil)
			return
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
			return
		}
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "doc.success_collection_category_update"), gin.H{
		"collection":        strings.TrimSpace(collection),
		"category":          category,
		"updated_documents": rowsAffected,
	})
}

func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	docs, err := h.useCase.ListDocuments(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
		return
	}

	response.Success(c, http.StatusOK, "success", "Documents retrieved successfully", docs)
}

func (h *DocumentHandler) ListDocumentsByCategory(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")
	categoryIDStr := c.Param("id")

	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "error", "Invalid category ID", nil)
		return
	}

	docs, err := h.useCase.ListDocumentsByCategory(c.Request.Context(), uint(categoryID), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), err.Error())
		return
	}

	response.Success(c, http.StatusOK, "success", "Documents retrieved successfully", docs)
}
