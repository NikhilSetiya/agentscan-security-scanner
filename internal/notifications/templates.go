package notifications

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	textTemplate "text/template"
	"time"
)

// DefaultTemplateManager implements the TemplateManager interface
type DefaultTemplateManager struct {
	textTemplates map[string]*textTemplate.Template
	htmlTemplates map[string]*template.Template
}

// NewDefaultTemplateManager creates a new template manager with default templates
func NewDefaultTemplateManager() *DefaultTemplateManager {
	tm := &DefaultTemplateManager{
		textTemplates: make(map[string]*textTemplate.Template),
		htmlTemplates: make(map[string]*template.Template),
	}

	tm.loadDefaultTemplates()
	return tm
}

// RenderScanCompleted renders a scan completed notification
func (tm *DefaultTemplateManager) RenderScanCompleted(notification ScanCompletedNotification, format string) (NotificationMessage, error) {
	templateName := "scan_completed"
	
	data := map[string]interface{}{
		"ScanID":       notification.ScanID.String(),
		"Repository":   notification.Repository,
		"Branch":       notification.Branch,
		"Commit":       notification.Commit[:8], // Short commit hash
		"Status":       notification.Status,
		"Duration":     formatDuration(notification.Duration),
		"FindingsCount": notification.FindingsCount,
		"DashboardURL": notification.DashboardURL,
		"Timestamp":    time.Now().Format("2006-01-02 15:04:05 UTC"),
	}

	subject, body, err := tm.renderTemplate(templateName, format, data)
	if err != nil {
		return NotificationMessage{}, err
	}

	return NotificationMessage{
		Subject: subject,
		Body:    body,
		Format:  format,
		Metadata: map[string]interface{}{
			"event_type":     "scan_completed",
			"repository":     notification.Repository,
			"branch":         notification.Branch,
			"commit":         notification.Commit,
			"status":         notification.Status,
			"findings_count": notification.FindingsCount,
			"dashboard_url":  notification.DashboardURL,
			"duration":       formatDuration(notification.Duration),
		},
	}, nil
}

// RenderCriticalFinding renders a critical finding notification
func (tm *DefaultTemplateManager) RenderCriticalFinding(notification CriticalFindingNotification, format string) (NotificationMessage, error) {
	templateName := "critical_finding"
	
	data := map[string]interface{}{
		"ScanID":      notification.ScanID.String(),
		"Repository":  notification.Repository,
		"Branch":      notification.Branch,
		"Commit":      notification.Commit[:8], // Short commit hash
		"Finding":     notification.Finding,
		"DashboardURL": notification.DashboardURL,
		"Timestamp":   time.Now().Format("2006-01-02 15:04:05 UTC"),
	}

	subject, body, err := tm.renderTemplate(templateName, format, data)
	if err != nil {
		return NotificationMessage{}, err
	}

	return NotificationMessage{
		Subject: subject,
		Body:    body,
		Format:  format,
		Metadata: map[string]interface{}{
			"event_type":    "critical_finding",
			"repository":    notification.Repository,
			"branch":        notification.Branch,
			"commit":        notification.Commit,
			"finding_id":    notification.Finding.ID,
			"severity":      notification.Finding.Severity,
			"dashboard_url": notification.DashboardURL,
		},
	}, nil
}

// RenderScanFailed renders a scan failed notification
func (tm *DefaultTemplateManager) RenderScanFailed(notification ScanFailedNotification, format string) (NotificationMessage, error) {
	templateName := "scan_failed"
	
	data := map[string]interface{}{
		"ScanID":      notification.ScanID.String(),
		"Repository":  notification.Repository,
		"Branch":      notification.Branch,
		"Commit":      notification.Commit[:8], // Short commit hash
		"Error":       notification.Error,
		"Duration":    formatDuration(notification.Duration),
		"DashboardURL": notification.DashboardURL,
		"Timestamp":   time.Now().Format("2006-01-02 15:04:05 UTC"),
	}

	subject, body, err := tm.renderTemplate(templateName, format, data)
	if err != nil {
		return NotificationMessage{}, err
	}

	return NotificationMessage{
		Subject: subject,
		Body:    body,
		Format:  format,
		Metadata: map[string]interface{}{
			"event_type":    "scan_failed",
			"repository":    notification.Repository,
			"branch":        notification.Branch,
			"commit":        notification.Commit,
			"error":         notification.Error,
			"dashboard_url": notification.DashboardURL,
			"duration":      formatDuration(notification.Duration),
		},
	}, nil
}

