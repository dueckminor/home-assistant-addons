package smtp

import (
	"fmt"
)

// SendPasswordResetEmail sends a password reset email
func (c *Client) SendPasswordResetEmail(userEmail, resetToken, resetURL string) error {
	senderEmail := c.config.From
	subject := "Password Reset Request"

	// Plain text version
	plainBody := fmt.Sprintf(`Hello,

You have requested to reset your password. Please click the link below to reset your password:

%s

If you did not request this password reset, please ignore this email.

This link will expire in 24 hours.

Best regards,
Gateway Team`, resetURL)

	// HTML version
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2c3e50;">Password Reset Request</h2>
        
        <p>Hello,</p>
        
        <p>You have requested to reset your password. Please click the button below to reset your password:</p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #3498db; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
        </div>
        
        <p>If the button doesn't work, you can copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #3498db;">%s</p>
        
        <p><strong>Important:</strong> If you did not request this password reset, please ignore this email.</p>
        
        <p>This link will expire in 24 hours for security reasons.</p>
        
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        
        <p style="font-size: 12px; color: #666;">
            Best regards,<br>
            Gateway Team
        </p>
    </div>
</body>
</html>`, resetURL, resetURL)

	message := &Message{
		From:     senderEmail,
		To:       []string{userEmail},
		Subject:  subject,
		Body:     plainBody,
		BodyHTML: htmlBody,
		Headers: map[string]string{
			"Reply-To":      senderEmail,
			"X-Mailer":      "Gateway SMTP Client",
			"X-Reset-Token": resetToken, // For tracking purposes
		},
	}

	return c.SendMail(message)
}

// SendWelcomeEmail sends a welcome email to new users
func (c *Client) SendWelcomeEmail(userEmail, username string) error {
	senderEmail := c.config.From

	subject := "Welcome to Gateway"

	plainBody := fmt.Sprintf(`Hello %s,

Welcome to Gateway! Your account has been successfully created.

You can now access your dashboard and configure your gateway services.

If you have any questions, please don't hesitate to contact support.

Best regards,
Gateway Team`, username)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to Gateway</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2c3e50;">Welcome to Gateway!</h2>
        
        <p>Hello %s,</p>
        
        <p>Welcome to Gateway! Your account has been successfully created.</p>
        
        <p>You can now access your dashboard and configure your gateway services.</p>
        
        <div style="background-color: #f8f9fa; padding: 20px; border-radius: 5px; margin: 20px 0;">
            <h3 style="margin-top: 0; color: #2c3e50;">What's Next?</h3>
            <ul>
                <li>Configure your domains</li>
                <li>Set up routing rules</li>
                <li>Manage SSL certificates</li>
                <li>Monitor your services</li>
            </ul>
        </div>
        
        <p>If you have any questions, please don't hesitate to contact support.</p>
        
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        
        <p style="font-size: 12px; color: #666;">
            Best regards,<br>
            Gateway Team
        </p>
    </div>
</body>
</html>`, username)

	message := &Message{
		From:     senderEmail,
		To:       []string{userEmail},
		Subject:  subject,
		Body:     plainBody,
		BodyHTML: htmlBody,
		Headers: map[string]string{
			"Reply-To": senderEmail,
			"X-Mailer": "Gateway SMTP Client",
		},
	}

	return c.SendMail(message)
}
