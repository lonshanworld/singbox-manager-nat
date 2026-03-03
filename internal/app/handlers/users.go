package handlers

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"singbox-manager/internal/config"
	"singbox-manager/internal/models"
	"singbox-manager/internal/store"
	"singbox-manager/internal/system"
)

type UserHandler struct {
	db *store.Store
}

func NewUserHandler(db *store.Store) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) PostAddUser(c fiber.Ctx) error {
	username := c.FormValue("username")
	dataStr := c.FormValue("data_limit_gb")
	expireDateStr := c.FormValue("expire_date")

	if username == "" || dataStr == "" {
		return c.Status(400).SendString("All fields are required")
	}

	dataLimit, err := strconv.Atoi(dataStr)
	if err != nil || dataLimit <= 0 {
		return c.Status(400).SendString("Invalid data limit")
	}

	// Parse expiration date if provided
	var expireDate *time.Time
	if expireDateStr != "" {
		t, err := time.Parse("2006-01-02", expireDateStr)
		if err == nil {
			expireDate = &t
		}
	}

	// Create user model
	newUser := models.User{
		Username:       username,
		UUID:           uuid.NewString(),
		Port:           2371,
		SpeedLimitMbps: 0,
		DataLimitGB:    dataLimit,
		UsedBytes:      0,
		Status:         "active",
		CreatedAt:      time.Now(),
		ExpireDate:     expireDate,
	}

	// 1. Save to database
	if err := h.db.AddUser(newUser); err != nil {
		return c.Status(400).SendString(fmt.Sprintf("Failed to add user: %v", err))
	}

	// 2. Regenerate Config and Restart Sing-box
	users, _ := h.db.GetUsers()
	setts, _ := h.db.GetSettings()
	if err := config.GenerateConfig(users, setts, "/etc/sing-box/config.json"); err != nil {
		log.Printf("Failed to generate config: %v", err)
		return c.Status(500).SendString("Failed to generate config")
	}
	system.RestartSingbox()

	return c.Redirect().To("/dashboard")
}

func (h *UserHandler) PostDeleteUser(c fiber.Ctx) error {
	username := c.FormValue("username")
	
	// Delete from Store
	if err := h.db.DeleteUser(username); err != nil {
		return c.Status(400).SendString(fmt.Sprintf("Failed to delete user: %v", err))
	}

	// Regenerate Config and Restart Sing-box
	newUsers, _ := h.db.GetUsers()
	setts, _ := h.db.GetSettings()
	config.GenerateConfig(newUsers, setts, "/etc/sing-box/config.json")
	system.RestartSingbox()

	return c.Redirect().To("/dashboard")
}

func (h *UserHandler) PostResetUsage(c fiber.Ctx) error {
	username := c.FormValue("username")
	
	err := h.db.UpdateUser(username, func(u *models.User) {
		u.UsedBytes = 0
		if u.Status == "blocked" {
			u.Status = "active"
		}
	})

	if err != nil {
		return c.Status(400).SendString(fmt.Sprintf("Failed to reset usage: %v", err))
	}

	// Regenerate Config and Restart Sing-box
	users, _ := h.db.GetUsers()
	setts, _ := h.db.GetSettings()
	config.GenerateConfig(users, setts, "/etc/sing-box/config.json")
	system.RestartSingbox()

	return c.Redirect().To("/dashboard")
}

func (h *UserHandler) PostReactivateUser(c fiber.Ctx) error {
	username := c.FormValue("username")

	err := h.db.UpdateUser(username, func(u *models.User) {
		u.Status = "active"
	})

	if err != nil {
		return c.Status(400).SendString(fmt.Sprintf("Failed to reactivate user: %v", err))
	}

	// Regenerate Config and Restart Sing-box
	users, _ := h.db.GetUsers()
	setts, _ := h.db.GetSettings()
	config.GenerateConfig(users, setts, "/etc/sing-box/config.json")
	system.RestartSingbox()

	return c.Redirect().To("/dashboard")
}

func (h *UserHandler) GetSubscription(c fiber.Ctx) error {
	username := c.Params("username")

	users, err := h.db.GetUsers()
	if err != nil {
		return c.Status(500).SendString("Database error")
	}

	var targetUser *models.User
	for i, u := range users {
		if u.Username == username {
			targetUser = &users[i]
			break
		}
	}

	if targetUser == nil {
		return c.Status(404).SendString("User not found")
	}

	setts, _ := h.db.GetSettings()
	
	// Add Hiddify-compatible subscription-userinfo header
	// spec: upload=<bytes>; download=<bytes>; total=<bytes>; expire=<unix_timestamp>
	totalBytes := int64(targetUser.DataLimitGB) * 1024 * 1024 * 1024
	usageHeader := fmt.Sprintf("upload=0; download=%d; total=%d", targetUser.UsedBytes, totalBytes)
	if targetUser.ExpireDate != nil {
		usageHeader += fmt.Sprintf("; expire=%d", targetUser.ExpireDate.Unix())
	}
	c.Set("Subscription-Userinfo", usageHeader)

	// Extract SNI from RealityDest (host:port)
	sni := setts.RealityDest
	if host, _, err := net.SplitHostPort(setts.RealityDest); err == nil {
		sni = host
	}

	// Template: vless://UUID@SERVER_IP:PORT?security=reality&type=tcp&pbk=PUBLIC_KEY&fp=chrome&sni=SNI&sid=SHORTID&encryption=none#USERNAME
	link := fmt.Sprintf("vless://%s@%s:%d?security=reality&type=tcp&pbk=%s&fp=chrome&sni=%s&sid=%s&encryption=none#%s",
		targetUser.UUID,
		setts.ServerAddress,
		targetUser.Port,
		setts.PublicKey,
		sni,
		setts.ShortID,
		targetUser.Username,
	)

	return c.SendString(link)
}
