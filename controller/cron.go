package controller

import (
	"fmt"
	"time"

	dao "github.com/mdhasib01/go-rest-starter/dao"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

// StartCronJob starts the background jobs. Every job must be idempotent:
// it runs at start-up and then every day.
func StartCronJob() error {
	if dao.DB == nil {
		return fmt.Errorf("database not initialised")
	}
	go func() {
		runDailyJobs()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			runDailyJobs()
		}
	}()
	return nil
}

func runDailyJobs() {
	defer func() {
		if r := recover(); r != nil {
			logger.GetLogger().LogErrors(fmt.Errorf("cron panic: %v", r), nil)
		}
	}()
	awardTopDonorsOfLastMonth(time.Now())
}

// awardTopDonorsOfLastMonth gives "Top Donor of the Month" (per blood
// bank) for the previous calendar month.
func awardTopDonorsOfLastMonth(now time.Time) {
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonth := firstOfThisMonth.AddDate(0, -1, 0)

	winners, err := dao.AwardTopDonorsOfMonth(lastMonth)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return
	}
	for _, w := range winners {
		notify([]int{w.UserId}, model.Notification{
			Kind:  model.NotifyBadgeEarned,
			Title: "Top Donor of " + lastMonth.Format("January 2006") + "!",
			Body:  "You earned the most points in your blood bank last month.",
			Data:  map[string]any{"badge": w.Badge, "blood_bank_id": w.BankId},
		})
	}
}
