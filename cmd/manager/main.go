package main

import (
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v3"

	"singbox-manager/internal/app/routes"
	"singbox-manager/internal/auth"
	"singbox-manager/internal/scheduler"
	"singbox-manager/internal/store"
	"singbox-manager/internal/telegram"
)

func main() {
	// Initialize Store
	db := store.NewStore("/etc/singbox-manager")
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer db.Close()

	// Load environment variables
	wd, _ := os.Getwd()
	log.Printf("Starting in directory: %s", wd)
	
	if err := auth.LoadEnv(".env"); err != nil {
		log.Printf("Warning: Failed to load .env file from %s: %v", wd, err)
	} else {
		log.Printf("Successfully loaded .env file from: %s/.env", wd)
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminHash := os.Getenv("ADMIN_PASSWORD_HASH")
	
	log.Printf("ENV DEBUG: Loaded ADMIN_USERNAME='%s' (len: %d)", adminUsername, len(adminUsername))
	
	if adminUsername == "" || adminHash == "" {
		log.Fatal("CRITICAL: ADMIN_USERNAME and ADMIN_PASSWORD_HASH must be set in .env")
	}

	// Load settings from DB
	settings, err := db.GetSettings()
	if err != nil {
		log.Fatalf("Failed to load settings: %v", err)
	}

	// Sync .env settings to DB if provided
	if val := os.Getenv("REALITY_PUBLIC_KEY"); val != "" {
		settings.PublicKey = val
	}
	if val := os.Getenv("REALITY_PRIVATE_KEY"); val != "" {
		settings.PrivateKey = val
	}
	if val := os.Getenv("REALITY_SHORT_ID"); val != "" {
		settings.ShortID = val
	}
	if val := os.Getenv("SERVER_ADDRESS"); val != "" {
		settings.ServerAddress = val
	}
	if val := os.Getenv("CLASH_API_ADDRESS"); val != "" {
		settings.ClashAPIAddress = val
	}
	if val := os.Getenv("CLASH_API_SECRET"); val != "" {
		settings.ClashAPISecret = val
	}
	if val := os.Getenv("REALITY_DEST"); val != "" {
		settings.RealityDest = val
	}
	if val := os.Getenv("SUBSCRIPTION_DOMAIN"); val != "" {
		settings.SubscriptionDomain = val
	}
	if val := os.Getenv("TELEGRAM_BOT_TOKEN"); val != "" {
		settings.TelegramBotToken = val
	}

	// Set defaults if nothing provided (first run)
	if settings.PublicKey == "" {
		settings.PublicKey = "PLEASE_GENERATE_PUBLIC_KEY"
		settings.PrivateKey = "PLEASE_GENERATE_PRIVATE_KEY"
		settings.ShortID = "0123456789abcdef"
	}
	if settings.ServerAddress == "" {
		settings.ServerAddress = "localhost"
	}
	if settings.ClashAPIAddress == "" {
		settings.ClashAPIAddress = "127.0.0.1:9090"
	}
	if settings.RealityDest == "" {
		settings.RealityDest = "www.google.com:443"
	}

	if err := db.SaveSettings(settings); err != nil {
		log.Fatalf("Failed to save settings: %v", err)
	}

	// Start Background Scheduler
	sched := scheduler.NewScheduler(db)
	sched.Start()

	// Start Telegram Bot if token provided
	if settings.TelegramBotToken != "" {
		bot, err := telegram.NewBot(settings.TelegramBotToken, db)
		if err != nil {
			log.Printf("Failed to start Telegram Bot: %v", err)
		} else {
			go bot.Start()
		}
	}

	// Setup Fiber Template Engine
	engine := html.New("./templates", ".html")
	engine.Reload(false) // Optimized for production

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Setup Routes
	routes.SetupRoutes(app, db)

	// Determine Port
	portStr := os.Getenv("PORT")
	if portStr == "" {
		if settings.ManagerPort > 0 {
			portStr = strconv.Itoa(settings.ManagerPort)
		} else {
			portStr = "8080"
		}
	}

	log.Printf("Singbox Manager starting on 127.0.0.1:%s (access via SSH tunnel only)", portStr)
	
	err = app.Listen("127.0.0.1:" + portStr)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
