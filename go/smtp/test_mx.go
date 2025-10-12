package smtp

import (
	"fmt"
	"log"
)

// TestMXDiscovery demonstrates MX record discovery
func TestMXDiscovery() {
	fmt.Println("=== MX Record Discovery Test ===")

	// Create a simple client for testing discovery
	client := &Client{
		config: AutoDiscoveryConfig,
		domain: "test.com",
	}

	// Test domains with known MX records
	testDomains := []string{
		"gmail.com",
		"yahoo.com",
		"outlook.com",
		"microsoft.com",
	}

	for _, domain := range testDomains {
		fmt.Printf("\n--- Testing domain: %s ---\n", domain)

		host, port, err := client.lookupMXRecord(domain)
		if err != nil {
			fmt.Printf("❌ Failed to lookup MX for %s: %v\n", domain, err)
			continue
		}

		fmt.Printf("✅ Found SMTP server for %s:\n", domain)
		fmt.Printf("   Host: %s\n", host)
		fmt.Printf("   Port: %d\n", port)
	}

	fmt.Println("\n=== Testing Full Discovery ===")

	// Test full discovery with multiple recipients
	recipients := []string{
		"user@gmail.com",
		"admin@yahoo.com",
	}

	host, port, err := client.discoverSMTPServer(recipients)
	if err != nil {
		fmt.Printf("❌ Discovery failed: %v\n", err)
	} else {
		fmt.Printf("✅ Discovered SMTP server:\n")
		fmt.Printf("   Host: %s\n", host)
		fmt.Printf("   Port: %d\n", port)
	}
}

// TestAutoDiscoveryFlow demonstrates the complete auto-discovery flow
func TestAutoDiscoveryFlow() {
	fmt.Println("\n=== Auto-Discovery Flow Test ===")

	dataDir := "/tmp/smtp-auto-test"

	// Create auto-discovery client
	client, err := NewAutoDiscoveryClient("example.com", dataDir)
	if err != nil {
		log.Printf("Failed to create auto-discovery client: %v", err)
		return
	}

	fmt.Println("✅ Created auto-discovery SMTP client")

	// Test message to Gmail (should discover smtp.gmail.com)
	message := &Message{
		From:    "sender@example.com",
		To:      []string{"test@gmail.com"},
		Subject: "Auto-Discovery Test",
		Body:    "This is a test of MX record auto-discovery",
		Headers: map[string]string{
			"X-Test": "MX-Discovery",
		},
	}

	fmt.Printf("📧 Testing email delivery to: %v\n", message.To)
	fmt.Printf("   Will discover SMTP server from MX records...\n")

	// This will fail at authentication since we don't have Gmail credentials
	// But it will successfully discover the SMTP server
	err = client.SendMail(message)
	if err != nil {
		fmt.Printf("📨 Email delivery failed (expected): %v\n", err)
		// Check if the error indicates successful discovery but auth failure
		if contains(err.Error(), "authentication failed") ||
			contains(err.Error(), "smtp.gmail.com") {
			fmt.Println("✅ MX discovery worked - found Gmail SMTP server!")
		}
	} else {
		fmt.Println("📫 Email sent successfully!")
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || (len(s) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
