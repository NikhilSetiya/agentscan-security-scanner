package notifications

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTemplateManager_RenderScanCompleted(t *testing.T) {
	tm := NewDefaultTemplateManager()

	notification := ScanCompletedNotification{
		ScanID:     uuid.New(),
		Repository: "test/repo",
		Branch:     "main",
		Commit:     "abc123def456789",
		Status:     "completed",
		Duration:   2*time.Minute + 30*time.Second,
		FindingsCount: FindingsCount{
			High:   2,
			Medium: 5,
			Low:    3,
			Total:  10,
		},
		DashboardURL: "https://dashboard.example.com/scan/123",
		UserID:       uuid.New(),
	}

	t.Run("markdown format", func(t *testing.T) {
		message, err := tm.RenderScanCompleted(notification, "markdown")
		require.NoError(t, err)

		assert.Equal(t, "✅ Scan completed for test/repo", message.Subject)
		assert.Contains(t, message.Body, "**Scan Completed Successfully**")
		assert.Contains(t, message.Body, "Repository: test/repo")
		assert.Contains(t, message.Body, "Branch: main")
		assert.Contains(t, message.Body, "Commit: abc123de")
		assert.Contains(t, message.Body, "Status: completed")
		assert.Contains(t, message.Body, "Duration: 2.5m")
		assert.Contains(t, message.Body, "High: 2")
		assert.Contains(t, message.Body, "Medium: 5")
		assert.Contains(t, message.Body, "Low: 3")
		assert.Contains(t, message.Body, "Total: 10")
		assert.Contains(t, message.Body, "[View Results](https://dashboard.example.com/scan/123)")
		assert.Equal(t, "markdown", message.Format)

		// Check metadata
		assert.Equal(t, "scan_completed", message.Metadata["event_type"])
		assert.Equal(t, "test/repo", message.Metadata["repository"])
		assert.Equal(t, "main", message.Metadata["branch"])
		assert.Equal(t, notification.FindingsCount, message.Metadata["findings_count"])
	})

	t.Run("html format", func(t *testing.T) {
		message, err := tm.RenderScanCompleted(notification, "html")
		require.NoError(t, err)

		assert.Equal(t, "✅ Scan completed for test/repo", message.Subject)
		assert.Contains(t, message.Body, "<h2>Scan Completed Successfully</h2>")
		assert.Contains(t, message.Body, "<table")
		assert.Contains(t, message.Body, "test/repo")
		assert.Contains(t, message.Body, "main")
		assert.Contains(t, message.Body, "abc123de")
		assert.Contains(t, message.Body, "completed")
		assert.Contains(t, message.Body, "2.5m")
		assert.Contains(t, message.Body, `<span style="color: #d73a49;">High: 2</span>`)
		assert.Contains(t, message.Body, `<span style="color: #fb8500;">Medium: 5</span>`)
		assert.Contains(t, message.Body, `<span style="color: #28a745;">Low: 3</span>`)
		assert.Contains(t, message.Body, `href="https://dashboard.example.com/scan/123"`)
		assert.Equal(t, "html", message.Format)
	})
}

func TestDefaultTemplateManager_RenderCriticalFinding(t *testing.T) {
	tm := NewDefaultTemplateManager()

	notification := CriticalFindingNotification{
		ScanID:     uuid.New(),
		Repository: "test/repo",
		Branch:     "main",
		Commit:     "abc123def456789",
		Finding: CriticalFinding{
			ID:          "finding-123",
			Title:       "SQL Injection vulnerability",
			Severity:    "high",
			Category:    "sql_injection",
			File:        "app.js",
			Line:        42,
			Tool:        "semgrep",
			Description: "Potential SQL injection detected in user input handling",
		},
		DashboardURL: "https://dashboard.example.com/finding/123",
		UserID:       uuid.New(),
	}

	t.Run("markdown format", func(t *testing.T) {
		message, err := tm.RenderCriticalFinding(notification, "markdown")
		require.NoError(t, err)

		assert.Equal(t, "🚨 Critical security finding in test/repo", message.Subject)
		assert.Contains(t, message.Body, "**Critical Security Finding Detected**")
		assert.Contains(t, message.Body, "Repository: test/repo")
		assert.Contains(t, message.Body, "Branch: main")
		assert.Contains(t, message.Body, "Commit: abc123de")
		assert.Contains(t, message.Body, "Title: SQL Injection vulnerability")
		assert.Contains(t, message.Body, "Severity: high")
		assert.Contains(t, message.Body, "Category: sql_injection")
		assert.Contains(t, message.Body, "File: app.js:42")
		assert.Contains(t, message.Body, "Tool: semgrep")
		assert.Contains(t, message.Body, "Potential SQL injection detected")
		assert.Contains(t, message.Body, "[View Details](https://dashboard.example.com/finding/123)")
		assert.Equal(t, "markdown", message.Format)

		// Check metadata
		assert.Equal(t, "critical_finding", message.Metadata["event_type"])
		assert.Equal(t, "finding-123", message.Metadata["finding_id"])
		assert.Equal(t, "high", message.Metadata["severity"])
	})

	t.Run("html format", func(t *testing.T) {
		message, err := tm.RenderCriticalFinding(notification, "html")
		require.NoError(t, err)

		assert.Equal(t, "🚨 Critical security finding in test/repo", message.Subject)
		assert.Contains(t, message.Body, `<h2 style="color: #d73a49;">Critical Security Finding Detected</h2>`)
		assert.Contains(t, message.Body, "SQL Injection vulnerability")
		assert.Contains(t, message.Body, `<span style="color: #d73a49; font-weight: bold;">high</span>`)
		assert.Contains(t, message.Body, "app.js:42")
		assert.Contains(t, message.Body, "semgrep")
		assert.Contains(t, message.Body, "Potential SQL injection detected")
		assert.Contains(t, message.Body, `href="https://dashboard.example.com/finding/123"`)
		assert.Equal(t, "html", message.Format)
	})
}

