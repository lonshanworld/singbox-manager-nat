package config

import (
	"encoding/json"
	"os"
	"singbox-manager/internal/models"
	"testing"
)

func TestGenerateConfig(t *testing.T) {
	users := []models.User{
		{Username: "user1", UUID: "uuid1", Port: 2371, Status: "active"},
		{Username: "user2", UUID: "uuid2", Port: 2371, Status: "active"},
	}
	settings := models.Settings{
		ClashAPIAddress: "127.0.0.1:9090",
		ClashAPISecret:  "mypassword",
		RealityDest:     "www.google.com:443",
		PrivateKey:      "privkey",
		ShortID:         "shortid",
	}

	tmpFile := "test_config.json"
	defer os.Remove(tmpFile)

	err := GenerateConfig(users, settings, tmpFile)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read generated config: %v", err)
	}

	var conf SingboxConfig
	if err := json.Unmarshal(data, &conf); err != nil {
		t.Fatalf("Failed to unmarshal generated config: %v", err)
	}

	// Verify Clash API
	if conf.Experimental == nil || conf.Experimental.ClashAPI == nil {
		t.Fatal("Experimental.ClashAPI is missing")
	}
	if conf.Experimental.ClashAPI.ExternalController != "127.0.0.1:9090" {
		t.Errorf("Expected ClashAPIAddress 127.0.0.1:9090, got %s", conf.Experimental.ClashAPI.ExternalController)
	}
	if conf.Experimental.ClashAPI.Secret != "mypassword" {
		t.Errorf("Expected ClashAPISecret mypassword, got %s", conf.Experimental.ClashAPI.Secret)
	}

	// Verify Cache File
	if conf.Experimental.CacheFile == nil {
		t.Fatal("Experimental.CacheFile is missing")
	}
	if conf.Experimental.CacheFile.Path != "/etc/sing-box/cache.db" {
		t.Errorf("Expected cache path /etc/sing-box/cache.db, got %s", conf.Experimental.CacheFile.Path)
	}

	// Verify Reality
	if len(conf.Inbounds) == 0 || conf.Inbounds[0].TLS == nil {
		t.Fatal("Inbound TLS is missing")
	}
	if conf.Inbounds[0].TLS.ServerName != "www.google.com" {
		t.Errorf("Expected ServerName www.google.com, got %s", conf.Inbounds[0].TLS.ServerName)
	}

	// Verify Sniffing
	if !conf.Inbounds[0].Sniff {
		t.Error("Expected Sniff true, got false")
	}
	if conf.Inbounds[0].SniffTimeout != "300ms" {
		t.Errorf("Expected SniffTimeout 300ms, got %s", conf.Inbounds[0].SniffTimeout)
	}
}
