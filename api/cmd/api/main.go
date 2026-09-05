package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

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

// shutdownTimeout limita a espera pelas requisicoes em andamento.
//
// Precisa ser MENOR que o prazo que o orquestrador da entre SIGTERM e SIGKILL
// (10s por padrao no Docker) -- se for maior, o processo e morto no meio do
// encerramento e a espera nao serve para nada.
const shutdownTimeout = 8 * time.Second

func main() {
	config.Load()
	if err := config.AppConfig.Validate(); err != nil {
		log.Fatalf("Invalid application configuration: %v", err)
	}

	// O indexer e o pricing-worker ja tratavam sinais; a API nao. Sem isso um
	// deploy derrubava requisicoes em andamento no meio -- inclusive uploads,
	// onde o cliente ja enviou o arquivo e nao recebe resposta, e a goroutine
	// de sync do catalogo de modelos morria abruptamente.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
	// O contexto de sinal e repassado para as tarefas de fundo registradas em
	// SetupRoutes: elas precisam encerrar junto, nao depender do processo
	// morrer para sumir.
	if err := api.SetupRoutesWithContext(ctx, router, database, config.AppConfig); err != nil {
		log.Fatalf("Failed to setup routes: %v", err)
	}

	server := &http.Server{
		Addr:         ":" + config.AppConfig.Port,
		Handler:      router,
		ReadTimeout:  config.AppConfig.ServerReadTimeout,
		WriteTimeout: config.AppConfig.ServerWriteTimeout,
		IdleTimeout:  config.AppConfig.ServerIdleTimeout,
	}

	// O servidor sobe em goroutine para o main poder esperar pelo sinal.
	// Falha de bind precisa derrubar o processo na hora: sem isso o container
	// ficaria "no ar" sem escutar em porta nenhuma.
	erroDoServidor := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erroDoServidor <- err
		}
	}()

	select {
	case err := <-erroDoServidor:
		log.Fatalf("Error starting server: %v", err)
	case <-ctx.Done():
		log.Println("Shutdown signal received, draining in-flight requests...")
	}

	// stop() antes do Shutdown: devolve o comportamento padrao ao sinal, de
	// modo que um segundo Ctrl+C mate o processo na hora em vez de ficar preso
	// esperando uma requisicao lenta.
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown timed out, forcing close: %v", err)
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("Error closing server: %v", closeErr)
		}
		return
	}

	log.Println("Server stopped cleanly.")
}
