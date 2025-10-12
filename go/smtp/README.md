# SMTP Client Package

A comprehensive SMTP client package for Go with DKIM support, designed for the Gateway project.

## Features

- 📧 **Full SMTP Support**: Send emails with plain text and HTML content
- 🔐 **DKIM Signing**: Automatic DKIM signature generation for email authentication
- 🚀 **SMTP Relay Support**: Use authenticated relays to avoid IP blacklisting (recommended)
- 🎯 **Auto-Discovery**: Automatic SMTP server discovery using MX record lookups
- � **Multiple Providers**: Pre-configured relay clients for Gmail, SendGrid, Mailgun
- 🎨 **Rich HTML Emails**: Support for multipart messages with attachments
- 🔒 **TLS Support**: Secure email transmission
- 🛠 **Easy Configuration**: Simple setup for common use cases

## Quick Start

### Basic Usage

```go
import "github.com/dueckminor/home-assistant-addons/go/smtp"

// Create a local SMTP client (for testing)
client, err := smtp.NewLocalClient("yourdomain.com", "/path/to/data")
if err != nil {
    log.Fatal(err)
}

// Send a simple email
message := &smtp.Message{
    From:    "sender@yourdomain.com",
    To:      []string{"recipient@example.com"},
    Subject: "Test Email",
    Body:    "Hello from Gateway SMTP client!",
}

err = client.SendMail(message)
if err != nil {
    log.Printf("Failed to send email: %v", err)
}
```

### SMTP Relay (Recommended for Production)

For production use, especially from residential or cloud IPs, use an SMTP relay to avoid spam filter blocking:

```go
// Gmail relay (requires app-specific password)
client, err := smtp.NewGmailRelayClient(
    "yourdomain.com",
    "your-email@gmail.com", 
    "your-app-password",
    "/path/to/data",
)

// SendGrid relay (commercial email service)
client, err := smtp.NewSendGridRelayClient(
    "yourdomain.com",
    "SG.your-api-key",
    "/path/to/data", 
)

// Send email via relay - avoids IP blacklisting
message := &smtp.Message{
    From:    "noreply@yourdomain.com",
    To:      []string{"user@anywhere.com"},
    Subject: "Production Email",
    Body:    "Sent via authenticated SMTP relay",
}

err = client.SendMail(message) // Routes through relay server
```

### Auto-Discovery (Direct Delivery)

⚠️ **Warning**: Direct delivery may be blocked by spam filters (Spamhaus, etc.) from residential/dynamic IPs.

```go
// Basic auto-discovery (port 25, no TLS) - may be blocked by spam filters
client, err := smtp.NewAutoDiscoveryClient("yourdomain.com", "/path/to/data")

// Auto-discovery with TLS (port 25 + STARTTLS) - enhanced security
clientTLS, err := smtp.NewAutoDiscoveryTLSClient("yourdomain.com", "/path/to/data")

// Send email - SMTP server is automatically discovered from recipient domain
message := &smtp.Message{
    From:    "sender@yourdomain.com", 
    To:      []string{"user@gmail.com"},     // Discovers MX server for gmail.com
    Subject: "Auto-Discovery Test",
    Body:    "Sent using MX record discovery!",
}

err = client.SendMail(message) // May fail due to IP blacklisting
```

#### Spam Filter Issues with Direct Delivery

- **Spamhaus Blocking**: Residential IPs often blocked by spam databases
- **Dynamic IP Lists**: Cloud/VPS IPs may be on dynamic IP blacklists  
- **Missing Reputation**: New IPs lack sending reputation
- **Solution**: Use authenticated SMTP relay instead of direct delivery

### Gmail Integration

```go
// Create Gmail client (requires app-specific password)
client, err := smtp.NewGmailClient(
    "yourdomain.com",
    "your-email@gmail.com", 
    "your-app-specific-password",
    "/path/to/data",
)

// Send password reset email
err = client.SendPasswordResetEmail(
    "user@example.com",
    "reset-token-123",
    "https://yourdomain.com/reset?token=reset-token-123",
)
```

### Custom SMTP Server

```go
client, err := smtp.NewCustomClient(
    "yourdomain.com",    // Domain
    "mail.yourdomain.com", // SMTP host
    587,                 // Port
    "username",          // SMTP username
    "password",          // SMTP password
    "/path/to/data",     // Data directory for DKIM keys
    true,                // Use TLS
)
```

## DKIM Authentication

The client automatically handles DKIM signing:

