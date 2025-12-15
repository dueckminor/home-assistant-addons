package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
)

func main() {
	fmt.Println("=== InfluxDB Auto-Detection Test ===\n")

	// Check environment
	supervisorToken := os.Getenv("SUPERVISOR_TOKEN")
	if supervisorToken == "" {
		fmt.Println("⚠️  SUPERVISOR_TOKEN not set")
		fmt.Println("   This is expected in development environment")
		fmt.Println("   In Home Assistant, this test would detect InfluxDB automatically\n")
	} else {
		fmt.Println("✅ SUPERVISOR_TOKEN found - running in Home Assistant\n")
	}

	// Create supervisor client
	client := homeassistant.NewSupervisorClient()

	// Attempt to detect InfluxDB
	fmt.Println("Detecting InfluxDB add-on...")
	config, err := client.DetectInfluxDB()
	if err != nil {
		log.Printf("Error: %v\n", err)
		if supervisorToken == "" {
			fmt.Println("\nThis is expected when not running in Home Assistant.")
		}
		return
	}

	// Display results
	fmt.Println("\n=== Detection Results ===")
	if config.Found {
		fmt.Printf("✅ InfluxDB Detected!\n\n")
		fmt.Printf("Name:     %s\n", config.Name)
		fmt.Printf("Slug:     %s\n", config.Slug)
		fmt.Printf("URL:      %s\n", config.URL)
		fmt.Printf("Database: %s\n", config.Database)
		if config.Username != "" {
			fmt.Printf("Username: %s\n", config.Username)
			fmt.Printf("Password: %s\n", maskPassword(config.Password))
		} else {
			fmt.Println("Auth:     No credentials in options (may use tokens)")
		}
	} else {
		fmt.Println("ℹ️  No InfluxDB add-on detected")
		fmt.Println("   The gateway will continue without metrics collection")
	}

	fmt.Println("\n=== How It Works ===")
	fmt.Println("1. Gateway checks for known InfluxDB add-on slugs:")
	fmt.Println("   - a0d7b954_influxdb (Community add-on)")
	fmt.Println("   - influxdb (Official)")
	fmt.Println("   - local_influxdb (Local)")
	fmt.Println("2. Checks add-on names containing 'influx'")
	fmt.Println("3. Retrieves network info and credentials from options")
	fmt.Println("4. Automatically configures metrics collection if found")
}

func maskPassword(password string) string {
	if password == "" {
		return "(empty)"
	}
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}
