package jobs

import (
	"log"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/models"

	"github.com/robfig/cron/v3"
)

// StartTokenCleanupJob configures and starts the cron for ExecuteTokenCleanup()
func StartTokenCleanupJob() {
	// init scheduler
	c := cron.New()

	// "0 3 * * *" - minute 0, hour 3, every day, every month, every day of the week
	_, err := c.AddFunc("0 3 * * *", ExecuteTokenCleanup)

	if err != nil {
		log.Fatalf("Error initializing CRON job: %v", err)
	}

	c.Start()
	log.Println("[SYSTEM] TokenCleanupJob scheduled successfully.")
}

// ExecuteTokenCleanup does garbage collection for expired tokens in the database, triggered everyday at 03:00 AM
func ExecuteTokenCleanup() {
	log.Println("[CRON JOB / MANUAL] Cleaning up expired tokens...")
	now := time.Now()

	// clean ActionTokens (few, so we use regular deletion)
	res1 := database.DB.Where("expires_at < ?", now).Delete(&models.ActionToken{})
	if res1.Error != nil {
		log.Printf("[CLEANUP] Error deleting ActionTokens: %v\n", res1.Error)
	} else if res1.RowsAffected > 0 {
		log.Printf("[CLEANUP] Deleted %d expired ActionTokens.\n", res1.RowsAffected)
	} else {
		log.Println("[CLEANUP] No expired ActionTokens found for deletion.")
	}

	// clean expired sessions using batching (big count of sessions, compared to the ActionTokens)
	var totalDeleted int64
	batchSize := config.Env.CleanupBatchSize

	for {
		// we can use DELETE w/ LIMIT, so we retrieve 1000 ids,
		// and then delete the sessions with ids that are in that set
		subQuery := database.DB.Table("sessions").Select("id").Where("expires_at < ?", now).Limit(batchSize)

		res2 := database.DB.Where("id IN (?)", subQuery).Delete(&models.Session{})

		if res2.Error != nil {
			log.Printf("[CLEANUP] Error deleting Sessions (Batch): %v\n", res2.Error)
			break
		}

		rowsAffected := res2.RowsAffected
		totalDeleted += rowsAffected

		// if we deleted less than 1000, means we are done
		if rowsAffected < int64(batchSize) {
			break
		}

		// sleep for a bit to let the db receive requests from the users
		time.Sleep(100 * time.Millisecond)
	}

	if totalDeleted > 0 {
		log.Printf("[CLEANUP] Deleted %d expired Sessions (Batch).\n", totalDeleted)
	} else {
		log.Println("[CLEANUP] No expired Sessions found for deletion.")
	}
}
