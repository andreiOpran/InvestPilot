package jobs

import (
	"log"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/robfig/cron/v3"
)

func StartRebalanceJob(rebalanceService services.RebalanceService) {
	c := cron.New()

	schedule := config.Env.RebalanceSchedule

	_, err := c.AddFunc(schedule, func() {
		log.Println("[CRON-JOB] Starting monthly batch rebalancing pipeline")

		if err := rebalanceService.RunMonthlyRebalance(); err != nil {
			log.Printf("[CRON-ERROR] Monthly rebalance execution failed: %v", err)
		} else {
			log.Println("[CRON-JOB] Monthly batch rebalancing completed successfully")
		}
	})

	if err != nil {
		log.Fatalf("[CRON-ERROR] Failed to schedule rebalance cron: %v", err)
	}

	c.Start()
	log.Println("[SYSTEM] Scheduled monthly rebalance cron job")
}