// renderTemplate renders a template with the given data
func (tm *DefaultTemplateManager) renderTemplate(templateName, format string, data map[string]interface{}) (string, string, error) {
	var subjectBuf, bodyBuf bytes.Buffer

	switch format {
	case "html":
		subjectTemplate, exists := tm.htmlTemplates[templateName+"_subject"]
		if !exists {
			return "", "", fmt.Errorf("HTML subject template not found: %s", templateName)
		}
		
		bodyTemplate, exists := tm.htmlTemplates[templateName+"_body"]
		if !exists {
			return "", "", fmt.Errorf("HTML body template not found: %s", templateName)
		}

		if err := subjectTemplate.Execute(&subjectBuf, data); err != nil {
			return "", "", fmt.Errorf("failed to execute HTML subject template: %w", err)
		}

		if err := bodyTemplate.Execute(&bodyBuf, data); err != nil {
			return "", "", fmt.Errorf("failed to execute HTML body template: %w", err)
		}

	case "markdown", "text":
		subjectTemplate, exists := tm.textTemplates[templateName+"_subject"]
		if !exists {
			return "", "", fmt.Errorf("text subject template not found: %s", templateName)
		}
		
		bodyTemplate, exists := tm.textTemplates[templateName+"_body"]
		if !exists {
			return "", "", fmt.Errorf("text body template not found: %s", templateName)
		}

		if err := subjectTemplate.Execute(&subjectBuf, data); err != nil {
			return "", "", fmt.Errorf("failed to execute text subject template: %w", err)
		}

		if err := bodyTemplate.Execute(&bodyBuf, data); err != nil {
			return "", "", fmt.Errorf("failed to execute text body template: %w", err)
		}

	default:
		return "", "", fmt.Errorf("unsupported format: %s", format)
	}

	return strings.TrimSpace(subjectBuf.String()), bodyBuf.String(), nil
}

