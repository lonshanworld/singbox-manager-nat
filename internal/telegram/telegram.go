package telegram

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"singbox-manager/internal/models"
	"singbox-manager/internal/store"
)

type Bot struct {
	api   *tgbotapi.BotAPI
	store *store.Store
}

func NewBot(token string, store *store.Store) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		api:   api,
		store: store,
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	log.Printf("Telegram Bot: Started as @%s", b.api.Self.UserName)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if !update.Message.IsCommand() {
			// Check if they pasted a VLESS link directly without a command
			text := strings.TrimSpace(update.Message.Text)
			if strings.HasPrefix(text, "vless://") {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, b.handleBind(update.Message.Chat.ID, text))
				b.api.Send(msg)
			}
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch update.Message.Command() {
		case "start":
			msg.Text = "Welcome to Singbox Manager Bot!\n\nTo link your account:\n1. Copy your VLESS link from the dashboard.\n2. Paste it here (or just send your UUID).\n3. Use /usage to check your data anytime."
		case "bind":
			msg.Text = b.handleBind(update.Message.Chat.ID, update.Message.CommandArguments())
		case "usage":
			msg.Text = b.handleUsage(update.Message)
		default:
			msg.Text = "I don't know that command."
		}

		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Telegram Bot: Error sending message: %v", err)
		}
	}
}

func (b *Bot) handleBind(chatID int64, input string) string {
	if input == "" {
		return "Please provide your UUID or VLESS link: /bind <UUID_OR_LINK>"
	}

	uuid := strings.TrimSpace(input)
	// If it's a VLESS link, extract the UUID (vless://UUID@...)
	if strings.HasPrefix(uuid, "vless://") {
		// Remove prefix and take everything before @
		content := strings.TrimPrefix(uuid, "vless://")
		atIndex := strings.Index(content, "@")
		if atIndex != -1 {
			uuid = content[:atIndex]
		}
	}

	users, err := b.store.GetUsers()
	if err != nil {
		return "System error: could not fetch users."
	}

	var targetUser *models.User
	for i := range users {
		if users[i].UUID == uuid {
			targetUser = &users[i]
			break
		}
	}

	if targetUser == nil {
		return "Error: Could not find an account with that UUID or link. Please check your data."
	}

	// Update user with Telegram ID
	err = b.store.UpdateUser(targetUser.Username, func(u *models.User) {
		u.TelegramID = chatID
	})

	if err != nil {
		return "Error: Failed to bind account."
	}

	return fmt.Sprintf("✅ Success! Your Telegram is now linked to: %s\n\nType /usage anytime to see your data.", targetUser.Username)
}

func (b *Bot) handleUsage(m *tgbotapi.Message) string {
	users, err := b.store.GetUsers()
	if err != nil {
		return "System error."
	}

	var user *models.User
	for i := range users {
		if users[i].TelegramID == m.Chat.ID {
			user = &users[i]
			break
		}
	}

	if user == nil {
		return "You haven't linked an account yet. Paste your VLESS link here or use /bind <UUID> first."
	}

	usedGB := float64(user.UsedBytes) / (1024 * 1024 * 1024)
	limitGB := float64(user.DataLimitGB)
	
	expireStr := "Never"
	if user.ExpireDate != nil {
		expireStr = user.ExpireDate.Format("2006-01-02")
	}

	return fmt.Sprintf("📊 *Usage Report for %s*\n\n"+
		"🔹 Status: %s\n"+
		"🔹 Used: %.3f GB\n"+
		"🔹 Limit: %.0f GB\n"+
		"🔹 Expires: %s\n\n"+
		"Remaining: %.3f GB",
		user.Username,
		strings.ToUpper(user.Status),
		usedGB,
		limitGB,
		expireStr,
		limitGB-usedGB,
	)
}
