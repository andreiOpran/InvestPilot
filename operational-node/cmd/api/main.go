package main

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/jobs"
	"github.com/andreiOpran/licenta/operational-node/internal/mailer"
	"github.com/andreiOpran/licenta/operational-node/internal/router"
)

func main() {
	config.LoadConfig()
	database.InitDB()
	mailer.InitEmailer()
	jobs.StartTokenCleanupJob()

	r := gin.Default()

	fmt.Println("Operational Node (Go) starting on port 8080...")

	router.SetupRoutes(r)

	r.Run(":8080")
}
