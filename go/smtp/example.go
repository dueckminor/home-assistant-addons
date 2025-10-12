package smtp

import (
	"fmt"
	"log"
	"os"
)

// Example demonstrates how to use the SMTP client
func Example() {
	// Example 1: Create a local SMTP client for testing
	domain := "example.com"
	dataDir := "/tmp/smtp-test"

	// Ensure data directory exists
	os.MkdirAll(dataDir, 0755)

	// Create local SMTP client (for testing with MailHog or similar)
	client, err := NewLocalClient(domain, dataDir)
	if err != nil {
		log.Fatalf("Failed to create SMTP client: %v", err)
	}

	fmt.Printf("Created SMTP client for domain: %s\n", domain)

	// Example 2: Send a simple email
	message := &Message{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Email",
		Body:    "This is a test email from the Gateway SMTP client.",
		Headers: map[string]string{
			"X-Mailer": "Gateway SMTP Client",
		},
	}

	// Note: This will fail if no SMTP server is running on localhost:1025
	// You can use MailHog for local testing: docker run -p 1025:1025 -p 8025:8025 mailhog/mailhog
	err = client.SendMail(message)
	if err != nil {
		fmt.Printf("Failed to send email (expected if no SMTP server running): %v\n", err)
	} else {
		fmt.Println("Email sent successfully!")
	}

	// Example 3: Send password reset email
	err = client.SendPasswordResetEmail(
		"user@example.com",
		"reset-token-123",
		"https://gateway.example.com/reset-password?token=reset-token-123",
	)
	if err != nil {
		fmt.Printf("Failed to send password reset email: %v\n", err)
	} else {
		fmt.Println("Password reset email sent!")
	}

	// Example 4: Gmail Relay client (avoids IP blacklisting issues)
	// Recommended for production use from residential/cloud IPs
	/*
		gmailRelayClient, err := NewGmailRelayClient(
			domain,
			"your-email@gmail.com",
			"your-app-specific-password",
			dataDir,
		)
		if err != nil {
			fmt.Printf("Failed to create Gmail relay client: %v\n", err)
		} else {
			fmt.Println("Created Gmail relay SMTP client!")

			relayMessage := &Message{
				From:    "sender@yourdomain.com",
				To:      []string{"recipient@anywhere.com"},
				Subject: "Relay Test",
				Body:    "Sent via Gmail relay - avoids IP blacklisting!",
			}

			err = gmailRelayClient.SendMail(relayMessage)
			if err != nil {
				fmt.Printf("Relay email failed: %v\n", err)
			} else {
				fmt.Println("Relay email sent successfully!")
			}
		}
	*/

	// Example 5: Auto-discovery SMTP client (discovers server from recipient MX records)
	// WARNING: May be blocked by spam filters from residential IPs
	autoClient, err := NewAutoDiscoveryClient(domain, dataDir)
	if err != nil {
		fmt.Printf("Failed to create auto-discovery client: %v\n", err)
	} else {
		fmt.Println("Created auto-discovery SMTP client (port 25, no TLS)!")

		// This will automatically discover the SMTP server and use port 25
		autoMessage := &Message{
			From:    "sender@example.com",
			To:      []string{"recipient@gmail.com"}, // Will discover MX server for gmail.com:25
			Subject: "Auto-Discovery Test",
			Body:    "This email was sent using MX record auto-discovery on port 25!",
		}

		err = autoClient.SendMail(autoMessage)
		if err != nil {
			fmt.Printf("Auto-discovery email failed: %v\n", err)
		} else {
			fmt.Println("Auto-discovery email sent!")
		}
	}

	// Example 5: Auto-discovery with TLS (port 25 + STARTTLS)
	autoTLSClient, err := NewAutoDiscoveryTLSClient(domain, dataDir)
	if err != nil {
		fmt.Printf("Failed to create auto-discovery TLS client: %v\n", err)
	} else {
		fmt.Println("Created auto-discovery SMTP client with TLS (port 25 + STARTTLS)!")

		autoTLSMessage := &Message{
			From:    "sender@example.com",
			To:      []string{"recipient@outlook.com"}, // Will discover MX server for outlook.com:25
			Subject: "Auto-Discovery TLS Test",
			Body:    "This email was sent using MX auto-discovery with STARTTLS on port 25!",
		}

		err = autoTLSClient.SendMail(autoTLSMessage)
		if err != nil {
			fmt.Printf("Auto-discovery TLS email failed: %v\n", err)
		} else {
			fmt.Println("Auto-discovery TLS email sent!")
		}
	}

	// Example 6: Create Gmail client (requires app-specific password)
	/*
		gmailClient, err := NewGmailClient(
			"yourdomain.com",
			"your-email@gmail.com",
			"your-app-specific-password",
			dataDir,
		)
		if err != nil {
			log.Fatalf("Failed to create Gmail client: %v", err)
		}

		err = gmailClient.SendWelcomeEmail("newuser@example.com", "John Doe")
		if err != nil {
			log.Printf("Failed to send welcome email: %v", err)
		}
	*/

	fmt.Println("SMTP client example completed!")
}

// ExampleCustomSMTP shows how to create a custom SMTP client
func ExampleCustomSMTP() {
	domain := "yourdomain.com"
	dataDir := "/path/to/data"

	// Create custom SMTP client
	client, err := NewCustomClient(
		domain,
		"mail.yourdomain.com", // Your SMTP server
		587,                   // SMTP port
		"username",            // SMTP username
		"password",            // SMTP password
		dataDir,
		true, // Use TLS
	)
	if err != nil {
		log.Fatalf("Failed to create custom SMTP client: %v", err)
	}

	// Send email with custom client
	message := &Message{
		From:     fmt.Sprintf("noreply@%s", domain),
		To:       []string{"recipient@example.com"},
		Subject:  "Welcome to Our Service",
		Body:     "Thank you for joining our service!",
		BodyHTML: "<h1>Welcome!</h1><p>Thank you for joining our service!</p>",
	}

	err = client.SendMail(message)
	if err != nil {
		log.Printf("Failed to send email: %v", err)
		return
	}

	fmt.Println("Custom SMTP email sent successfully!")
}
