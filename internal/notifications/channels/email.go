package channels

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/notifications"
)

// EmailHandler implements notification sending via email
type EmailHandler struct {
	logger *zap.Logger
}

// EmailMessage represents an email message
type EmailMessage struct {
	From        string
	To          []string
	Subject     string
	Body        string
	ContentType string
	Headers     map[string]string
}

// NewEmailHandler creates a new email notification handler
func NewEmailHandler(logger *zap.Logger) *EmailHandler {
	return &EmailHandler{
		logger: logger,
	}
}

// Send sends a notification via email
func (h *EmailHandler) Send(ctx context.Context, channel notifications.NotificationChannel, message notifications.NotificationMessage) error {
	if channel.Config.EmailAddress == "" {
		return fmt.Errorf("email address not configured")
	}

	if channel.Config.SMTPServer == "" {
		return fmt.Errorf("SMTP server not configured")
	}

	emailMsg := h.buildEmailMessage(channel, message)

	// Send email
	err := h.sendEmail(ctx, channel.Config, emailMsg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	h.logger.Info("Successfully sent email notification",
		zap.String("channel_id", channel.ID.String()),
		zap.String("to", channel.Config.EmailAddress),
		zap.String("smtp_server", channel.Config.SMTPServer))

	return nil
}

// Test tests the email channel connectivity
func (h *EmailHandler) Test(ctx context.Context, channel notifications.NotificationChannel) error {
	if channel.Config.EmailAddress == "" {
		return fmt.Errorf("email address not configured")
	}

	if channel.Config.SMTPServer == "" {
		return fmt.Errorf("SMTP server not configured")
	}

	testMessage := notifications.NotificationMessage{
		Subject: "AgentScan Test Notification",
		Body:    "This is a test notification from AgentScan. If you receive this, your email integration is working correctly!",
		Format:  "html",
	}

	return h.Send(ctx, channel, testMessage)
}

// GetChannelType returns the channel type
func (h *EmailHandler) GetChannelType() notifications.NotificationChannelType {
	return notifications.ChannelTypeEmail
}

// buildEmailMessage converts a generic notification message to email format
func (h *EmailHandler) buildEmailMessage(channel notifications.NotificationChannel, message notifications.NotificationMessage) EmailMessage {
	emailMsg := EmailMessage{
		From:    "noreply@agentscan.io",
		To:      []string{channel.Config.EmailAddress},
		Subject: message.Subject,
		Body:    message.Body,
		Headers: make(map[string]string),
	}

	// Set content type based on format
	switch message.Format {
	case "html":
		emailMsg.ContentType = "text/html; charset=UTF-8"
	case "markdown":
		// Convert markdown to HTML for email
		emailMsg.Body = h.markdownToHTML(message.Body)
		emailMsg.ContentType = "text/html; charset=UTF-8"
	default:
		emailMsg.ContentType = "text/plain; charset=UTF-8"
	}

	// Add custom headers
	emailMsg.Headers["X-Mailer"] = "AgentScan Security Scanner"
	emailMsg.Headers["X-Priority"] = "3"

	// Set priority based on message type
	if eventType, exists := message.Metadata["event_type"]; exists {
		switch eventType {
		case "critical_finding":
			emailMsg.Headers["X-Priority"] = "1" // High priority
			emailMsg.Headers["Importance"] = "high"
		case "scan_failed":
			emailMsg.Headers["X-Priority"] = "2" // High priority
			emailMsg.Headers["Importance"] = "high"
		}
	}

	return emailMsg
}

// sendEmail sends an email using SMTP
func (h *EmailHandler) sendEmail(ctx context.Context, config notifications.ChannelConfig, msg EmailMessage) error {
	// Build message
	message := h.buildMIMEMessage(msg)

	// Setup authentication
	var auth smtp.Auth
	if config.SMTPUsername != "" && config.SMTPPassword != "" {
		auth = smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPServer)
	}

	// Determine server address
	serverAddr := fmt.Sprintf("%s:%d", config.SMTPServer, config.SMTPPort)
	if config.SMTPPort == 0 {
		serverAddr = fmt.Sprintf("%s:587", config.SMTPServer) // Default to 587
	}

	// Send email with timeout
	done := make(chan error, 1)
	go func() {
		// For TLS connections (port 465)
		if config.SMTPPort == 465 {
			done <- h.sendEmailTLS(serverAddr, auth, msg.From, msg.To, message)
		} else {
			// For STARTTLS connections (port 587, 25)
			done <- smtp.SendMail(serverAddr, auth, msg.From, msg.To, []byte(message))
		}
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("email send timeout")
	}
}

// sendEmailTLS sends email over TLS connection
func (h *EmailHandler) sendEmailTLS(serverAddr string, auth smtp.Auth, from string, to []string, message string) error {
	// Create TLS connection
	tlsConfig := &tls.Config{
		ServerName: strings.Split(serverAddr, ":")[0],
	}

	conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, tlsConfig.ServerName)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return writer.Close()
}

// buildMIMEMessage builds a MIME-formatted email message
func (h *EmailHandler) buildMIMEMessage(msg EmailMessage) string {
	var message strings.Builder

	// Headers
	message.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	message.WriteString(fmt.Sprintf("Content-Type: %s\r\n", msg.ContentType))
	message.WriteString("MIME-Version: 1.0\r\n")

	// Custom headers
	for key, value := range msg.Headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Empty line to separate headers from body
	message.WriteString("\r\n")

	// Body
	message.WriteString(msg.Body)

	return message.String()
}

// markdownToHTML converts simple markdown to HTML
func (h *EmailHandler) markdownToHTML(markdown string) string {
	html := markdown

	// Convert headers
	html = strings.ReplaceAll(html, "### ", "<h3>")
	html = strings.ReplaceAll(html, "## ", "<h2>")
	html = strings.ReplaceAll(html, "# ", "<h1>")

	// Convert bold
	html = strings.ReplaceAll(html, "**", "<strong>")
	html = strings.ReplaceAll(html, "__", "<strong>")

	// Convert italic
	html = strings.ReplaceAll(html, "*", "<em>")
	html = strings.ReplaceAll(html, "_", "<em>")

	// Convert line breaks
	html = strings.ReplaceAll(html, "\n", "<br>\n")

	// Wrap in basic HTML structure
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>AgentScan Notification</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background-color: #f4f4f4; padding: 20px; border-radius: 5px; }
        .content { padding: 20px; }
        .footer { background-color: #f4f4f4; padding: 10px; text-align: center; font-size: 12px; }
    </style>
</head>
<body>
    <div class="header">
        <h2>AgentScan Security Scanner</h2>
    </div>
    <div class="content">
        %s
    </div>
    <div class="footer">
        <p>This notification was sent by AgentScan Security Scanner</p>
    </div>
</body>
</html>`, html)
}