package handlers

import (
	"net/http"
	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type MetaHandler struct{}

func NewMetaHandler() *MetaHandler {
	return &MetaHandler{}
}

// @Summary      Get runtime profile
// @Description  Returns the current primary use case, scope, explicit non-goals, and academic/reference artifacts for the main ThingIndex runtime.
// @Tags         meta
// @Produce      json
// @Success      200 {object} response.Response{data=dto.RuntimeProfileResponse}
// @Router       /meta/runtime-profile [get]
func (h *MetaHandler) GetRuntimeProfile(c *gin.Context) {
	lang := c.GetString("language")
	response.Success(
		c,
		http.StatusOK,
		"success",
		i18n.T(lang, "meta.success_runtime_profile"),
		dto.ToRuntimeProfileResponse(domain.CurrentRuntimeProfile()),
	)
}
