package main

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"licenta/go-node/internal/database"
	"licenta/go-node/internal/jobs"
	"licenta/go-node/internal/mailer"
	"licenta/go-node/internal/router"
)

func main() {
	database.InitDB()
	mailer.InitEmailer()
	jobs.StartTokenCleanupJob()

	r := gin.Default()

	fmt.Println("Operational Node (Go) starting on port 8080...")

	router.SetupRoutes(r)

	r.Run(":8080")
}
