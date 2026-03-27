package response

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Status  int         `json:"status"`
	Type    string      `json:"type"` // success, error, warning, info
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

func Success(c *gin.Context, status int, responseType string, message string, data interface{}) {
	c.JSON(status, Response{
		Status:  status,
		Type:    responseType,
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, status int, responseType string, message string, errors interface{}) {
	c.JSON(status, Response{
		Status:  status,
		Type:    responseType,
		Message: message,
		Errors:  errors,
	})
}
