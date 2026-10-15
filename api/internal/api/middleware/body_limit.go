package middleware

import (
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
)

const JSONBodyLimitBytes int64 = 1 << 20

// JSONBodyLimitMiddleware limita o corpo das requisicoes que nao sao
// multipart; upload e avatar tem limite proprio, maior.
func JSONBodyLimitMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && !isMultipart(c.Request) {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

func isMultipart(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "multipart/form-data"
}
