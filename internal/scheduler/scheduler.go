package scheduler

import (
	"log"
	"time"

	"singbox-manager/internal/config"
	"singbox-manager/internal/models"
	"singbox-manager/internal/network"
	"singbox-manager/internal/store"
	"singbox-manager/internal/system"
)

type Scheduler struct {
	store *store.Store
}

func NewScheduler(store *store.Store) *Scheduler {
	return &Scheduler{
		store: store,
	}
}

// Start begins background tasks
func (s *Scheduler) Start() {
	go s.runUsagePoller()
	go s.runMonthlyReset()
}

// runUsagePoller updates bytes used every 5 minutes and blocks users if limit exceeded
func (s *Scheduler) runUsagePoller() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.PollUsageOnce()
	}
}

// PollUsageOnce performs one run of usage verification, can be manually triggered
func (s *Scheduler) PollUsageOnce() {
	users, err := s.store.GetUsers()
	if err != nil {
		log.Printf("Scheduler: Failed to get users: %v", err)
		return
	}

	needsSingboxRestart := false
	var activeUsers []models.User // for config generation

	for _, u := range users {
		if u.Status != "active" {
			continue
		}

		// Check expiry
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if u.ExpireDate != nil {
			userExp := time.Date(u.ExpireDate.Year(), u.ExpireDate.Month(), u.ExpireDate.Day(), 0, 0, 0, 0, u.ExpireDate.Location())
			if today.After(userExp) {
				log.Printf("Scheduler: User %s expired on %s.", u.Username, u.ExpireDate.Format("2006-01-02"))
				u.Status = "blocked"
				needsSingboxRestart = true

				s.store.UpdateUser(u.Username, func(dbU *models.User) {
					dbU.Status = "blocked"
				})
				continue
			}
		}

		// Get user usage from sing-box Clash API
		settings, _ := s.store.GetSettings()
		currentBytes, err := network.GetUserUsage(settings.ClashAPIAddress, settings.ClashAPISecret, u.Username)
		if err != nil {
			log.Printf("Scheduler: Failed to get user %s usage: %v", u.Username, err)
			activeUsers = append(activeUsers, u)
			continue
		}

		// Calculate increment
		increment := int64(0)
		if currentBytes >= u.LastSeenBytes {
			increment = currentBytes - u.LastSeenBytes
		} else {
			// sing-box restarted, reset counter
			increment = currentBytes
		}

		u.UsedBytes += increment
		u.LastSeenBytes = currentBytes

		limitBytes := int64(u.DataLimitGB) * 1024 * 1024 * 1024

		if u.UsedBytes >= limitBytes {
			log.Printf("Scheduler: User %s reached data limit. Blocking user.", u.Username)
			u.Status = "blocked"
			needsSingboxRestart = true
		} else {
			activeUsers = append(activeUsers, u)
		}

		// Save updated user to DB
		s.store.UpdateUser(u.Username, func(dbU *models.User) {
			dbU.UsedBytes = u.UsedBytes
			dbU.LastSeenBytes = u.LastSeenBytes
			dbU.Status = u.Status
		})
	}

	if needsSingboxRestart {
		// Regenerate config without blocked users
		settings, _ := s.store.GetSettings()
		config.GenerateConfig(activeUsers, settings, "/etc/sing-box/config.json")
		system.RestartSingbox()
	}
}

// runMonthlyReset checks daily if it is the 1st day of the month
func (s *Scheduler) runMonthlyReset() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if time.Now().Day() == 1 {
			s.PerformMonthlyReset()
		}
	}
}

// PerformMonthlyReset zeroes bytes, unblocks users
func (s *Scheduler) PerformMonthlyReset() {
	log.Println("Scheduler: Executing monthly reset...")

	users, err := s.store.GetUsers()
	if err != nil {
		log.Printf("Monthly Reset Error: %v", err)
		return
	}

	needsSingboxRestart := false
	for _, u := range users {
		u.UsedBytes = 0
		if u.Status == "blocked" {
			u.Status = "active"
			needsSingboxRestart = true
		}

		s.store.UpdateUser(u.Username, func(dbU *models.User) {
			dbU.UsedBytes = 0
			dbU.Status = u.Status
		})
	}

	if needsSingboxRestart {
		// Re-fetch the clean list to generate config
		cleanUsers, _ := s.store.GetUsers()
		settings, _ := s.store.GetSettings()
		config.GenerateConfig(cleanUsers, settings, "/etc/sing-box/config.json")
		system.RestartSingbox()
	}
}
