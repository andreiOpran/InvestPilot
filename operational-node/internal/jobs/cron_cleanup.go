package jobs

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
)

// StartTokenCleanupJob configures and starts the cron for ExecuteTokenCleanup()
func StartTokenCleanupJob() {
	// init scheduler
	c := cron.New()
	schedule := config.Env.CleanupCronSchedule
	_, err := c.AddFunc(schedule, ExecuteTokenCleanup)

	if err != nil {
		log.Fatalf("Error initializing CRON job: %v", err)
	}

	c.Start()
	log.Println("[SYSTEM] TokenCleanupJob scheduled successfully.")
}

// ExecuteTokenCleanup does garbage collection for expired tokens in the database, triggered everyday at 03:00 AM
func ExecuteTokenCleanup() {
	log.Println("[CRON JOB] Cleaning up expired tokens...")
	now := time.Now()

	// init cleanup repository using the global db explicitly for the cron job
	repo := repositories.NewCleanupRepository(database.DB)

	// clean actiontokens (few, so we use regular deletion)
	rowsAffected, err := repo.DeleteExpiredActionTokens(now)
	if err != nil {
		log.Printf("[CLEANUP] Error deleting ActionTokens: %v\n", err)
	} else if rowsAffected > 0 {
		log.Printf("[CLEANUP] Deleted %d expired ActionTokens.\n", rowsAffected)
	} else {
		log.Println("[CLEANUP] No expired ActionTokens found for deletion.")
	}

	// clean expired sessions using batching (big count of sessions, compared to the actiontokens)
	var totalDeleted int64
	batchSize := config.Env.CleanupBatchSize

	for {
		// we use the repository which handles the subquery delete safely
		deleted, err := repo.DeleteExpiredSessionsBatch(now, batchSize)
		if err != nil {
			log.Printf("[CLEANUP] Error deleting Sessions (Batch): %v\n", err)
			break
		}

		totalDeleted += deleted

		// if we deleted less than the batch size, means we are done
		if deleted < int64(batchSize) {
			break
		}

		// sleep for a bit to let the db receive requests from the users
		time.Sleep(config.Env.CronBatchSleep)
	}

	if totalDeleted > 0 {
		log.Printf("[CLEANUP] Deleted %d expired Sessions (Batch).\n", totalDeleted)
	} else {
		log.Println("[CLEANUP] No expired Sessions found for deletion.")
	}

	// clean old login attempts using tracking retention duration
	retentionDate := now.Add(-time.Hour * 24 * time.Duration(config.Env.LoginAttemptRetentionDays))
	totalDeleted = 0

	for {
		deleted, err := repo.DeleteOldLoginAttemptsBatch(retentionDate, batchSize)
		if err != nil {
			log.Printf("[CLEANUP] Error deleting LoginAttempts (Batch): %v\n", err)
			break
		}

		totalDeleted += deleted

		// if we deleted less than the batch size, means we are done
		if deleted < int64(batchSize) {
			break
		}

		// sleep for a bit to let the db receive requests from the users
		time.Sleep(config.Env.CronBatchSleep)
	}

	if totalDeleted > 0 {
		log.Printf("[CLEANUP] Deleted %d expired LoginAttempts.\n", totalDeleted)
	} else {
		log.Println("[CLEANUP] No expired LoginAttempts found for deletion.")
	}
}
