package smtp

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// signDKIM signs the email message with DKIM signature
func (c *Client) signDKIM(message string, msg *Message) (string, error) {
	if c.dkimKey == nil {
		return message, nil
	}

	// Parse headers and body
	parts := strings.Split(message, "\r\n\r\n")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid email format")
	}

	headers := parts[0]
	body := strings.Join(parts[1:], "\r\n\r\n")

	// Create DKIM signature
	dkimHeader, err := c.createDKIMSignature(headers, body)
	if err != nil {
		return "", err
	}

	// Insert DKIM-Signature header at the beginning
	signedMessage := fmt.Sprintf("DKIM-Signature: %s\r\n%s\r\n\r\n%s", dkimHeader, headers, body)
	return signedMessage, nil
}

// createDKIMSignature creates a DKIM signature for the email
func (c *Client) createDKIMSignature(headers, body string) (string, error) {
	// Canonicalize body (simple canonicalization)
	canonicalizedBody := c.canonicalizeBody(body)

	// Hash body
	bodyHash := sha256.Sum256([]byte(canonicalizedBody))
	bodyHashB64 := base64.StdEncoding.EncodeToString(bodyHash[:])

	// Headers to sign (commonly signed headers)
	headersToSign := []string{"from", "to", "subject", "date", "mime-version"}

	// Canonicalize headers
	canonicalizedHeaders, err := c.canonicalizeHeaders(headers, headersToSign)
	if err != nil {
		return "", err
	}

	// Create DKIM-Signature header (without signature value)
	timestamp := time.Now().Unix()
	dkimParams := map[string]string{
		"v":  "1",                              // Version
		"a":  "rsa-sha256",                     // Algorithm
		"c":  "simple/simple",                  // Canonicalization
		"d":  c.domain,                         // Domain
		"s":  c.selector,                       // Selector
		"t":  fmt.Sprintf("%d", timestamp),     // Timestamp
		"h":  strings.Join(headersToSign, ":"), // Headers
		"bh": bodyHashB64,                      // Body hash
		"b":  "",                               // Signature (empty for now)
	}

	// Build DKIM header
	var dkimParts []string
	for _, key := range []string{"v", "a", "c", "d", "s", "t", "h", "bh", "b"} {
		if val, exists := dkimParams[key]; exists {
			dkimParts = append(dkimParts, fmt.Sprintf("%s=%s", key, val))
		}
	}
	dkimHeader := strings.Join(dkimParts, "; ")

	// Canonicalize DKIM header for signing
	dkimForSigning := c.canonicalizeDKIMHeader(dkimHeader)

	// Create signature input
	signatureInput := canonicalizedHeaders + "dkim-signature:" + dkimForSigning

	// Sign with RSA-SHA256
	hash := sha256.Sum256([]byte(signatureInput))
	signature, err := c.dkimKey.Sign(rand.Reader, hash[:], crypto.SHA256)
	if err != nil {
		return "", fmt.Errorf("failed to sign DKIM: %w", err)
	}

	// Encode signature
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	// Replace empty signature with actual signature
	finalDKIMHeader := strings.Replace(dkimHeader, "b=", fmt.Sprintf("b=%s", signatureB64), 1)

	return finalDKIMHeader, nil
}

// canonicalizeBody performs simple canonicalization on the body
func (c *Client) canonicalizeBody(body string) string {
	// Simple canonicalization: just normalize line endings and remove trailing empty lines
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\n", "\r\n")

	// Remove trailing empty lines
	for strings.HasSuffix(normalized, "\r\n\r\n") {
		normalized = strings.TrimSuffix(normalized, "\r\n")
	}

	// Ensure body ends with CRLF
	if !strings.HasSuffix(normalized, "\r\n") {
		normalized += "\r\n"
	}

	return normalized
}

// canonicalizeHeaders canonicalizes headers for DKIM signing
func (c *Client) canonicalizeHeaders(headers string, headersToSign []string) (string, error) {
	headerMap := make(map[string][]string)

	// Parse headers
	headerLines := strings.Split(headers, "\r\n")
	var currentHeader string
	var currentValue string

	for _, line := range headerLines {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			// Continuation of previous header
			currentValue += " " + strings.TrimSpace(line)
		} else {
			// Save previous header if exists
			if currentHeader != "" {
				headerKey := strings.ToLower(currentHeader)
				headerMap[headerKey] = append(headerMap[headerKey], currentValue)
			}

			// Start new header
			if colonIndex := strings.Index(line, ":"); colonIndex != -1 {
				currentHeader = strings.TrimSpace(line[:colonIndex])
				currentValue = strings.TrimSpace(line[colonIndex+1:])
			}
		}
	}

	// Save last header
	if currentHeader != "" {
		headerKey := strings.ToLower(currentHeader)
		headerMap[headerKey] = append(headerMap[headerKey], currentValue)
	}

	// Build canonicalized headers
	var canonicalized strings.Builder
	for _, headerName := range headersToSign {
		headerName = strings.ToLower(headerName)
		if values, exists := headerMap[headerName]; exists {
			// Use the last occurrence of each header
			value := values[len(values)-1]
			// Simple canonicalization: normalize whitespace
			value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
			value = strings.TrimSpace(value)
			canonicalized.WriteString(fmt.Sprintf("%s:%s\r\n", headerName, value))
		}
	}

	return canonicalized.String(), nil
}

// canonicalizeDKIMHeader canonicalizes the DKIM-Signature header for signing
func (c *Client) canonicalizeDKIMHeader(dkimHeader string) string {
	// Simple canonicalization: normalize whitespace and remove signature value
	normalized := regexp.MustCompile(`\s+`).ReplaceAllString(dkimHeader, " ")
	normalized = strings.TrimSpace(normalized)

	// Remove the signature value (b=)
	re := regexp.MustCompile(`b=[^;]*`)
	normalized = re.ReplaceAllString(normalized, "b=")

	return normalized
}
