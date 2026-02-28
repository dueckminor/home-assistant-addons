package homeassistant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type AddonInfo struct {
	Slug        string         `json:"slug"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	State       string         `json:"state"`
	Network     map[string]int `json:"network"`
	NetworkMode string         `json:"network_mode"`
	Options     map[string]any `json:"options"`
}

type AddonsResponse struct {
	Result string `json:"result"`
	Data   struct {
		Addons []AddonInfo `json:"addons"`
	} `json:"data"`
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
		token:   supervisorToken,
		baseURL: supervisorURI,
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

	return addonsResp.Data.Addons, nil
}

// GetAddonInfo retrieves detailed info for a specific add-on
func (sc *SupervisorClient) GetAddonInfo(slug string) (*AddonInfo, error) {
	req, err := http.NewRequest("GET", sc.baseURL+"/addons/"+slug+"/info", nil)
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

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("add-on %s not found (possibly a local add-on)", slug)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var addonResp AddonResponse
	if err := json.NewDecoder(resp.Body).Decode(&addonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if addonResp.Result != "ok" {
		return nil, fmt.Errorf("API returned error result: %s", addonResp.Result)
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
			// Try to get detailed info, but fall back to basic info if it fails
			info := &addon
			if detailedInfo, err := sc.GetAddonInfo(addon.Slug); err == nil {
				info = detailedInfo
			}
			// If GetAddonInfo fails, we'll use the basic info from the initial list

			// Skip our own add-on to avoid self-reference
			if info.Slug == "local_home_assistant_gateway" {
				continue
			}

			// Find the main service port
			if len(info.Network) > 0 {
				for portName, port := range info.Network {
					if port == 0 {
						// Handle the case where port might be 0 (ingress-only add-ons)
						// these addons are meant to be reached via the home-assistant UI
						continue
					}

					// Convert slug to hostname (underscores to hyphens)
					hostname := strings.ReplaceAll(info.Slug, "_", "-")

					target := AddonTarget{
						Name:        info.Name,
						Slug:        info.Slug,
						Description: info.Description,
						Hostname:    hostname,
						Port:        port,
						PortName:    portName,
						URL:         fmt.Sprintf("http://%s:%d", hostname, port),
					}

					targets = append(targets, target)
					break // Only take the first port for now
				}
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
