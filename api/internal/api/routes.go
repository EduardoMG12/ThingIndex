package api

import (
	"fmt"
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

func SetupRoutes(router *gin.Engine, db *gorm.DB, cfg *config.Config) error {
	if err := router.SetTrustedProxies(cfg.TrustedProxyList()); err != nil {
		return fmt.Errorf("failed to configure trusted proxies: %w", err)
	}
	router.MaxMultipartMemory = cfg.MaxUploadSizeBytes
	router.Use(middleware.I18nMiddleware())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CorsAllowedOrigins(),
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
		return fmt.Errorf("failed to initialize minio storage: %w", err)
	}

	authRepo := repository.NewGormAuthRepository(db)
	categoryRepo := repository.NewGormCategoryRepository(db)
	collectionRepo := repository.NewGormCollectionRepository(db)
	documentRepo := repository.NewGormCatalogDocumentRepository(db)
	structuredDataRepo := repository.NewGormStructuredDataRepository(db)
	collectionContextRepo := repository.NewGormCollectionContextRepository(db)

	smtpMailer := mailer.NewSMTPMailer(cfg)

	authUseCase := usecase.NewAuthUseCase(authRepo, smtpMailer, cfg)

	authHandler := handlers.NewAuthHandler(authUseCase)

	providerRepo := repository.NewGormProviderRepository(db)
	providerUseCase := usecase.NewProviderUseCase(providerRepo, cfg)
	providerHandler := handlers.NewProviderHandler(providerUseCase)
	pricingRepo := repository.NewGormPricingRepository(db)
	pricingUseCase := usecase.NewPricingUseCase(pricingRepo)
	pricingHandler := handlers.NewPricingHandler(pricingUseCase)

	docUseCase := usecase.NewDocumentUseCase(
		categoryRepo,
		collectionRepo,
		documentRepo,
		structuredDataRepo,
		collectionContextRepo,
		minioStorage,
		providerUseCase,
		cfg.MaxUploadSizeBytes,
	)
	docHandler := handlers.NewDocumentHandler(docUseCase, cfg.UploadRequestLimitBytes())

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
		auth.Use(middleware.RateLimitMiddleware(cfg.AuthRateLimitRequests, cfg.AuthRateLimitWindow))
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
			user.PATCH("/providers/:providerType", providerHandler.UpdateProviderState)
			user.POST("/pricing-sources", pricingHandler.SaveSource)
			user.GET("/pricing-sources", pricingHandler.ListSources)
			user.GET("/prices", pricingHandler.ListPrices)
		}

		documents := v1.Group("/documents")
		documents.Use(middleware.AuthMiddleware(cfg))
		{
			documents.POST("/upload", docHandler.Upload)
			documents.POST("/collections", docHandler.CreateCollection)
			documents.GET("/collections", docHandler.ListCollections)
			documents.GET("/collections/:id/context", docHandler.GetCollectionContext)
			documents.GET("/collections/:id", docHandler.GetCollection)
			documents.PATCH("/collections/:collection/category", docHandler.UpdateCollectionCategory)
			documents.POST("/categories", docHandler.CreateCategory)
			documents.GET("/categories", docHandler.ListCategories)
			documents.PATCH("/categories/:id", docHandler.UpdateCategory)
			documents.GET("/category/:id", docHandler.ListDocumentsByCategory)
			documents.POST("/:id/paragraph-nodes", docHandler.StoreParagraphNodes)
			documents.POST("/:id/table-sections", docHandler.StoreTableSections)
			documents.POST("/:id/metadata-pairs", docHandler.StoreMetadataPairs)
			documents.GET("/:id/content", docHandler.GetDocumentContent)
			documents.GET("/:id/index", docHandler.GetDocumentIndex)
			documents.GET("/:id", docHandler.GetDocument)
			documents.GET("/", docHandler.ListDocuments)
		}
	}

	return nil
}
