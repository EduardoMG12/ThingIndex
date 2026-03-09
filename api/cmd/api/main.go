package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"thing-index/db"
	"thing-index/internal/api"
	"thing-index/internal/config"
)

// @title           Thing-Index API
// @version         1.0
// @description     API for managing tokens and AI data processing.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8000
// @BasePath  /api/v1

func main() {
	config.Load()

	log.Println("Starting server on port " + config.AppConfig.Port + "...")

	database, err := db.Connect()
	if err != nil {
		log.Printf("Warning: Fail to connect to database, continuing without DB... %v\n", err)
	} else {
		log.Println("Connected to PostgreSQL successfully!")
		sqlDB, err := database.DB()
		if err == nil {
			defer sqlDB.Close()
		}
	}

	router := gin.Default()
	
	api.SetupRoutes(router, database, config.AppConfig)

	if err := router.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}