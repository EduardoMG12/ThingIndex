package handlers

import (
	"errors"
	"net/http"

	"thing-index/internal/domain"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"

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
	title := c.PostForm("title")
	categoryName := c.PostForm("category")
	var categoryDesc string

	if categoryName == "" {
		categoryName = i18n.T(lang, "doc.default_category_name")
		categoryDesc = i18n.T(lang, "doc.default_category_description")
	} else {
		categoryDesc = i18n.T(lang, "doc.auto_category_desc")
	}

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
		title, 
		file, 
		fileHeader.Size, 
		fileHeader.Header.Get("Content-Type"), 
		categoryName,
		categoryDesc,
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
