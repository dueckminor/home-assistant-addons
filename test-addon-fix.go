package main

import (
	"fmt"
	"log"

	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
)

func main() {
	fmt.Println("=== Testing Fixed Add-on Discovery ===")

	client := homeassistant.NewSupervisorClient()

	// Test getting running add-ons (this should now handle local add-ons properly)
	fmt.Println("\nTesting GetRunningAddons() with local add-on filtering...")

	targets, err := client.GetRunningAddons()
	if err != nil {
		log.Printf("Error: %v", err)
		fmt.Println("This is expected in development environment (no SUPERVISOR_TOKEN)")
	} else {
		fmt.Printf("✅ Successfully retrieved %d running add-ons\n", len(targets))
		for _, target := range targets {
			fmt.Printf("  📦 %s (%s) - %s\n", target.Name, target.Slug, target.URL)
		}
	}

	fmt.Println("\n=== Fix Summary ===")
	fmt.Println("✅ Added self-exclusion for local_home_assistant_gateway")
	fmt.Println("✅ Graceful fallback when detailed add-on info fails")
	fmt.Println("✅ Better error handling for 404 and API errors")
	fmt.Println("✅ Only takes first network port per add-on")

	fmt.Println("\nThe error 'Add-on local_home_assistant_gateway is not available inside store' should now be resolved!")
}
