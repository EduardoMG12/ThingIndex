package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func I18nMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.GetHeader("Accept-Language")

		finalLang := "en"
		if strings.HasPrefix(strings.ToLower(lang), "pt") {
			finalLang = "pt_BR"
		}

		c.Set("language", finalLang)

		c.Next()
	}
}
