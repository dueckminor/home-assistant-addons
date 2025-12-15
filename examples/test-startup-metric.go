package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/homeassistant"
)

func main() {
	fmt.Println("=== Startup Metric Test ===\n")

	// Check environment
	supervisorToken := os.Getenv("SUPERVISOR_TOKEN")
	if supervisorToken == "" {
		fmt.Println("⚠️  SUPERVISOR_TOKEN not set")
		fmt.Println("   This test simulates what happens in Home Assistant\n")
	}

	// Simulate detection
	fmt.Println("Simulating InfluxDB detection and startup metric...")

	// In real scenario, this would be done by the gateway
	config := &homeassistant.InfluxDBConfig{
		Found:    true,
		Name:     "InfluxDB (simulated)",
		Slug:     "a0d7b954_influxdb",
		URL:      "http://localhost:8086", // Change to your InfluxDB URL for testing
		Username: "homeassistant",
		Password: "password", // Change to your password for testing
		Database: "homeassistant",
	}

	// Try to send startup metric
	fmt.Println("\nAttempting to send startup metric...")
	client, err := config.CreateClient()
	if err != nil {
		log.Printf("❌ Failed to create InfluxDB client: %v\n", err)
		fmt.Println("\nThis is expected if InfluxDB is not running at localhost:8086")
		fmt.Println("When running in Home Assistant, the URL would be auto-detected")
		return
	}
	defer client.Close()

	tags := map[string]string{
		"service": "gateway",
		"event":   "startup",
	}

	err = client.SendMetric("gateway_events", 1, tags)
	if err != nil {
		log.Printf("❌ Failed to send metric: %v\n", err)
		fmt.Println("\nThis is expected if InfluxDB is not running or has different credentials")
		return
	}

	fmt.Println("✅ Startup metric sent successfully!")

	fmt.Println("\n=== Metric Details ===")
	fmt.Println("Measurement:  gateway_events")
	fmt.Println("Value:        1")
	fmt.Println("Tags:")
	fmt.Println("  - service: gateway")
	fmt.Println("  - event: startup")
	fmt.Printf("Timestamp:    %s\n", time.Now().Format(time.RFC3339))

	fmt.Println("\n=== InfluxDB Query ===")
	fmt.Println("To view this data in InfluxDB:")
	fmt.Println("  SELECT * FROM gateway_events WHERE event='startup'")
	fmt.Println("\nTo count restarts over time:")
	fmt.Println("  SELECT COUNT(value) FROM gateway_events")
	fmt.Println("    WHERE event='startup'")
	fmt.Println("    GROUP BY time(1h)")
}
