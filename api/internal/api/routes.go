package api

import (
	"net/http"
	"thing-index/internal/api/middleware"
	"thing-index/internal/config"
	"thing-index/internal/handlers"
	"thing-index/internal/mailer"
	"thing-index/internal/pkg/response"
	"thing-index/internal/repository"
	"thing-index/internal/usecase"

	_ "thing-index/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB, cfg *config.Config) {
	// Global Middlewares
	router.Use(middleware.I18nMiddleware())

	// CORS Middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	authRepo := repository.NewGormAuthRepository(db)
	smtpMailer := mailer.NewSMTPMailer(cfg)
	authUseCase := usecase.NewAuthUseCase(authRepo, smtpMailer, cfg)
	authHandler := handlers.NewAuthHandler(authUseCase)

	// Scalar API Documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/docs", func(c *gin.Context) {
		html := `
		<!doctype html>
		<html>
		  <head>
		    <title>Thing-Index API Reference</title>
		    <meta charset="utf-8" />
		    <meta name="viewport" content="width=device-width, initial-scale=1" />
		    <style>
		      body { margin: 0; }
		    </style>
		  </head>
		  <body>
		    <script id="api-reference" data-url="/swagger/doc.json"></script>
		    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
		  </body>
		</html>`
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			response.Success(c, http.StatusOK, "success", "API Thing-Index no ar! 🚀", nil)
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
