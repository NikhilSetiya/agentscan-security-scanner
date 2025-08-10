package findings

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportService_ExportJSON(t *testing.T) {
	exporter := NewExportService("http://localhost:8080")

	// Create test findings
	findings := []*Finding{
		{
			ID:          uuid.New(),
			Tool:        "semgrep",
			RuleID:      "test-rule-1",
			Severity:    "high",
			Category:    "security",
			Title:       "SQL Injection",
			Description: "Potential SQL injection vulnerability",
			FilePath:    "app.js",
			LineNumber:  &[]int{42}[0],
			Status:      FindingStatusOpen,
			Confidence:  0.95,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Tool:        "eslint",
			RuleID:      "test-rule-2",
			Severity:    "medium",
			Category:    "security",
			Title:       "XSS Vulnerability",
			Description: "Potential XSS vulnerability",
			FilePath:    "ui.js",
			LineNumber:  &[]int{15}[0],
			Status:      FindingStatusOpen,
			Confidence:  0.87,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	result, err := exporter.Export(context.Background(), findings, ExportFormatJSON)
	require.NoError(t, err)

	assert.Equal(t, ExportFormatJSON, result.Format)
	assert.Contains(t, result.Filename, "findings_")
	assert.Contains(t, result.Filename, ".json")
	assert.Contains(t, result.URL, "http://localhost:8080/exports/")
	assert.Greater(t, result.Size, int64(0))
	assert.False(t, result.GeneratedAt.IsZero())
	assert.True(t, result.ExpiresAt.After(result.GeneratedAt))
}

func TestExportService_ExportPDF(t *testing.T) {
	exporter := NewExportService("http://localhost:8080")

	// Create test findings
	findings := []*Finding{
		{
			ID:          uuid.New(),
			Tool:        "semgrep",
			RuleID:      "test-rule-1",
			Severity:    "high",
			Category:    "security",
			Title:       "SQL Injection",
			Description: "Potential SQL injection vulnerability",
			FilePath:    "app.js",
			LineNumber:  &[]int{42}[0],
			Status:      FindingStatusOpen,
			Confidence:  0.95,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	result, err := exporter.Export(context.Background(), findings, ExportFormatPDF)
	require.NoError(t, err)

	assert.Equal(t, ExportFormatPDF, result.Format)
	assert.Contains(t, result.Filename, "findings_")
	assert.Contains(t, result.Filename, ".pdf")
	assert.Greater(t, result.Size, int64(0))
}

func TestExportService_ExportCSV(t *testing.T) {
	exporter := NewExportService("http://localhost:8080")

	// Create test findings
	findings := []*Finding{
		{
			ID:          uuid.New(),
			Tool:        "semgrep",
			RuleID:      "test-rule-1",
			Severity:    "high",
			Category:    "security",
			Title:       "SQL Injection",
			Description: "Potential SQL injection vulnerability",
			FilePath:    "app.js",
			LineNumber:  &[]int{42}[0],
			Status:      FindingStatusOpen,
			Confidence:  0.95,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	result, err := exporter.Export(context.Background(), findings, ExportFormatCSV)
	require.NoError(t, err)

	assert.Equal(t, ExportFormatCSV, result.Format)
	assert.Contains(t, result.Filename, "findings_")
	assert.Contains(t, result.Filename, ".csv")
	assert.Greater(t, result.Size, int64(0))
}

func TestExportService_GenerateSummary(t *testing.T) {
	exporter := NewExportService("http://localhost:8080")

	findings := []*Finding{
		{Severity: "high", Status: FindingStatusOpen},
		{Severity: "high", Status: FindingStatusFixed},
		{Severity: "medium", Status: FindingStatusOpen},
		{Severity: "low", Status: FindingStatusIgnored},
		{Severity: "low", Status: FindingStatusFalsePositive},
	}

	summary := exporter.generateSummary(findings)

	assert.Equal(t, 5, summary.Total)
	assert.Equal(t, 2, summary.High)
	assert.Equal(t, 1, summary.Medium)
	assert.Equal(t, 2, summary.Low)
	assert.Equal(t, 2, summary.Open)
	assert.Equal(t, 1, summary.Fixed)
	assert.Equal(t, 1, summary.Ignored)
	assert.Equal(t, 1, summary.FalsePositives)
}

func TestExportService_UnsupportedFormat(t *testing.T) {
	exporter := NewExportService("http://localhost:8080")

	findings := []*Finding{}
	_, err := exporter.Export(context.Background(), findings, ExportFormat("unsupported"))
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestExportService_GenerateReportTemplate(t *testing.T) {
	exporter := NewExportService("http://localhost:8080")

	findings := []*Finding{
		{
			ID:          uuid.New(),
			Tool:        "semgrep",
			RuleID:      "test-rule-1",
			Severity:    "high",
			Title:       "SQL Injection",
			Description: "Potential SQL injection vulnerability",
			FilePath:    "app.js",
			LineNumber:  &[]int{42}[0],
		},
	}

	html, err := exporter.GenerateReportTemplate(findings)
	require.NoError(t, err)

	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "Security Scan Report")
	assert.Contains(t, html, "SQL Injection")
	assert.Contains(t, html, "app.js")
	assert.Contains(t, html, "high")
}