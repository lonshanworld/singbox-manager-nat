package config

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"singbox-manager/internal/models"
)

type SingboxConfig struct {
	Log          LogConfig       `json:"log"`
	Experimental *Experimental   `json:"experimental,omitempty"`
	Inbounds     []InboundConfig `json:"inbounds"`
	Outbounds    []OutboundConfig`json:"outbounds"`
	Route        *RouteConfig    `json:"route,omitempty"`
	Stats        *StatsConfig    `json:"stats,omitempty"`
}

type LogConfig struct {
	Disabled  bool   `json:"disabled"`
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
}

type Experimental struct {
	CacheFile *CacheFileConfig `json:"cache_file,omitempty"`
	ClashAPI  *ClashAPIConfig `json:"clash_api,omitempty"`
}

type CacheFileConfig struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}

type ClashAPIConfig struct {
	ExternalController string `json:"external_controller"`
	Secret             string `json:"secret,omitempty"`
}

type InboundConfig struct {
	Type         string      `json:"type"`
	Tag          string      `json:"tag"`
	Listen       string      `json:"listen"`
	ListenPort   int         `json:"listen_port"`
	Users        []VLESSUser `json:"users"`
	TLS          *TLSConfig  `json:"tls,omitempty"`
	Sniff        bool        `json:"sniff,omitempty"`
	SniffTimeout string      `json:"sniff_timeout,omitempty"`
}

type VLESSUser struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type TLSConfig struct {
	Enabled    bool          `json:"enabled"`
	ServerName string        `json:"server_name"`
	Reality    RealityConfig`json:"reality"`
}

type RealityConfig struct {
	Enabled   bool            `json:"enabled"`
	Handshake HandshakeConfig`json:"handshake"`
	PrivateKey string         `json:"private_key"`
	ShortID   []string        `json:"short_id"`
}

type HandshakeConfig struct {
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
}

type OutboundConfig struct {
	Type string `json:"type"`
	Tag  string `json:"tag"`
}

type RouteConfig struct {
	Rules []RouteRule `json:"rules"`
	Final string      `json:"final"`
}

type RouteRule struct {
	AuthUser []string `json:"auth_user,omitempty"`
	Action   string   `json:"action"`
	Outbound string   `json:"outbound"`
}

type StatsConfig struct{}

// GenerateConfig generates a sing-box config.json based on active users and settings
func GenerateConfig(users []models.User, customSettings models.Settings, configPath string) error {

	conf := SingboxConfig{
		Log: LogConfig{
			Disabled:  false,
			Level:     "warn",
			Timestamp: true,
		},
		Experimental: &Experimental{
			CacheFile: &CacheFileConfig{
				Enabled: true,
				Path:    "/etc/sing-box/cache.db",
			},
			ClashAPI: &ClashAPIConfig{
				ExternalController: customSettings.ClashAPIAddress,
				Secret:             customSettings.ClashAPISecret,
			},
		},
		Outbounds: []OutboundConfig{
			{Type: "direct", Tag: "direct"},
			{Type: "block", Tag: "block"},
		},
		Stats: &StatsConfig{},
	}

	// Single inbound for all users
	inboundPort := 2371 // Default port, could be made configurable via settings if needed
	
	// Use the port of the first user if available, or 2371
	// Filter active users
	var vlessUsers []VLESSUser
	for _, u := range users {
		if u.Status == "active" {
			vlessUsers = append(vlessUsers, VLESSUser{
				UUID: u.UUID,
				Name: u.Username,
			})
		}
	}

	if len(vlessUsers) > 0 {
		// Extract SNI and Port from RealityDest (host:port)
		sni := customSettings.RealityDest
		handshakePort := 443
		if host, portStr, err := net.SplitHostPort(customSettings.RealityDest); err == nil {
			sni = host
			if p, err := strconv.Atoi(portStr); err == nil {
				handshakePort = p
			}
		}

		conf.Inbounds = []InboundConfig{
			{
				Type:       "vless",
				Tag:        "vless-inbound",
				Listen:     "::",
				ListenPort: inboundPort,
				Users:      vlessUsers,
				TLS: &TLSConfig{
					Enabled:    true,
					ServerName: sni,
					Reality: RealityConfig{
						Enabled: true,
						Handshake: HandshakeConfig{
							Server:     sni,
							ServerPort: handshakePort,
						},
						PrivateKey: customSettings.PrivateKey,
						ShortID:    []string{customSettings.ShortID},
					},
				},
				Sniff:        true,
				SniffTimeout: "300ms",
			},
		}

		// Generate unique outbounds for each user for tracking
		var extraOutbounds []OutboundConfig
		var rules []RouteRule
		for _, u := range vlessUsers {
			tag := "outbound_" + u.Name
			extraOutbounds = append(extraOutbounds, OutboundConfig{
				Type: "direct",
				Tag:  tag,
			})
			rules = append(rules, RouteRule{
				AuthUser: []string{u.Name},
				Action:   "route",
				Outbound: tag,
			})
		}

		conf.Outbounds = append(conf.Outbounds, extraOutbounds...)
		conf.Route = &RouteConfig{
			Rules: rules,
			Final: "direct",
		}
	}

	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
