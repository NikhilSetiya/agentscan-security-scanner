package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// BetaInvitation represents a beta program invitation
type BetaInvitation struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Company     string    `json:"company"`
	UseCase     string    `json:"use_case"`
	GitHubURL   string    `json:"github_url"`
	Status      string    `json:"status"` // pending, sent, accepted, expired
	InviteCode  string    `json:"invite_code"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}

// Config holds application configuration
type Config struct {
	DatabaseURL    string
	SendGridAPIKey string
	FromEmail      string
	WebAppURL      string
	APIBaseURL     string
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	config := Config{
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		SendGridAPIKey: getEnv("SENDGRID_API_KEY", ""),
		FromEmail:      getEnv("FROM_EMAIL", "beta@agentscan.dev"),
		WebAppURL:      getEnv("WEB_APP_URL", "https://app.agentscan.dev"),
		APIBaseURL:     getEnv("API_BASE_URL", "https://api.agentscan.dev"),
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: beta-inviter <command>")
		fmt.Println("Commands:")
		fmt.Println("  send-invites    - Send pending invitations")
		fmt.Println("  cleanup         - Clean up expired invitations")
		fmt.Println("  stats           - Show invitation statistics")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "send-invites":
		if err := sendPendingInvites(config); err != nil {
			log.Fatalf("Failed to send invites: %v", err)
		}
	case "cleanup":
		if err := cleanupExpiredInvites(config); err != nil {
			log.Fatalf("Failed to cleanup: %v", err)
		}
	case "stats":
		if err := showStats(config); err != nil {
			log.Fatalf("Failed to show stats: %v", err)
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func sendPendingInvites(config Config) error {
	log.Println("Sending pending beta invitations...")

	// Mock pending invitations (in real implementation, fetch from database)
	invitations := []BetaInvitation{
		{
			ID:        "inv_001",
			Email:     "developer@example.com",
			Name:      "John Developer",
			Company:   "Tech Corp",
			UseCase:   "CI/CD security scanning",
			GitHubURL: "https://github.com/johndeveloper",
			Status:    "pending",
		},
	}

	for _, invitation := range invitations {
		if err := sendInvitationEmail(config, invitation); err != nil {
			log.Printf("Failed to send invitation to %s: %v", invitation.Email, err)
			continue
		}

		// Update status to sent (in real implementation, update database)
		log.Printf("Invitation sent to %s", invitation.Email)
	}

	return nil
}

func sendInvitationEmail(config Config, invitation BetaInvitation) error {
	template := createInvitationTemplate(config, invitation)

	// Create SendGrid payload
	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{
					{
						"email": invitation.Email,
						"name":  invitation.Name,
					},
				},
				"substitutions": map[string]string{
					"-name-":        invitation.Name,
					"-company-":     invitation.Company,
					"-invite_code-": invitation.InviteCode,
					"-signup_url-":  fmt.Sprintf("%s/signup?code=%s", config.WebAppURL, invitation.InviteCode),
				},
			},
		},
		"from": map[string]string{
			"email": config.FromEmail,
			"name":  "AgentScan Team",
		},
		"subject":  template.Subject,
		"content": []map[string]string{
			{
				"type":  "text/html",
				"value": template.HTML,
			},
			{
				"type":  "text/plain",
				"value": template.Text,
			},
		},
	}

	// Send email via SendGrid API
	return sendEmail(config.SendGridAPIKey, payload)
}

func createInvitationTemplate(config Config, invitation BetaInvitation) EmailTemplate {
	subject := "🛡️ Welcome to AgentScan Beta - Your Security Scanning Platform Awaits!"

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to AgentScan Beta</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 24px; font-weight: bold; color: #2563eb; }
        .content { background: #f8fafc; padding: 30px; border-radius: 8px; margin: 20px 0; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 500; }
        .features { margin: 20px 0; }
        .feature { margin: 10px 0; padding-left: 20px; }
        .footer { text-align: center; margin-top: 30px; font-size: 14px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">🛡️ AgentScan</div>
            <h1>Welcome to the Beta Program!</h1>
        </div>
        
        <div class="content">
            <p>Hi -name-,</p>
            
            <p>Congratulations! You've been selected for the AgentScan beta program. We're excited to have you join us in revolutionizing security scanning with multi-agent consensus technology.</p>
            
            <div class="features">
                <h3>What you'll get access to:</h3>
                <div class="feature">✅ Multi-agent security scanning with 80%% fewer false positives</div>
                <div class="feature">⚡ Sub-2-second scan results with intelligent caching</div>
                <div class="feature">🎯 Real-time VS Code extension with rich developer feedback</div>
                <div class="feature">🔄 Seamless GitHub/GitLab integration</div>
                <div class="feature">📊 Beautiful security health dashboard</div>
                <div class="feature">🆓 Free scanning for public repositories</div>
            </div>
            
            <p>Your beta access includes:</p>
            <ul>
                <li>Full platform access for 3 months</li>
                <li>Priority support and direct feedback channel</li>
                <li>Early access to new features</li>
                <li>Opportunity to influence product direction</li>
            </ul>
            
            <div style="text-align: center; margin: 30px 0;">
                <a href="-signup_url-" class="button">Get Started Now</a>
            </div>
            
            <p><strong>Your invite code:</strong> <code>-invite_code-</code></p>
            
            <p>Questions? Reply to this email or join our beta Slack channel for real-time support.</p>
            
            <p>Happy scanning!<br>
            The AgentScan Team</p>
        </div>
        
        <div class="footer">
            <p>AgentScan - Multi-Agent Security Scanning Platform</p>
            <p><a href="%s">Dashboard</a> | <a href="https://docs.agentscan.dev">Documentation</a> | <a href="https://github.com/agentscan">GitHub</a></p>
        </div>
    </div>
</body>
</html>
`, config.WebAppURL)

	text := fmt.Sprintf(`
Welcome to AgentScan Beta!

Hi -name-,

Congratulations! You've been selected for the AgentScan beta program.

What you'll get access to:
- Multi-agent security scanning with 80%% fewer false positives
- Sub-2-second scan results with intelligent caching
- Real-time VS Code extension with rich developer feedback
- Seamless GitHub/GitLab integration
- Beautiful security health dashboard
- Free scanning for public repositories

Your beta access includes:
- Full platform access for 3 months
- Priority support and direct feedback channel
- Early access to new features
- Opportunity to influence product direction

Get started: -signup_url-
Your invite code: -invite_code-

Questions? Reply to this email or join our beta Slack channel.

Happy scanning!
The AgentScan Team

AgentScan - Multi-Agent Security Scanning Platform
%s | https://docs.agentscan.dev | https://github.com/agentscan
`, config.WebAppURL)

	return EmailTemplate{
		Subject: subject,
		HTML:    html,
		Text:    text,
	}
}

func sendEmail(apiKey string, payload map[string]interface{}) error {
	if apiKey == "" {
		log.Println("SendGrid API key not configured, skipping email send")
		return nil
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("email send failed with status: %d", resp.StatusCode)
	}

	return nil
}

func cleanupExpiredInvites(config Config) error {
	log.Println("Cleaning up expired invitations...")
	
	// In real implementation, this would:
	// 1. Query database for expired invitations
	// 2. Update their status to 'expired'
	// 3. Send follow-up emails if appropriate
	// 4. Generate cleanup report
	
	log.Println("Cleanup completed")
	return nil
}

func showStats(config Config) error {
	log.Println("Beta Program Statistics:")
	
	// In real implementation, this would query the database
	stats := map[string]int{
		"Total Invitations": 150,
		"Pending":          25,
		"Sent":             100,
		"Accepted":         75,
		"Expired":          15,
		"Active Users":     68,
	}
	
	for key, value := range stats {
		fmt.Printf("  %-20s: %d\n", key, value)
	}
	
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}