package smtp

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Config holds SMTP client configuration
type Config struct {
	Host     string // SMTP server host (optional - will be discovered from recipient if empty)
	Port     int    // SMTP server port (optional - will use standard ports if 0)
	Username string // SMTP username (optional)
	Password string // SMTP password (optional)
	UseTLS   bool   // Whether to use TLS
	Timeout  time.Duration
	From     string // Default for the sender address
}

// Message represents an email message
type Message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	BodyHTML    string
	Headers     map[string]string
	Attachments []Attachment
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Client represents an SMTP client with DKIM support
type Client struct {
	config Config
}

// NewClient creates a new SMTP client
func NewClient(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Client{
		config: config,
	}
}

// SendMail sends an email message
func (c *Client) SendMail(msg *Message) error {
	// Build the email message
	emailContent, err := c.buildMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	// Connect to SMTP server
	addr := net.JoinHostPort(c.config.Host, fmt.Sprintf("%d", c.config.Port))

	var conn net.Conn
	useTLS := c.config.UseTLS

	if useTLS {
		tlsConfig := &tls.Config{ServerName: c.config.Host}
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: c.config.Timeout}, "tcp", addr, tlsConfig)
	} else {
		conn, err = net.DialTimeout("tcp", addr, c.config.Timeout)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server %s: %w", addr, err)
	}
	defer conn.Close()

	// Create SMTP client
	smtpClient, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer smtpClient.Quit()

	// Authenticate if credentials are provided
	if c.config.Username != "" && c.config.Password != "" {
		auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)
		if err = smtpClient.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed for %s: %w", c.config.Host, err)
		}
	}

	from := msg.From
	if from == "" {
		from = c.config.From
	}

	// Set sender
	if err = smtpClient.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	allRecipients := append(msg.To, msg.Cc...)
	allRecipients = append(allRecipients, msg.Bcc...)
	for _, recipient := range allRecipients {
		if err = smtpClient.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := smtpClient.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}
	defer writer.Close()

	if _, err = writer.Write([]byte(emailContent)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// buildMessage constructs the email message with headers and body
func (c *Client) buildMessage(msg *Message) (string, error) {
	var builder strings.Builder

	// Add standard headers
	builder.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))

	if len(msg.To) > 0 {
		builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	}

	if len(msg.Cc) > 0 {
		builder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.Cc, ", ")))
	}

	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	builder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	builder.WriteString("MIME-Version: 1.0\r\n")

	// Add custom headers
	for key, value := range msg.Headers {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Determine content type
	if msg.BodyHTML != "" && msg.Body != "" {
		// Multipart message
		boundary := fmt.Sprintf("boundary_%d", time.Now().Unix())
		builder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		builder.WriteString("\r\n")

		// Plain text part
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(msg.Body)
		builder.WriteString("\r\n")

		// HTML part
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(msg.BodyHTML)
		builder.WriteString("\r\n")

		builder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if msg.BodyHTML != "" {
		// HTML only
		builder.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(msg.BodyHTML)
	} else {
		// Plain text only
		builder.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(msg.Body)
	}

	return builder.String(), nil
}
