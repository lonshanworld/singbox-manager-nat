package handlers

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"singbox-manager/internal/store"
)

type AuthHandler struct {
	db *store.Store
}

func NewAuthHandler(db *store.Store) *AuthHandler {
	return &AuthHandler{db: db}
}

func (h *AuthHandler) GetLogin(c fiber.Ctx) error {
	return c.Render("login", fiber.Map{
		"Error": "",
	})
}

func (h *AuthHandler) PostLogin(c fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	setts, err := h.db.GetSettings()
	if err != nil {
		log.Printf("Login failed to get settings: %v", err)
		return c.Render("login", fiber.Map{"Error": "Database error"})
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminHash := os.Getenv("ADMIN_PASSWORD_HASH")

	if adminUsername == "" || adminHash == "" {
		log.Printf("AUTH DEBUG: ADMIN_USERNAME or ADMIN_PASSWORD_HASH is EMPTY in environment!")
	}

	if username != adminUsername {
		log.Printf("AUTH DEBUG: Username mismatch. Got: '%s', Expected: '%s'", username, adminUsername)
		return c.Render("login", fiber.Map{"Error": "Invalid credentials"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(adminHash), []byte(password))
	if err != nil {
		log.Printf("AUTH DEBUG: Password check FAILED for user %s: %v", username, err)
		log.Printf("AUTH DEBUG: Hash being used: '%s' (len: %d)", adminHash, len(adminHash))
		return c.Render("login", fiber.Map{"Error": "Invalid credentials"})
	}

	log.Printf("AUTH DEBUG: Login successful for user: %s", username)

	// Generate a secure session token
	sessionToken := uuid.NewString()

	// Save to DB
	setts.AdminSession = sessionToken
	if saveErr := h.db.SaveSettings(setts); saveErr != nil {
		log.Printf("Failed to bind admin session to settings: %v", saveErr)
	}

	// Set session
	c.Cookie(&fiber.Cookie{
		Name:     "admin_session",
		Value:    sessionToken,
		HTTPOnly: true,
		Secure:   false, // Accessed via SSH tunnel over localhost, no HTTPS needed
		SameSite: "Strict",
		Path:     "/",
		MaxAge:   86400, // 24 hour session expiry
	})

	return c.Redirect().To("/dashboard")
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	setts, err := h.db.GetSettings()
	if err == nil {
		setts.AdminSession = ""
		h.db.SaveSettings(setts)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "admin_session",
		Value:    "",
		HTTPOnly: true,
		MaxAge:   -1, // Delete
	})
	return c.Redirect().To("/login")
}
