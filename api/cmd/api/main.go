package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"thing-index/db"
	"thing-index/internal/api"
	"thing-index/internal/config"
)

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
	
	api.SetupRoutes(router)

	if err := router.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}