func TestDefaultTemplateManager_RenderScanFailed(t *testing.T) {
	tm := NewDefaultTemplateManager()

	notification := ScanFailedNotification{
		ScanID:       uuid.New(),
		Repository:   "test/repo",
		Branch:       "main",
		Commit:       "abc123def456789",
		Error:        "Docker container failed to start: image not found",
		Duration:     30 * time.Second,
		DashboardURL: "https://dashboard.example.com/scan/123",
		UserID:       uuid.New(),
	}

	t.Run("markdown format", func(t *testing.T) {
		message, err := tm.RenderScanFailed(notification, "markdown")
		require.NoError(t, err)

		assert.Equal(t, "❌ Scan failed for test/repo", message.Subject)
		assert.Contains(t, message.Body, "**Scan Failed**")
		assert.Contains(t, message.Body, "Repository: test/repo")
		assert.Contains(t, message.Body, "Branch: main")
		assert.Contains(t, message.Body, "Commit: abc123de")
		assert.Contains(t, message.Body, "Duration: 30.0s")
		assert.Contains(t, message.Body, "Docker container failed to start: image not found")
		assert.Contains(t, message.Body, "[View Details](https://dashboard.example.com/scan/123)")
		assert.Equal(t, "markdown", message.Format)

		// Check metadata
		assert.Equal(t, "scan_failed", message.Metadata["event_type"])
		assert.Equal(t, "Docker container failed to start: image not found", message.Metadata["error"])
	})

	t.Run("html format", func(t *testing.T) {
		message, err := tm.RenderScanFailed(notification, "html")
		require.NoError(t, err)

		assert.Equal(t, "❌ Scan failed for test/repo", message.Subject)
		assert.Contains(t, message.Body, `<h2 style="color: #d73a49;">Scan Failed</h2>`)
		assert.Contains(t, message.Body, "test/repo")
		assert.Contains(t, message.Body, "main")
		assert.Contains(t, message.Body, "abc123de")
		assert.Contains(t, message.Body, "30.0s")
		assert.Contains(t, message.Body, "Docker container failed to start: image not found")
		assert.Contains(t, message.Body, `href="https://dashboard.example.com/scan/123"`)
		assert.Equal(t, "html", message.Format)
	})
}

func TestDefaultTemplateManager_UnsupportedFormat(t *testing.T) {
	tm := NewDefaultTemplateManager()

	notification := ScanCompletedNotification{
		ScanID:     uuid.New(),
		Repository: "test/repo",
		Branch:     "main",
		Commit:     "abc123def456789",
		Status:     "completed",
		Duration:   time.Minute,
		FindingsCount: FindingsCount{
			Total: 5,
		},
		UserID: uuid.New(),
	}

	_, err := tm.RenderScanCompleted(notification, "unsupported")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format: unsupported")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds",
			duration: 30 * time.Second,
			expected: "30.0s",
		},
		{
			name:     "minutes",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2.5m",
		},
		{
			name:     "hours",
			duration: 1*time.Hour + 30*time.Minute,
			expected: "1.5h",
		},
		{
			name:     "sub-second",
			duration: 500 * time.Millisecond,
			expected: "0.5s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultTemplateManager_TemplateExecution(t *testing.T) {
	tm := NewDefaultTemplateManager()

	// Test that all templates can be executed without errors
	testData := map[string]interface{}{
		"ScanID":     uuid.New().String(),
		"Repository": "test/repo",
		"Branch":     "main",
		"Commit":     "abc123de",
		"Status":     "completed",
		"Duration":   "2.5m",
		"FindingsCount": FindingsCount{
			High:   1,
			Medium: 2,
			Low:    3,
			Total:  6,
		},
		"DashboardURL": "https://dashboard.example.com",
		"Timestamp":    time.Now().Format("2006-01-02 15:04:05 UTC"),
		"Finding": CriticalFinding{
			ID:          "finding-123",
			Title:       "Test Finding",
			Severity:    "high",
			Category:    "test",
			File:        "test.js",
			Line:        42,
			Tool:        "test-tool",
			Description: "Test description",
		},
		"Error": "Test error message",
	}

	templates := []string{
		"scan_completed_subject",
		"scan_completed_body",
		"critical_finding_subject",
		"critical_finding_body",
		"scan_failed_subject",
		"scan_failed_body",
	}

	for _, templateName := range templates {
		t.Run(templateName, func(t *testing.T) {
			// Test text template
			if textTemplate, exists := tm.textTemplates[templateName]; exists {
				var buf strings.Builder
				err := textTemplate.Execute(&buf, testData)
				require.NoError(t, err)
				assert.NotEmpty(t, buf.String())
			}

			// Test HTML template
			if htmlTemplate, exists := tm.htmlTemplates[templateName]; exists {
				var buf strings.Builder
				err := htmlTemplate.Execute(&buf, testData)
				require.NoError(t, err)
				assert.NotEmpty(t, buf.String())
			}
		})
	}
}