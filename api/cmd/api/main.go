package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"thing-index/db"
	"thing-index/internal/api"
	"thing-index/internal/config"
)

// @title           Thing-Index API
// @version         1.0
// @description     Open-source HTTP API for ThingIndex covering auth, document cataloging, provider configuration, monitoring, and grounded chat.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Use "Bearer <token>".

// @host      localhost:8000
// @BasePath  /api/v1

func main() {
	config.Load()
	if err := config.AppConfig.Validate(); err != nil {
		log.Fatalf("Invalid application configuration: %v", err)
	}

	log.Println("Starting server on port " + config.AppConfig.Port + "...")

	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	log.Println("Connected to PostgreSQL successfully!")
	sqlDB, err := database.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	router := gin.Default()
	if err := api.SetupRoutes(router, database, config.AppConfig); err != nil {
		log.Fatalf("Failed to setup routes: %v", err)
	}

	server := &http.Server{
		Addr:         ":" + config.AppConfig.Port,
		Handler:      router,
		ReadTimeout:  config.AppConfig.ServerReadTimeout,
		WriteTimeout: config.AppConfig.ServerWriteTimeout,
		IdleTimeout:  config.AppConfig.ServerIdleTimeout,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting server: %v", err)
	}
}
