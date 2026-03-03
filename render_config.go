package main

import (
	"fmt"
	"os"
	"singbox-manager/internal/config"
	"singbox-manager/internal/models"
)

func main() {
	users := []models.User{
		{
			Username: "user_one",
			UUID:     "90e9aebc-e0cd-48b4-a45d-38cb20f468c7",
			Port:     2371,
			Status:   "active",
		},
		{
			Username: "user_two",
			UUID:     "a1b2c3d4-e5f6-4a5b-b6c7-d8e9f0a1b2c3",
			Port:     2371,
			Status:   "active",
		},
	}

	settings := models.Settings{
		ClashAPIAddress: "0.0.0.0:9090",
		ClashAPISecret:  "mysecret",
		RealityDest:     "www.google.com:443",
		PrivateKey:      "SFDmBHqJCDvxYsYrFmU40e8mV3sbyZ_wnfn3jPNBZWM",
		ShortID:         "0ebed78e68d7bc07",
	}

	configPath := "sample_config.json"
	err := config.GenerateConfig(users, settings, configPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}
