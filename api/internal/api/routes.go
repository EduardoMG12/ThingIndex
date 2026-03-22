package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"thing-index/internal/api/middleware"
	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/handlers"
	"thing-index/internal/mailer"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/storage"
	"thing-index/internal/repository"
	"thing-index/internal/usecase"
	"time"

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
	router.Use(middleware.RequestContextMiddleware())
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
	chatConversationRepo := repository.NewGormChatConversationRepository(db)

	smtpMailer := mailer.NewSMTPMailer(cfg)

	authUseCase := usecase.NewAuthUseCase(authRepo, smtpMailer, cfg)

	authHandler := handlers.NewAuthHandler(authUseCase)
	metaHandler := handlers.NewMetaHandler()

	providerRepo := repository.NewGormProviderRepository(db)
	providerUseCase := usecase.NewProviderUseCase(providerRepo, cfg)
	providerHandler := handlers.NewProviderHandler(providerUseCase)
	startProviderModelCatalogSync(context.Background(), providerUseCase, cfg.ModelCatalogSyncInterval)
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
	chatUseCase := usecase.NewChatUseCase(collectionContextRepo, chatConversationRepo, providerUseCase)
	chatHandler := handlers.NewChatHandler(chatUseCase)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/docs", func(c *gin.Context) {
		html, err := renderAPIReferencePage(buildAPIReferencePageData(c.Request))
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to build API reference page")
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", html)
	})

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			response.Success(c, http.StatusOK, "success", "API Thing-Index no ar! 🚀", nil)
		})
		v1.GET("/meta/runtime-profile", metaHandler.GetRuntimeProfile)
		v1.GET("/providers/support", providerHandler.ListSupportMatrix)

		auth := v1.Group("/auth")
		auth.Use(middleware.RateLimitMiddleware(cfg.AuthRateLimitRequests, cfg.AuthRateLimitWindow))
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}

		user := v1.Group("/user")
		user.Use(middleware.AuthMiddleware(cfg, authRepo))
		{
			user.POST("/providers", providerHandler.SaveCredentials)
			user.POST("/providers/test", providerHandler.TestConnection)
			user.POST("/providers/models", providerHandler.ListModels)
			user.GET("/providers", providerHandler.ListProviders)
			user.GET("/providers/runtime", providerHandler.ListRuntimeOverview)
			user.GET("/providers/audit", providerHandler.ListProviderAuditEvents)
			user.PATCH("/providers/:providerType", providerHandler.UpdateProviderState)
			user.POST("/pricing-sources", pricingHandler.SaveSource)
			user.GET("/pricing-sources", pricingHandler.ListSources)
			user.GET("/prices", pricingHandler.ListPrices)
			user.GET("/prices/history", pricingHandler.ListPriceHistory)
		}

		documents := v1.Group("/documents")
		documents.Use(middleware.AuthMiddleware(cfg, authRepo))
		{
			documents.POST("/upload", docHandler.Upload)
			documents.GET("/audit", docHandler.ListDocumentAuditEvents)
			documents.POST("/collections", docHandler.CreateCollection)
			documents.GET("/collections", docHandler.ListCollections)
			documents.GET("/collections/:id/context", docHandler.GetCollectionContext)
			documents.GET("/collections/:id", docHandler.GetCollection)
			documents.DELETE("/collections/:id/documents", docHandler.DeleteCollectionDocuments)
			documents.PATCH("/collections/:collection/category", docHandler.UpdateCollectionCategory)
			documents.POST("/categories", docHandler.CreateCategory)
			documents.GET("/categories", docHandler.ListCategories)
			documents.PATCH("/categories/:id", docHandler.UpdateCategory)
			documents.GET("/category/:id", docHandler.ListDocumentsByCategory)
			documents.POST("/:id/paragraph-nodes", docHandler.StoreParagraphNodes)
			documents.POST("/:id/table-sections", docHandler.StoreTableSections)
			documents.POST("/:id/metadata-pairs", docHandler.StoreMetadataPairs)
			documents.GET("/:id/content", docHandler.GetDocumentContent)
			documents.POST("/:id/retry", docHandler.RetryIndexing)
			documents.DELETE("/:id", docHandler.DeleteDocument)
			documents.GET("/monitoring/summary", docHandler.GetDocumentMonitoringSummary)
			documents.GET("/:id/index", docHandler.GetDocumentIndex)
			documents.GET("/:id", docHandler.GetDocument)
			documents.GET("/", docHandler.ListDocuments)
		}

		chat := v1.Group("/chat")
		chat.Use(middleware.AuthMiddleware(cfg, authRepo))
		{
			chat.POST("/conversations", chatHandler.CreateConversation)
			chat.GET("/conversations", chatHandler.ListConversations)
			chat.GET("/conversations/:id", chatHandler.GetConversation)
			chat.PATCH("/conversations/:id", chatHandler.UpdateConversation)
			chat.DELETE("/conversations/:id", chatHandler.DeleteConversation)
			chat.POST("/stream", chatHandler.StreamCollectionChat)
		}
	}

	return nil
}

func startProviderModelCatalogSync(ctx context.Context, providerUseCase domain.ProviderUseCase, interval time.Duration) {
	if interval <= 0 {
		return
	}

	go func() {
		runSync := func() {
			syncCtx, cancel := context.WithTimeout(ctx, interval)
			defer cancel()

			if err := providerUseCase.SyncModelCatalogs(syncCtx); err != nil {
				log.Printf("provider model catalog sync failed: %v", err)
			}
		}

		runSync()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runSync()
			}
		}
	}()
}
