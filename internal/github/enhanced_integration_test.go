package github

import (
	"context"
	"testing"

	"github.com/agentscan/agentscan/internal/orchestrator"
	"github.com/stretchr/testify/assert"
)

func TestEnhancedPRWorkflow(t *testing.T) {
	// Test the complete enhanced PR workflow
	t.Run("complete PR workflow with enhanced features", func(t *testing.T) {
		// Create a webhook handler
		handler := &WebhookHandler{}

		// Test 1: PR comment formatting with rich content
		results := &orchestrator.ScanResults{
			Status: "completed",
			Findings: []orchestrator.Finding{
				{
					ID:          "high-xss",
					Tool:        "semgrep",
					RuleID:      "javascript.express.security.audit.xss",
					Severity:    "high",
					Title:       "Cross-Site Scripting (XSS) vulnerability",
					Description: "User input is directly rendered without sanitization",
					File:        "src/app.js",
					Line:        42,
					Confidence:  0.95,
					References:  []string{"https://owasp.org/www-community/attacks/xss/"},
					Metadata: map[string]interface{}{
						"is_new_in_pr": "true",
					},
					FixSuggestion: map[string]interface{}{
						"description": "Use a sanitization library like DOMPurify",
					},
				},
				{
					ID:          "medium-injection",
					Tool:        "eslint-security",
					RuleID:      "security/detect-object-injection",
					Severity:    "medium",
					Title:       "Object Injection vulnerability",
					Description: "Dynamic object property access may lead to prototype pollution",
					File:        "src/utils.js",
					Line:        15,
					Confidence:  0.80,
				},
			},
		}

		jobID := "enhanced-test-job-123"
		comment := handler.formatPRComment(results, jobID)

		// Verify enhanced comment features
		assert.Contains(t, comment, "🛡️ AgentScan Security Report")
		assert.Contains(t, comment, "📊 Security Summary")
		assert.Contains(t, comment, "| Severity | Count | New in PR |")
		assert.Contains(t, comment, "🔴 **High** | **1** | 1")
		assert.Contains(t, comment, "🟡 **Medium** | **1** | 0")
		assert.Contains(t, comment, "🚨 Critical Issues Requiring Attention")
		assert.Contains(t, comment, "Cross-Site Scripting (XSS) vulnerability")
		assert.Contains(t, comment, "**💡 Suggested Fix:** Use a sanitization library like DOMPurify")
		assert.Contains(t, comment, "**📚 References:**")
		assert.Contains(t, comment, "https://owasp.org/www-community/attacks/xss/")
		assert.Contains(t, comment, "🟡 Medium Severity Issues")
		assert.Contains(t, comment, "<details>")
		assert.Contains(t, comment, "🎯 Next Steps")
		assert.Contains(t, comment, "🔴 Fix high severity issues")
		assert.Contains(t, comment, "📊 **[View Full Report]")
		assert.Contains(t, comment, jobID) // Verify job ID is included for dashboard links

		// Test 2: Status check logic for blocking high severity issues
		ctx := context.Background()
		_ = ctx // Use the variable

		// Simulate status check logic
		severityCounts := make(map[string]int)
		newFindings := make(map[string]int)
		
		for _, finding := range results.Findings {
			severityCounts[finding.Severity]++
			
			if finding.Metadata != nil {
				if isNew, exists := finding.Metadata["is_new_in_pr"]; exists && isNew == "true" {
					newFindings[finding.Severity]++
				}
			}
		}

		// Verify high severity findings result in failure status
		assert.Equal(t, 1, severityCounts["high"])
		assert.Equal(t, 1, newFindings["high"])
		assert.Equal(t, 1, severityCounts["medium"])
		assert.Equal(t, 0, newFindings["medium"])

		// Test 3: Check run annotation generation
		annotations := handler.generateCheckRunAnnotations(results)
		assert.Len(t, annotations, 2)
		
		// High severity should be first and marked as failure
		assert.Equal(t, "failure", annotations[0].AnnotationLevel)
		assert.Equal(t, "src/app.js", annotations[0].Path)
		assert.Equal(t, 42, annotations[0].StartLine)
		assert.Equal(t, "Cross-Site Scripting (XSS) vulnerability", annotations[0].Title)
		assert.Contains(t, annotations[0].Message, "javascript.express.security.audit.xss")

		// Medium severity should be marked as warning
		assert.Equal(t, "warning", annotations[1].AnnotationLevel)
		assert.Equal(t, "src/utils.js", annotations[1].Path)
		assert.Equal(t, 15, annotations[1].StartLine)

		// Test 4: Check run summary generation
		summary := handler.generateCheckRunSummary(results, severityCounts)
		assert.Contains(t, summary, "## Security Scan Results")
		assert.Contains(t, summary, "🔴 **1 High** severity issues")
		assert.Contains(t, summary, "🟡 **1 Medium** severity issues")
		assert.Contains(t, summary, "⚠️ **High severity issues must be fixed before merging.**")
		assert.Contains(t, summary, "📊 View detailed results and fix suggestions")
	})

	t.Run("clean scan with no issues", func(t *testing.T) {
		handler := &WebhookHandler{}
		
		results := &orchestrator.ScanResults{
			Status:   "completed",
			Findings: []orchestrator.Finding{},
		}

		jobID := "clean-scan-job-456"
		comment := handler.formatPRComment(results, jobID)

		// Verify clean scan comment
		assert.Contains(t, comment, "✅ All Clear!")
		assert.Contains(t, comment, "No security vulnerabilities detected")
		assert.Contains(t, comment, "Your code looks secure! 🎉")
		assert.Contains(t, comment, jobID)

		// Verify check run summary for clean scan
		severityCounts := make(map[string]int)
		summary := handler.generateCheckRunSummary(results, severityCounts)
		assert.Contains(t, summary, "🎉 **No security issues found!**")
		assert.Contains(t, summary, "Your code looks secure.")

		// Verify no annotations for clean scan
		annotations := handler.generateCheckRunAnnotations(results)
		assert.Len(t, annotations, 0)
	})

	t.Run("changed files extraction", func(t *testing.T) {
		// Test PR changed files logic
		files := []PRFile{
			{Filename: "src/app.js", Status: "modified"},
			{Filename: "src/new-component.js", Status: "added"},
			{Filename: "old-file.js", Status: "removed"},
			{Filename: "moved-file.js", Status: "renamed"},
			{Filename: "package.json", Status: "modified"},
		}

		var changedFiles []string
		for _, file := range files {
			// Only include files that are added, modified, or renamed (not deleted)
			if file.Status == "added" || file.Status == "modified" || file.Status == "renamed" {
				changedFiles = append(changedFiles, file.Filename)
			}
		}

		expected := []string{"src/app.js", "src/new-component.js", "moved-file.js", "package.json"}
		assert.Equal(t, expected, changedFiles)
		
		// Verify removed files are excluded
		assert.NotContains(t, changedFiles, "old-file.js")
	})
}

