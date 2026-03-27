package handlers

import (
	"net/http"
	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type MetaHandler struct {
	benchmarkProvider domain.IndexingBenchmarkProvider
}

func NewMetaHandler(benchmarkProvider domain.IndexingBenchmarkProvider) *MetaHandler {
	return &MetaHandler{
		benchmarkProvider: benchmarkProvider,
	}
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

// @Summary      Get indexing benchmark baseline
// @Description  Returns the controlled corpus, exact-match benchmark metrics, and interpretation guidance for the current relational indexing baseline.
// @Tags         meta
// @Produce      json
// @Success      200 {object} response.Response{data=dto.IndexingBenchmarkReportResponse}
// @Failure      500 {object} response.Response
// @Router       /meta/indexing-benchmark [get]
func (h *MetaHandler) GetIndexingBenchmark(c *gin.Context) {
	lang := c.GetString("language")
	if h.benchmarkProvider == nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	report, err := h.benchmarkProvider.GetPublicIndexingBenchmark(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(
		c,
		http.StatusOK,
		"success",
		i18n.T(lang, "meta.success_indexing_benchmark"),
		dto.ToIndexingBenchmarkReportResponse(report),
	)
}
