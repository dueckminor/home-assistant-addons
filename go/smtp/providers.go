package smtp

import "time"

// Common SMTP provider configurations
var (
	// Gmail SMTP configuration (requires app-specific password)
	GmailConfig = Config{
		Host:    "smtp.gmail.com",
		Port:    587,
		UseTLS:  true,
		Timeout: 30 * time.Second,
	}

	// Outlook/Hotmail SMTP configuration
	OutlookConfig = Config{
		Host:    "smtp-mail.outlook.com",
		Port:    587,
		UseTLS:  true,
		Timeout: 30 * time.Second,
	}

	// SendGrid SMTP configuration
	SendGridConfig = Config{
		Host:    "smtp.sendgrid.net",
		Port:    587,
		UseTLS:  true,
		Timeout: 30 * time.Second,
	}

	// Mailgun SMTP configuration
	MailgunConfig = Config{
		Host:    "smtp.mailgun.org",
		Port:    587,
		UseTLS:  true,
		Timeout: 30 * time.Second,
	}

	// Local SMTP server configuration (for development/testing)
	LocalConfig = Config{
		Host:    "localhost",
		Port:    1025, // Common port for local SMTP servers like MailHog
		UseTLS:  false,
		Timeout: 10 * time.Second,
	}

	// Self-hosted SMTP configuration template
	SelfHostedConfig = Config{
		Host:    "", // Set your SMTP server
		Port:    587,
		UseTLS:  true,
		Timeout: 30 * time.Second,
	}

	// Auto-discovery configuration - discovers SMTP server from recipient domains
	AutoDiscoveryConfig = Config{
		Host:    "",    // Empty - will be auto-discovered from recipient MX records
		Port:    0,     // 0 - will use port 25 for MX delivery
		UseTLS:  false, // Disabled by default for MX delivery (port 25)
		Timeout: 30 * time.Second,
	}

	// Auto-discovery with TLS configuration - for servers supporting STARTTLS on port 25
	AutoDiscoveryTLSConfig = Config{
		Host:    "",   // Empty - will be auto-discovered from recipient MX records
		Port:    0,    // 0 - will use port 25 for MX delivery
		UseTLS:  true, // Enabled - will attempt STARTTLS on port 25
		Timeout: 30 * time.Second,
	}

	// Gmail Relay Configuration - Use Gmail as SMTP relay (requires app password)
	GmailRelayConfig = Config{
		UseRelay:    true,
		RelayHost:   "smtp.gmail.com",
		RelayPort:   587,
		RelayUseTLS: true,
		Timeout:     30 * time.Second,
	}

	// SendGrid Relay Configuration - Use SendGrid as SMTP relay
	SendGridRelayConfig = Config{
		UseRelay:    true,
		RelayHost:   "smtp.sendgrid.net",
		RelayPort:   587,
		RelayUseTLS: true,
		Timeout:     30 * time.Second,
	}

	// Mailgun Relay Configuration - Use Mailgun as SMTP relay
	MailgunRelayConfig = Config{
		UseRelay:    true,
		RelayHost:   "smtp.mailgun.org",
		RelayPort:   587,
		RelayUseTLS: true,
		Timeout:     30 * time.Second,
	}
)

// NewGmailClient creates an SMTP client configured for Gmail
func NewGmailClient(domain, username, password, dataDir string) (*Client, error) {
	config := GmailConfig
	config.Username = username
	config.Password = password

	return CreateClientForDomain(domain, dataDir, config)
}

// NewOutlookClient creates an SMTP client configured for Outlook/Hotmail
func NewOutlookClient(domain, username, password, dataDir string) (*Client, error) {
	config := OutlookConfig
	config.Username = username
	config.Password = password

	return CreateClientForDomain(domain, dataDir, config)
}

// NewSendGridClient creates an SMTP client configured for SendGrid
func NewSendGridClient(domain, apiKey, dataDir string) (*Client, error) {
	config := SendGridConfig
	config.Username = "apikey" // SendGrid uses "apikey" as username
	config.Password = apiKey

	return CreateClientForDomain(domain, dataDir, config)
}

// NewMailgunClient creates an SMTP client configured for Mailgun
func NewMailgunClient(domain, username, password, dataDir string) (*Client, error) {
	config := MailgunConfig
	config.Username = username
	config.Password = password

	return CreateClientForDomain(domain, dataDir, config)
}

// NewLocalClient creates an SMTP client for local testing (no authentication)
func NewLocalClient(domain, dataDir string) (*Client, error) {
	config := LocalConfig
	// No username/password for local testing

	return CreateClientForDomain(domain, dataDir, config)
}

// NewAutoDiscoveryClient creates an SMTP client that discovers servers from recipient MX records
// Uses port 25 without TLS for maximum compatibility
func NewAutoDiscoveryClient(domain, dataDir string) (*Client, error) {
	config := AutoDiscoveryConfig
	// No username/password - relies on recipient's SMTP server accepting mail

	return CreateClientForDomain(domain, dataDir, config)
}

// NewAutoDiscoveryTLSClient creates an SMTP client with MX discovery and TLS support
// Uses port 25 with STARTTLS for enhanced security (may not work with all servers)
func NewAutoDiscoveryTLSClient(domain, dataDir string) (*Client, error) {
	config := AutoDiscoveryTLSConfig
	// No username/password - relies on recipient's SMTP server accepting mail

	return CreateClientForDomain(domain, dataDir, config)
}

// NewGmailRelayClient creates an SMTP client using Gmail as relay (avoids IP blacklisting)
func NewGmailRelayClient(domain, username, password, dataDir string) (*Client, error) {
	config := GmailRelayConfig
	config.Username = username
	config.Password = password

	return CreateClientForDomain(domain, dataDir, config)
}

// NewSendGridRelayClient creates an SMTP client using SendGrid as relay
func NewSendGridRelayClient(domain, apiKey, dataDir string) (*Client, error) {
	config := SendGridRelayConfig
	config.Username = "apikey" // SendGrid uses "apikey" as username
	config.Password = apiKey

	return CreateClientForDomain(domain, dataDir, config)
}

// NewMailgunRelayClient creates an SMTP client using Mailgun as relay
func NewMailgunRelayClient(domain, username, password, dataDir string) (*Client, error) {
	config := MailgunRelayConfig
	config.Username = username
	config.Password = password

	return CreateClientForDomain(domain, dataDir, config)
}

// NewCustomClient creates an SMTP client with custom configuration
func NewCustomClient(domain, host string, port int, username, password, dataDir string, useTLS bool) (*Client, error) {
	config := Config{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		UseTLS:   useTLS,
		Timeout:  30 * time.Second,
	}

	return CreateClientForDomain(domain, dataDir, config)
}

// NewCustomRelayClient creates an SMTP client with custom relay configuration
func NewCustomRelayClient(domain, relayHost string, relayPort int, username, password, dataDir string, useTLS bool) (*Client, error) {
	config := Config{
		UseRelay:    true,
		RelayHost:   relayHost,
		RelayPort:   relayPort,
		RelayUseTLS: useTLS,
		Username:    username,
		Password:    password,
		Timeout:     30 * time.Second,
	}

	return CreateClientForDomain(domain, dataDir, config)
}