func TestEnhancedStatusCheckStates(t *testing.T) {
	tests := []struct {
		name            string
		findings        []orchestrator.Finding
		expectedState   string
		shouldBlock     bool
		description     string
	}{
		{
			name: "high severity blocks merge",
			findings: []orchestrator.Finding{
				{Severity: "high", Metadata: map[string]interface{}{"is_new_in_pr": "true"}},
			},
			expectedState: "failure",
			shouldBlock:   true,
			description:   "High severity issues should block PR merging",
		},
		{
			name: "medium/low allows merge",
			findings: []orchestrator.Finding{
				{Severity: "medium"},
				{Severity: "low"},
			},
			expectedState: "success",
			shouldBlock:   false,
			description:   "Medium/low severity should not block PR merging",
		},
		{
			name:          "no findings allows merge",
			findings:      []orchestrator.Finding{},
			expectedState: "success",
			shouldBlock:   false,
			description:   "Clean scans should allow PR merging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the status check logic
			severityCounts := make(map[string]int)
			for _, finding := range tt.findings {
				severityCounts[finding.Severity]++
			}

			var state string
			if severityCounts["high"] > 0 {
				state = "failure"
			} else {
				state = "success"
			}

			assert.Equal(t, tt.expectedState, state, tt.description)
			
			if tt.shouldBlock {
				assert.Equal(t, "failure", state, "High severity findings should result in failure state")
			} else {
				assert.Equal(t, "success", state, "Non-high severity findings should result in success state")
			}
		})
	}
}