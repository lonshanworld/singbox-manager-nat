package scheduler

import (
	"log"
	"time"

	"singbox-manager/internal/config"
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
	// Run once immediately on startup to sync state
	go func() {
		s.PollUsageOnce()
	}()
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

	settings, err := s.store.GetSettings()
	if err != nil {
		log.Printf("Scheduler: Failed to get settings: %v", err)
		return
	}

	needsSingboxRestart := false

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
				log.Printf("Scheduler: User %s expired on %s. Blocking.", u.Username, u.ExpireDate.Format("2006-01-02"))
				s.store.UpdateUser(u.Username, func(dbU *models.User) {
					dbU.Status = "blocked"
				})
				needsSingboxRestart = true
				continue
			}
		}

		// Get user usage from sing-box Clash API
		currentBytes, err := network.GetUserUsage(settings.ClashAPIAddress, settings.ClashAPISecret, u.Username)
		if err != nil {
			log.Printf("Scheduler: Failed to get user %s usage: %v", u.Username, err)
			continue // keep user active, try again next poll
		}

		// Calculate increment (handle sing-box restarts resetting counters)
		increment := int64(0)
		if currentBytes >= u.LastSeenBytes {
			increment = currentBytes - u.LastSeenBytes
		} else {
			// Counter reset (sing-box restarted) — count from 0
			increment = currentBytes
		}

		newUsedBytes := u.UsedBytes + increment
		limitBytes := int64(u.DataLimitGB) * 1024 * 1024 * 1024

		if newUsedBytes >= limitBytes {
			log.Printf("Scheduler: User %s reached data limit (%d GB). Blocking.", u.Username, u.DataLimitGB)
			s.store.UpdateUser(u.Username, func(dbU *models.User) {
				dbU.UsedBytes = newUsedBytes
				dbU.LastSeenBytes = currentBytes
				dbU.Status = "blocked"
			})
			needsSingboxRestart = true
		} else {
			s.store.UpdateUser(u.Username, func(dbU *models.User) {
				dbU.UsedBytes = newUsedBytes
				dbU.LastSeenBytes = currentBytes
			})
		}
	}

	if needsSingboxRestart {
		// Re-fetch from DB so config reflects the true committed state
		freshUsers, err := s.store.GetUsers()
		if err != nil {
			log.Printf("Scheduler: Failed to re-fetch users for config: %v", err)
			return
		}
		config.GenerateConfig(freshUsers, settings, "/etc/sing-box/config.json")
		system.RestartSingbox()
	}
}

// runMonthlyReset checks periodically if the current month has changed
func (s *Scheduler) runMonthlyReset() {
	// Check immediately on startup
	s.checkMonthlyReset()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.checkMonthlyReset()
	}
}

func (s *Scheduler) checkMonthlyReset() {
	now := time.Now()
	currentMonthStr := now.Format("2006-01") // e.g., "2023-11"

	setts, err := s.store.GetSettings()
	if err != nil {
		log.Printf("Scheduler: Monthly Reset check failed: %v", err)
		return
	}

	if setts.LastResetMonth == "" {
		// First run ever, set it to current month but don't reset everyone's 0 usage
		setts.LastResetMonth = currentMonthStr
		s.store.SaveSettings(setts)
		return
	}

	if setts.LastResetMonth != currentMonthStr {
		log.Printf("Scheduler: New month detected (%s), executing monthly reset...", currentMonthStr)
		s.PerformMonthlyReset()

		setts.LastResetMonth = currentMonthStr
		s.store.SaveSettings(setts)
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
		settings, err := s.store.GetSettings()
		if err != nil {
			log.Printf("Monthly Reset Error fetching settings: %v", err)
			return
		}
		
		if err := config.GenerateConfig(cleanUsers, settings, "/etc/sing-box/config.json"); err != nil {
			log.Printf("Monthly Reset failed to generate config: %v", err)
			return
		}
		system.RestartSingbox()
	}
}
