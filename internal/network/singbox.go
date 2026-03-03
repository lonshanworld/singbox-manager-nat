package network

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ClashProxies struct {
	Proxies map[string]struct {
		Up   int64 `json:"up"`
		Down int64 `json:"down"`
	} `json:"proxies"`
}

// GetUserUsage queries the sing-box Clash API for a specific user's traffic
func GetUserUsage(address string, secret string, username string) (int64, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://%s/proxies", address)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to contact sing-box Clash API at %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return 0, fmt.Errorf("sing-box Clash API returned Unauthorized (401). Check secret.")
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("sing-box Clash API returned status: %d", resp.StatusCode)
	}

	var data ClashProxies
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("failed to decode sing-box clash stats: %v", err)
	}

	// In the Clash API, we look for the unique outbound tag we created for this user
	proxy, ok := data.Proxies["outbound_"+username]
	if !ok {
		return 0, nil
	}

	return proxy.Up + proxy.Down, nil
}

type ClashTraffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// GetTraffic queries the sing-box Clash API for real-time traffic throughput
func GetTraffic(address string, secret string) (int64, int64, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://%s/traffic", address)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, err
	}

	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var data ClashTraffic
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, err
	}

	return data.Up, data.Down, nil
}

// DetectInterface returns the name of the primary network interface or a fallback
func DetectInterface() string {
	// For simplicity, we default to eth0 which is common on Alpine VPS
	// or we could use net package to find it.
	return "eth0"
}
