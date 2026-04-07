package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/clients"
	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/jobs"
	"github.com/andreiOpran/licenta/operational-node/internal/mailer"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
	"github.com/andreiOpran/licenta/operational-node/internal/router"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

func main() {
	rebalanceRepo := repositories.NewRebalanceRepository(database.DB)
	userRepo := repositories.NewUserRepository(database.DB)

	rebalanceService := services.NewRebalanceService(rebalanceRepo, userRepo)

	config.LoadConfig()
	database.InitDB()
	mailer.InitEmailer()
	clients.InitRabbitMQ()
	jobs.StartTokenCleanupJob()
	jobs.StartDataPipelineJob()
	jobs.StartRebalanceJob(rebalanceService)

	r := gin.Default()
	router.SetupRoutes(r)

	fmt.Println("Operational Node (Go) starting on port " + config.Env.ServerPort + "...")

	// setup http server
	srv := &http.Server{
		Addr:    ":" + config.Env.ServerPort,
		Handler: r,
	}

	// run server in goroutine so it does not block
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// wait for interrupt signal to gracefull shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 5 seconds timeou for existing conns to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
