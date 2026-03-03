package handlers

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"singbox-manager/internal/network"
	"singbox-manager/internal/store"
)

type DashboardHandler struct {
	db *store.Store
}

func NewDashboardHandler(db *store.Store) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) GetDashboard(c fiber.Ctx) error {
	users, err := h.db.GetUsers()
	if err != nil {
		return c.Status(500).SendString("Error fetching users")
	}

	total := len(users)
	active := 0
	blocked := 0

	for _, u := range users {
		if u.Status == "active" {
			active++
		} else {
			blocked++
		}
	}


	settings, _ := h.db.GetSettings()
	up, down, err := network.GetTraffic(settings.ClashAPIAddress, settings.ClashAPISecret)
	if err != nil {
		// Log error but don't fail, maybe sing-box is down
		up = 0
		down = 0
	}

	subDomain := settings.SubscriptionDomain
	if subDomain == "" {
		subDomain = fmt.Sprintf("%s:%d", settings.ServerAddress, settings.ManagerPort)
	}

	return c.Render("dashboard", fiber.Map{
		"TotalUsers":         total,
		"ActiveUsers":        active,
		"BlockedUsers":       blocked,
		"ServerIP":           settings.ServerAddress,
		"ManagerPort":        settings.ManagerPort,
		"SubscriptionDomain": subDomain,
		"SingboxRunning":     true, // Simplification
		"TrafficUp":          up,
		"TrafficDown":        down,
		"Users":              users,
	})
}
