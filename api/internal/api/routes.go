package api

import (
	"net/http"
	"thing-index/internal/api/middleware"
	"thing-index/internal/config"
	"thing-index/internal/handlers"
	"thing-index/internal/mailer"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/storage"
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
	router.Use(middleware.I18nMiddleware())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	minioStorage, err := storage.NewMinioStorage(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucketName,
		cfg.MinioUseSSL == "true",
	)
	if err != nil {
		panic("Failed to initialize MinIO Storage: " + err.Error())
	}

	authRepo := repository.NewGormAuthRepository(db)
	docRepo := repository.NewGormDocumentRepository(db)

	smtpMailer := mailer.NewSMTPMailer(cfg)

	authUseCase := usecase.NewAuthUseCase(authRepo, smtpMailer, cfg)

	authHandler := handlers.NewAuthHandler(authUseCase)

	providerRepo := repository.NewGormProviderRepository(db)
	providerUseCase := usecase.NewProviderUseCase(providerRepo, cfg)
	providerHandler := handlers.NewProviderHandler(providerUseCase)

	docUseCase := usecase.NewDocumentUseCase(docRepo, minioStorage, providerUseCase)
	docHandler := handlers.NewDocumentHandler(docUseCase)

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

		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}

		user := v1.Group("/user")
		user.Use(middleware.AuthMiddleware(cfg))
		{
			user.POST("/providers", providerHandler.SaveCredentials)
			user.GET("/providers", providerHandler.ListProviders)
		}

		documents := v1.Group("/documents")
		documents.Use(middleware.AuthMiddleware(cfg))
		{
			documents.POST("/upload", docHandler.Upload)
			documents.POST("/categories", docHandler.CreateCategory)
			documents.GET("/categories", docHandler.ListCategories)
			documents.GET("/collections", docHandler.ListCollections)
			documents.PATCH("/categories/:id", docHandler.UpdateCategory)
			documents.GET("/", docHandler.ListDocuments)
			documents.GET("/category/:id", docHandler.ListDocumentsByCategory)
		}
	}
}
