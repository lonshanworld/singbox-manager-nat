package routes

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"singbox-manager/internal/app/handlers"
	"singbox-manager/internal/app/middlewares"
	"singbox-manager/internal/store"
)

func SetupRoutes(app *fiber.App, db *store.Store) {
	// Middleware
	app.Use(middlewares.AuthMiddleware(db))
	// CSRF not needed: panel is bound to 127.0.0.1, only accessible via SSH tunnel

	authH := handlers.NewAuthHandler(db)
	dashH := handlers.NewDashboardHandler(db)
	userH := handlers.NewUserHandler(db)
	sysH := handlers.NewSystemHandler()

	app.Get("/", func(c fiber.Ctx) error {
		return c.Redirect().To("/dashboard")
	})

	app.Get("/login", authH.GetLogin)
	app.Post("/login", limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
	}), authH.PostLogin)
	app.Post("/logout", authH.Logout)

	app.Get("/dashboard", dashH.GetDashboard)

	app.Post("/users/add", userH.PostAddUser)
	app.Post("/users/delete", userH.PostDeleteUser)
	app.Post("/users/reset", userH.PostResetUsage)
	app.Post("/users/activate", userH.PostReactivateUser)

	// Public link (bypasses auth per middleware logic)
	app.Get("/subscription/:username", userH.GetSubscription)

	app.Post("/system/restart_singbox", sysH.PostRestartSingbox)
	app.Post("/system/start_singbox", sysH.PostStartSingbox)
	app.Post("/system/stop_singbox", sysH.PostStopSingbox)

	app.Get("/static/*", func(c fiber.Ctx) error {
		path := c.Params("*")
		// Prevent directory traversal
		clean := filepath.Clean(path)
		if strings.Contains(clean, "..") {
			return c.Status(403).SendString("Forbidden")
		}
		return c.SendFile("./static/" + clean)
	})
}
