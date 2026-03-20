package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"thing-index/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func parseOptionalUintQuery(c *gin.Context, queryName string, invalidMessage string) (*uint, bool) {
	value := strings.TrimSpace(c.Query(queryName))
	if value == "" {
		return nil, true
	}

	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "error", invalidMessage, nil)
		return nil, false
	}

	result := uint(parsed)
	return &result, true
}

func parseOptionalLimitQuery(c *gin.Context, defaultLimit int, invalidMessage string) (int, bool) {
	value := strings.TrimSpace(c.Query("limit"))
	if value == "" {
		return defaultLimit, true
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		response.Error(c, http.StatusBadRequest, "error", invalidMessage, nil)
		return 0, false
	}

	if parsed > 100 {
		parsed = 100
	}

	return parsed, true
}