// loadDefaultTemplates loads the default notification templates
func (tm *DefaultTemplateManager) loadDefaultTemplates() {
	// Scan Completed Templates
	tm.textTemplates["scan_completed_subject"] = textTemplate.Must(textTemplate.New("scan_completed_subject").Parse(
		"✅ Scan completed for {{.Repository}}",
	))

	tm.textTemplates["scan_completed_body"] = textTemplate.Must(textTemplate.New("scan_completed_body").Parse(
		`**Scan Completed Successfully**

Repository: {{.Repository}}
Branch: {{.Branch}}
Commit: {{.Commit}}
Status: {{.Status}}
Duration: {{.Duration}}

**Findings Summary:**
- High: {{.FindingsCount.High}}
- Medium: {{.FindingsCount.Medium}}
- Low: {{.FindingsCount.Low}}
- Total: {{.FindingsCount.Total}}

{{if .DashboardURL}}[View Results]({{.DashboardURL}}){{end}}

Scanned at {{.Timestamp}}`,
	))

	tm.htmlTemplates["scan_completed_subject"] = template.Must(template.New("scan_completed_subject").Parse(
		"✅ Scan completed for {{.Repository}}",
	))

	tm.htmlTemplates["scan_completed_body"] = template.Must(template.New("scan_completed_body").Parse(
		`<h2>Scan Completed Successfully</h2>

<table style="border-collapse: collapse; width: 100%;">
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Repository:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Repository}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Branch:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Branch}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Commit:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Commit}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Status:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Status}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Duration:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Duration}}</td></tr>
</table>

<h3>Findings Summary</h3>
<ul>
<li><span style="color: #d73a49;">High: {{.FindingsCount.High}}</span></li>
<li><span style="color: #fb8500;">Medium: {{.FindingsCount.Medium}}</span></li>
<li><span style="color: #28a745;">Low: {{.FindingsCount.Low}}</span></li>
<li><strong>Total: {{.FindingsCount.Total}}</strong></li>
</ul>

{{if .DashboardURL}}<p><a href="{{.DashboardURL}}" style="background-color: #0366d6; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">View Results</a></p>{{end}}

<p><small>Scanned at {{.Timestamp}}</small></p>`,
	))

	// Critical Finding Templates
	tm.textTemplates["critical_finding_subject"] = textTemplate.Must(textTemplate.New("critical_finding_subject").Parse(
		"🚨 Critical security finding in {{.Repository}}",
	))

	tm.textTemplates["critical_finding_body"] = textTemplate.Must(textTemplate.New("critical_finding_body").Parse(
		`**Critical Security Finding Detected**

Repository: {{.Repository}}
Branch: {{.Branch}}
Commit: {{.Commit}}

**Finding Details:**
- Title: {{.Finding.Title}}
- Severity: {{.Finding.Severity}}
- Category: {{.Finding.Category}}
- File: {{.Finding.File}}:{{.Finding.Line}}
- Tool: {{.Finding.Tool}}

**Description:**
{{.Finding.Description}}

{{if .DashboardURL}}[View Details]({{.DashboardURL}}){{end}}

Detected at {{.Timestamp}}`,
	))

	tm.htmlTemplates["critical_finding_subject"] = template.Must(template.New("critical_finding_subject").Parse(
		"🚨 Critical security finding in {{.Repository}}",
	))

	tm.htmlTemplates["critical_finding_body"] = template.Must(template.New("critical_finding_body").Parse(
		`<h2 style="color: #d73a49;">Critical Security Finding Detected</h2>

<table style="border-collapse: collapse; width: 100%;">
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Repository:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Repository}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Branch:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Branch}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Commit:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Commit}}</td></tr>
</table>

<h3>Finding Details</h3>
<table style="border-collapse: collapse; width: 100%;">
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Title:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Finding.Title}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Severity:</strong></td><td style="padding: 8px; border: 1px solid #ddd;"><span style="color: #d73a49; font-weight: bold;">{{.Finding.Severity}}</span></td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Category:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Finding.Category}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Location:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Finding.File}}:{{.Finding.Line}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Tool:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Finding.Tool}}</td></tr>
</table>

<h3>Description</h3>
<p>{{.Finding.Description}}</p>

{{if .DashboardURL}}<p><a href="{{.DashboardURL}}" style="background-color: #d73a49; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">View Details</a></p>{{end}}

<p><small>Detected at {{.Timestamp}}</small></p>`,
	))

	// Scan Failed Templates
	tm.textTemplates["scan_failed_subject"] = textTemplate.Must(textTemplate.New("scan_failed_subject").Parse(
		"❌ Scan failed for {{.Repository}}",
	))

	tm.textTemplates["scan_failed_body"] = textTemplate.Must(textTemplate.New("scan_failed_body").Parse(
		`**Scan Failed**

Repository: {{.Repository}}
Branch: {{.Branch}}
Commit: {{.Commit}}
Duration: {{.Duration}}

**Error:**
{{.Error}}

{{if .DashboardURL}}[View Details]({{.DashboardURL}}){{end}}

Failed at {{.Timestamp}}`,
	))

	tm.htmlTemplates["scan_failed_subject"] = template.Must(template.New("scan_failed_subject").Parse(
		"❌ Scan failed for {{.Repository}}",
	))

	tm.htmlTemplates["scan_failed_body"] = template.Must(template.New("scan_failed_body").Parse(
		`<h2 style="color: #d73a49;">Scan Failed</h2>

<table style="border-collapse: collapse; width: 100%;">
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Repository:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Repository}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Branch:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Branch}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Commit:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Commit}}</td></tr>
<tr><td style="padding: 8px; border: 1px solid #ddd;"><strong>Duration:</strong></td><td style="padding: 8px; border: 1px solid #ddd;">{{.Duration}}</td></tr>
</table>

<h3>Error Details</h3>
<div style="background-color: #f8f8f8; padding: 15px; border-left: 4px solid #d73a49; margin: 15px 0;">
<pre style="margin: 0; white-space: pre-wrap;">{{.Error}}</pre>
</div>

{{if .DashboardURL}}<p><a href="{{.DashboardURL}}" style="background-color: #6f42c1; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">View Details</a></p>{{end}}

<p><small>Failed at {{.Timestamp}}</small></p>`,
	))
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}