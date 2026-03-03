package middlewares

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
	"singbox-manager/internal/store"
)

func AuthMiddleware(db *store.Store) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Public paths
		path := c.Path()
		if path == "/login" || strings.HasPrefix(path, "/subscription/") || strings.HasPrefix(path, "/static/") {
			return c.Next()
		}

		// Check for valid session cookie
		cookie := c.Cookies("admin_session")
		setts, err := db.GetSettings()
		if err != nil {
			log.Printf("AuthMiddleware: Failed to get settings: %v", err)
			return c.Redirect().To("/login")
		}

		// Require a cryptographically secure matching session token
		if setts.AdminSession == "" || cookie != setts.AdminSession {
			return c.Redirect().To("/login")
		}

		return c.Next()
	}
}
