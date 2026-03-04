package models

import "time"

type User struct {
	Username       string    `json:"username"`
	UUID           string    `json:"uuid"`
	Port           int       `json:"port"`
	SpeedLimitMbps int       `json:"speed_limit_mbps"`
	DataLimitGB    int       `json:"data_limit_gb"`
	UsedBytes      int64     `json:"used_bytes"`
	LastSeenBytes  int64     `json:"last_seen_bytes"`
	Status         string    `json:"status"` // "active" or "blocked"
	CreatedAt      time.Time `json:"created_at"`
	ExpireDate     *time.Time`json:"expire_date,omitempty"`
	TelegramID     int64     `json:"telegram_id,omitempty"`
}

type Settings struct {
	AdminSession  string `json:"admin_session,omitempty"` // Secure random session token
	ManagerPort   int    `json:"manager_port"`
	PublicKey     string `json:"public_key"`
	PrivateKey    string `json:"private_key"`
	ShortID       string `json:"short_id"`
	ServerAddress string `json:"server_address"`
	ClashAPIAddress string `json:"clash_api_address"`
	ClashAPISecret  string `json:"clash_api_secret"`
	RealityDest     string `json:"reality_dest"`
	SubscriptionDomain string `json:"subscription_domain"`
	TelegramBotToken   string `json:"telegram_bot_token,omitempty"`
	LastResetMonth     string `json:"last_reset_month,omitempty"`
}
