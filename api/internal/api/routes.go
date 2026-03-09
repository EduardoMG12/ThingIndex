package api

import (
	"net/http"
	"thing-index/internal/config"
	"thing-index/internal/handlers"
	"thing-index/internal/mailer"
	"thing-index/internal/repository"
	"thing-index/internal/usecase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB, cfg *config.Config) {
	authRepo := repository.NewGormAuthRepository(db)
	smtpMailer := mailer.NewSMTPMailer(cfg)
	authUseCase := usecase.NewAuthUseCase(authRepo, smtpMailer, cfg)
	authHandler := handlers.NewAuthHandler(authUseCase)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "API Thing-Index no ar! 🚀"})
		})

		// Auth Routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}
	}
}
