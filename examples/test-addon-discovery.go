package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
)

// TestAddonDiscovery demonstrates the add-on discovery functionality
func main() {
	fmt.Println("=== Home Assistant Add-on Discovery Test ===")

	// Create supervisor client
	client := homeassistant.NewSupervisorClient()

	// Test getting all add-ons
	fmt.Println("\n1. Testing GetAllAddons()...")
	allAddons, err := client.GetAllAddons()
	if err != nil {
		log.Printf("Error getting all add-ons: %v", err)
	} else {
		fmt.Printf("Found %d total add-ons\n", len(allAddons))
		for i, addon := range allAddons {
			if i < 3 { // Show first 3
				fmt.Printf("  - %s (%s)\n", addon.Name, addon.Slug)
			}
		}
		if len(allAddons) > 3 {
			fmt.Printf("  ... and %d more\n", len(allAddons)-3)
		}
	}

	// Test getting running add-ons (this is the main feature)
	fmt.Println("\n2. Testing GetRunningAddons()...")
	runningTargets, err := client.GetRunningAddons()
	if err != nil {
		log.Printf("Error getting running add-ons: %v", err)
	} else {
		fmt.Printf("Found %d running add-ons with network details:\n", len(runningTargets))
		for _, target := range runningTargets {
			fmt.Printf("  📦 %s\n", target.Name)
			fmt.Printf("     URL: %s\n", target.URL)
			fmt.Printf("     Description: %s\n", target.Description)
			fmt.Printf("     Network: %s:%d\n", target.Hostname, target.Port)
			fmt.Println()
		}
	}

	// Test getting specific add-on info
	if len(runningTargets) > 0 {
		testSlug := runningTargets[0].Slug
		fmt.Printf("3. Testing GetAddonInfo() for '%s'...\n", testSlug)

		addonInfo, err := client.GetAddonInfo(testSlug)
		if err != nil {
			log.Printf("Error getting add-on info: %v", err)
		} else {
			// Pretty print the addon info
			infoJSON, _ := json.MarshalIndent(addonInfo, "", "  ")
			fmt.Printf("Add-on details:\n%s\n", string(infoJSON))
		}
	}

	fmt.Println("\n=== Test Configuration ===")

	// Check if we're running in Home Assistant environment
	supervisorToken := os.Getenv("SUPERVISOR_TOKEN")
	if supervisorToken == "" {
		fmt.Println("⚠️  SUPERVISOR_TOKEN not set - running outside Home Assistant")
		fmt.Println("   This is expected in development environment")
		fmt.Println("   In production, this would be automatically provided")
	} else {
		fmt.Println("✅ SUPERVISOR_TOKEN found - running in Home Assistant")
	}

	fmt.Println("\n=== Usage Example for Frontend ===")
	fmt.Println("// JavaScript code to use the new API:")
	fmt.Println("fetch('/api/addons/running')")
	fmt.Println("  .then(response => response.json())")
	fmt.Println("  .then(data => {")
	fmt.Println("    const addons = data.data;")
	fmt.Println("    // Populate dropdown with friendly names")
	fmt.Println("    addons.forEach(addon => {")
	fmt.Println("      addOption(addon.name, addon.url);")
	fmt.Println("    });")
	fmt.Println("  });")
}
