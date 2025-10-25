package homeassistant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type AddonInfo struct {
	Slug        string                 `json:"slug"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	State       string                 `json:"state"`
	Network     map[string]int         `json:"network"`
	NetworkMode string                 `json:"network_mode"`
	Options     map[string]interface{} `json:"options"`
}

type AddonsResponse struct {
	Result string      `json:"result"`
	Data   []AddonInfo `json:"data"`
}

type AddonResponse struct {
	Result string    `json:"result"`
	Data   AddonInfo `json:"data"`
}

type SupervisorClient struct {
	token   string
	baseURL string
}

func NewSupervisorClient() *SupervisorClient {
	return &SupervisorClient{
		token:   os.Getenv("SUPERVISOR_TOKEN"),
		baseURL: "http://supervisor",
	}
}

// GetAllAddons retrieves all installed add-ons
func (sc *SupervisorClient) GetAllAddons() ([]AddonInfo, error) {
	req, err := http.NewRequest("GET", sc.baseURL+"/addons", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+sc.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var addonsResp AddonsResponse
	if err := json.NewDecoder(resp.Body).Decode(&addonsResp); err != nil {
		return nil, err
	}

	return addonsResp.Data, nil
}

// GetAddonInfo retrieves detailed info for a specific add-on
func (sc *SupervisorClient) GetAddonInfo(slug string) (*AddonInfo, error) {
	req, err := http.NewRequest("GET", sc.baseURL+"/addons/"+slug, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+sc.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var addonResp AddonResponse
	if err := json.NewDecoder(resp.Body).Decode(&addonResp); err != nil {
		return nil, err
	}

	return &addonResp.Data, nil
}

// GetRunningAddons returns only running add-ons with their network details
func (sc *SupervisorClient) GetRunningAddons() ([]AddonTarget, error) {
	addons, err := sc.GetAllAddons()
	if err != nil {
		return nil, err
	}

	var targets []AddonTarget
	for _, addon := range addons {
		if addon.State == "started" {
			// Get detailed info to get network configuration
			info, err := sc.GetAddonInfo(addon.Slug)
			if err != nil {
				continue // Skip if we can't get info
			}

			// Find the main service port
			for portName, port := range info.Network {
				target := AddonTarget{
					Name:        info.Name,
					Slug:        info.Slug,
					Description: info.Description,
					Hostname:    info.Slug, // Add-ons are accessible via their slug as hostname
					Port:        port,
					PortName:    portName,
					URL:         fmt.Sprintf("http://%s:%d", info.Slug, port),
				}
				targets = append(targets, target)
			}
		}
	}

	return targets, nil
}

type AddonTarget struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Hostname    string `json:"hostname"`
	Port        int    `json:"port"`
	PortName    string `json:"port_name"`
	URL         string `json:"url"`
}