1. **Key Generation**: RSA keys are automatically generated and stored per domain
2. **DNS Integration**: Works with the Gateway DNS server for DKIM record publishing
3. **Signature**: All emails are automatically signed with domain DKIM keys

### DKIM Key Storage

DKIM keys are stored as `dkim-<domain>.key` in the specified data directory:
- `/data/dkim-example.com.key`
- `/data/dkim-mail.example.com.key`

## Supported Providers

### Pre-configured Providers

- **Gmail**: `NewGmailClient()` - Requires app-specific password
- **Outlook**: `NewOutlookClient()` - Supports Outlook/Hotmail accounts  
- **SendGrid**: `NewSendGridClient()` - Uses API key authentication
- **Mailgun**: `NewMailgunClient()` - SMTP API access
- **Local**: `NewLocalClient()` - For development with MailHog/similar

### Provider Configuration

```go
// SendGrid example
client, err := smtp.NewSendGridClient(
    "yourdomain.com",
    "SG.your-api-key-here", 
    "/path/to/data",
)

// Mailgun example  
client, err := smtp.NewMailgunClient(
    "yourdomain.com",
    "postmaster@mg.yourdomain.com",
    "your-mailgun-key",
    "/path/to/data", 
)
```

## Email Templates

### Password Reset Email

```go
err = client.SendPasswordResetEmail(
    "user@example.com",           // Recipient
    "secure-reset-token",         // Reset token
    "https://app.com/reset?token=secure-reset-token", // Reset URL
)
```

### Welcome Email

```go
err = client.SendWelcomeEmail(
    "newuser@example.com",        // Recipient
    "John Doe",                   // Username
)
```

### Custom Messages

```go
message := &smtp.Message{
    From:     "noreply@yourdomain.com",
    To:       []string{"user@example.com"},
    Cc:       []string{"admin@yourdomain.com"},
    Subject:  "Custom Email",
    Body:     "Plain text version",
    BodyHTML: "<h1>HTML Version</h1><p>Rich content here</p>",
    Headers: map[string]string{
        "Reply-To":    "support@yourdomain.com",
        "X-Priority":  "1",
        "X-Mailer":    "Gateway SMTP v1.0",
    },
}
```

## Development Setup

### Local Testing with MailHog

```bash
# Run MailHog for local SMTP testing
docker run -p 1025:1025 -p 8025:8025 mailhog/mailhog

# Use local client
client, _ := smtp.NewLocalClient("test.com", "/tmp/smtp-data")
```

Access MailHog web interface at http://localhost:8025

### DNS Requirements

For production use, ensure your DNS has proper mail records:

- **MX Record**: `yourdomain.com. IN MX 10 yourdomain.com.`
- **SPF Record**: `"v=spf1 a:yourdomain.com ~all"`
- **DKIM Record**: `default._domainkey.yourdomain.com. IN TXT "v=DKIM1; k=rsa; p=<public-key>"`
- **DMARC Record**: `_dmarc.yourdomain.com. IN TXT "v=DMARC1; p=none; rua=mailto:postmaster@yourdomain.com"`

## Error Handling

```go
err := client.SendMail(message)
if err != nil {
    // Handle specific error types
    switch {
    case strings.Contains(err.Error(), "authentication failed"):
        log.Printf("SMTP authentication failed - check credentials")
    case strings.Contains(err.Error(), "connection refused"):
        log.Printf("Cannot connect to SMTP server - check host/port")
    case strings.Contains(err.Error(), "DKIM"):
        log.Printf("DKIM signing failed - check private key")
    default:
        log.Printf("Email send failed: %v", err)
    }
}
```

## Security Best Practices

1. **App-Specific Passwords**: Use app-specific passwords for Gmail/Outlook
2. **Environment Variables**: Store SMTP credentials in environment variables
3. **TLS Encryption**: Always use TLS for production SMTP connections
4. **DKIM Keys**: Protect DKIM private keys with proper file permissions
5. **Rate Limiting**: Implement rate limiting for email sending

## Integration with Gateway

The SMTP client integrates seamlessly with the Gateway project:

```go
// In gateway startup
smtpClient, err := smtp.CreateClientForDomain(
    domain.Name,
    gateway.DataDir,
    smtp.Config{
        Host:     os.Getenv("SMTP_HOST"),
        Port:     587,
        Username: os.Getenv("SMTP_USERNAME"),
        Password: os.Getenv("SMTP_PASSWORD"),
        UseTLS:   true,
    },
)

// Use for password resets
err = smtpClient.SendPasswordResetEmail(userEmail, resetToken, resetURL)
```

## License

Part of the home-assistant-addons project.