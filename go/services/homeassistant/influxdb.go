package homeassistant

import (
	"fmt"
	"log"

	"github.com/dueckminor/home-assistant-addons/go/services/influxdb"
)

// InfluxDBConfig represents the detected InfluxDB configuration
type InfluxDBConfig struct {
	Found    bool
	Slug     string
	Name     string
	URL      string
	Username string
	Password string
	Database string
}

// CreateClient creates an InfluxDB client from the configuration
func (c *InfluxDBConfig) CreateClient() (influxdb.Client, error) {
	if !c.Found {
		return nil, fmt.Errorf("InfluxDB not found")
	}
	return influxdb.NewClient(c.URL, c.Database, c.Username, c.Password)
}

// Common InfluxDB add-on slugs in the Home Assistant ecosystem
var influxDBSlugs = []string{
	"a0d7b954_influxdb", // Community add-on
	"influxdb",          // Official add-on (if exists)
	"local_influxdb",    // Local add-on
	"hassio_influxdb",   // Legacy name
}

// DetectInfluxDB attempts to automatically detect a running InfluxDB add-on
func (sc *SupervisorClient) DetectInfluxDB() (*InfluxDBConfig, error) {
	log.Println("Detecting InfluxDB add-on...")

	// Get all running add-ons
	addons, err := sc.GetAllAddons()
	if err != nil {
		return nil, fmt.Errorf("failed to get add-ons: %w", err)
	}

	// Check for known InfluxDB add-ons
	for _, addon := range addons {
		if addon.State != "started" {
			continue
		}

		// Check if this is a known InfluxDB add-on
		isInfluxDB := false
		for _, slug := range influxDBSlugs {
			if addon.Slug == slug {
				isInfluxDB = true
				break
			}
		}

		// Also check if the name contains "influx" (case-insensitive)
		if !isInfluxDB && containsIgnoreCase(addon.Name, "influx") {
			isInfluxDB = true
		}

		if !isInfluxDB {
			continue
		}

		log.Printf("Found potential InfluxDB add-on: %s (%s)", addon.Name, addon.Slug)

		// Get detailed info to access options
		info, err := sc.GetAddonInfo(addon.Slug)
		if err != nil {
			log.Printf("Warning: Could not get detailed info for %s: %v", addon.Slug, err)
			continue
		}

		config := &InfluxDBConfig{
			Found: true,
			Slug:  info.Slug,
			Name:  info.Name,
		}

		// Extract connection details from network info
		if len(info.Network) > 0 {
			hostname := convertSlugToHostname(info.Slug)

			// Try to find the HTTP port (prefer 8086, the standard InfluxDB port)
			if port, ok := info.Network["8086/tcp"]; ok && port > 0 {
				config.URL = fmt.Sprintf("http://%s:%d", hostname, port)
				log.Printf("InfluxDB URL: %s (standard HTTP port)", config.URL)
			} else {
				// Fallback to first available port
				for portName, port := range info.Network {
					if port > 0 {
						config.URL = fmt.Sprintf("http://%s:%d", hostname, port)
						log.Printf("InfluxDB URL: %s (port: %s)", config.URL, portName)
						break
					}
				}
			}
		}

		// Extract credentials from options
		if info.Options != nil {
			// Try common option keys for username
			if username, ok := info.Options["username"].(string); ok {
				config.Username = username
			} else if username, ok := info.Options["user"].(string); ok {
				config.Username = username
			}

			// Try common option keys for password
			if password, ok := info.Options["password"].(string); ok {
				config.Password = password
			} else if password, ok := info.Options["pass"].(string); ok {
				config.Password = password
			}

			// Database name (often "homeassistant" or "ha")
			if db, ok := info.Options["database"].(string); ok {
				config.Database = db
			} else if db, ok := info.Options["db"].(string); ok {
				config.Database = db
			}

			// Default database if not specified
			if config.Database == "" {
				config.Database = "homeassistant"
			}
		}

		// Log what we found (without exposing password)
		if config.Username != "" {
			log.Printf("InfluxDB credentials found - Username: %s, Database: %s", config.Username, config.Database)
		} else {
			log.Println("InfluxDB found but no credentials in options (may use authentication tokens)")
		}

		return config, nil
	}

	log.Println("No InfluxDB add-on detected")
	return &InfluxDBConfig{Found: false}, nil
}

// Helper function to convert slug to hostname (replace underscores with hyphens)
func convertSlugToHostname(slug string) string {
	result := ""
	for _, char := range slug {
		if char == '_' {
			result += "-"
		} else {
			result += string(char)
		}
	}
	return result
}

// Helper function for case-insensitive substring search
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := ""
	for _, char := range s {
		if char >= 'A' && char <= 'Z' {
			result += string(char + 32)
		} else {
			result += string(char)
		}
	}
	return result
}

func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
