package findings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
)

// ExportService handles exporting findings to various formats
type ExportService struct {
	storageURL string // Base URL for accessing exported files
}

// NewExportService creates a new export service
func NewExportService(storageURL string) *ExportService {
	return &ExportService{
		storageURL: storageURL,
	}
}

// Export exports findings to the specified format
func (e *ExportService) Export(ctx context.Context, findings []*Finding, format ExportFormat) (*ExportResult, error) {
	switch format {
	case ExportFormatJSON:
		return e.exportJSON(ctx, findings)
	case ExportFormatPDF:
		return e.exportPDF(ctx, findings)
	case ExportFormatCSV:
		return e.exportCSV(ctx, findings)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportJSON exports findings as JSON
func (e *ExportService) exportJSON(ctx context.Context, findings []*Finding) (*ExportResult, error) {
	exportData := map[string]interface{}{
		"export_info": map[string]interface{}{
			"generated_at": time.Now(),
			"format":       "json",
			"version":      "1.0",
			"total_count":  len(findings),
		},
		"findings": findings,
		"summary": e.generateSummary(findings),
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	filename := fmt.Sprintf("findings_%s.json", time.Now().Format("20060102_150405"))
	
	// In a real implementation, you would save this to S3 or similar storage
	// For now, we'll simulate the file creation
	result := &ExportResult{
		ID:          uuid.New(),
		Format:      ExportFormatJSON,
		Filename:    filename,
		URL:         fmt.Sprintf("%s/exports/%s", e.storageURL, filename),
		Size:        int64(len(data)),
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Expire after 24 hours
	}

	return result, nil
}

// exportPDF exports findings as PDF
func (e *ExportService) exportPDF(ctx context.Context, findings []*Finding) (*ExportResult, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Security Scan Report")
	pdf.Ln(15)

	// Add summary
	summary := e.generateSummary(findings)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 10, "Summary")
	pdf.Ln(8)
	
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 6, fmt.Sprintf("Total Findings: %d", summary.Total))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("High Severity: %d", summary.High))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("Medium Severity: %d", summary.Medium))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("Low Severity: %d", summary.Low))
	pdf.Ln(12)

	// Add findings details
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 10, "Findings Details")
	pdf.Ln(10)

	for i, finding := range findings {
		if i > 0 {
			pdf.Ln(5)
		}

		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(40, 6, fmt.Sprintf("%d. %s", i+1, finding.Title))
		pdf.Ln(6)

		pdf.SetFont("Arial", "", 9)
		pdf.Cell(40, 5, fmt.Sprintf("Severity: %s | Tool: %s | Rule: %s", finding.Severity, finding.Tool, finding.RuleID))
		pdf.Ln(5)
		pdf.Cell(40, 5, fmt.Sprintf("File: %s", finding.FilePath))
		if finding.LineNumber != nil {
			pdf.Cell(40, 5, fmt.Sprintf(" (Line %d)", *finding.LineNumber))
		}
		pdf.Ln(5)

		if finding.Description != "" {
			// Wrap long descriptions
			pdf.MultiCell(0, 4, finding.Description, "", "", false)
			pdf.Ln(2)
		}

		// Add page break if needed
		if pdf.GetY() > 250 {
			pdf.AddPage()
		}
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	filename := fmt.Sprintf("findings_%s.pdf", time.Now().Format("20060102_150405"))
	
	result := &ExportResult{
		ID:          uuid.New(),
		Format:      ExportFormatPDF,
		Filename:    filename,
		URL:         fmt.Sprintf("%s/exports/%s", e.storageURL, filename),
		Size:        int64(buf.Len()),
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	return result, nil
}

// exportCSV exports findings as CSV
func (e *ExportService) exportCSV(ctx context.Context, findings []*Finding) (*ExportResult, error) {
	var buf bytes.Buffer
	
	// Write CSV header
	buf.WriteString("ID,Tool,Rule ID,Severity,Category,Title,Description,File Path,Line Number,Status,Confidence,Created At\n")
	
	// Write findings data
	for _, finding := range findings {
		lineNumber := ""
		if finding.LineNumber != nil {
			lineNumber = fmt.Sprintf("%d", *finding.LineNumber)
		}
		
		// Escape CSV fields
		title := escapeCSVField(finding.Title)
		description := escapeCSVField(finding.Description)
		filePath := escapeCSVField(finding.FilePath)
		
		buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%.2f,%s\n",
			finding.ID.String(),
			finding.Tool,
			finding.RuleID,
			finding.Severity,
			finding.Category,
			title,
			description,
			filePath,
			lineNumber,
			string(finding.Status),
			finding.Confidence,
			finding.CreatedAt.Format("2006-01-02 15:04:05"),
		))
	}

	filename := fmt.Sprintf("findings_%s.csv", time.Now().Format("20060102_150405"))
	
	result := &ExportResult{
		ID:          uuid.New(),
		Format:      ExportFormatCSV,
		Filename:    filename,
		URL:         fmt.Sprintf("%s/exports/%s", e.storageURL, filename),
		Size:        int64(buf.Len()),
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	return result, nil
}

// generateSummary creates a summary of findings
func (e *ExportService) generateSummary(findings []*Finding) FindingStats {
	stats := FindingStats{}
	
	for _, finding := range findings {
		stats.Total++
		
		switch finding.Severity {
		case "high":
			stats.High++
		case "medium":
			stats.Medium++
		case "low":
			stats.Low++
		}
		
		switch finding.Status {
		case FindingStatusOpen:
			stats.Open++
		case FindingStatusFixed:
			stats.Fixed++
		case FindingStatusIgnored:
			stats.Ignored++
		case FindingStatusFalsePositive:
			stats.FalsePositives++
		}
	}
	
	return stats
}

// escapeCSVField escapes a field for CSV output
func escapeCSVField(field string) string {
	// If field contains comma, newline, or quote, wrap in quotes and escape internal quotes
	if containsSpecialChars(field) {
		field = `"` + escapeQuotes(field) + `"`
	}
	return field
}

// containsSpecialChars checks if a string contains CSV special characters
func containsSpecialChars(s string) bool {
	for _, char := range s {
		if char == ',' || char == '\n' || char == '\r' || char == '"' {
			return true
		}
	}
	return false
}

// escapeQuotes escapes quotes in a string for CSV
func escapeQuotes(s string) string {
	return fmt.Sprintf(`%s`, template.HTMLEscapeString(s))
}

// GenerateReportTemplate generates an HTML template for reports
func (e *ExportService) GenerateReportTemplate(findings []*Finding) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Security Scan Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { border-bottom: 2px solid #333; padding-bottom: 20px; margin-bottom: 30px; }
        .summary { background: #f5f5f5; padding: 20px; margin-bottom: 30px; border-radius: 5px; }
        .finding { border: 1px solid #ddd; margin-bottom: 20px; padding: 15px; border-radius: 5px; }
        .severity-high { border-left: 5px solid #dc2626; }
        .severity-medium { border-left: 5px solid #d97706; }
        .severity-low { border-left: 5px solid #2563eb; }
        .finding-title { font-weight: bold; font-size: 16px; margin-bottom: 10px; }
        .finding-meta { color: #666; font-size: 14px; margin-bottom: 10px; }
        .finding-description { margin-bottom: 10px; }
        .code-snippet { background: #f8f8f8; padding: 10px; border-radius: 3px; font-family: monospace; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Security Scan Report</h1>
        <p>Generated on: {{.GeneratedAt}}</p>
    </div>
    
    <div class="summary">
        <h2>Summary</h2>
        <p>Total Findings: {{.Summary.Total}}</p>
        <p>High Severity: {{.Summary.High}}</p>
        <p>Medium Severity: {{.Summary.Medium}}</p>
        <p>Low Severity: {{.Summary.Low}}</p>
    </div>
    
    <div class="findings">
        <h2>Findings Details</h2>
        {{range $index, $finding := .Findings}}
        <div class="finding severity-{{$finding.Severity}}">
            <div class="finding-title">{{$finding.Title}}</div>
            <div class="finding-meta">
                Severity: {{$finding.Severity}} | Tool: {{$finding.Tool}} | Rule: {{$finding.RuleID}}
            </div>
            <div class="finding-meta">
                File: {{$finding.FilePath}}{{if $finding.LineNumber}} (Line {{$finding.LineNumber}}){{end}}
            </div>
            {{if $finding.Description}}
            <div class="finding-description">{{$finding.Description}}</div>
            {{end}}
            {{if $finding.CodeSnippet}}
            <div class="code-snippet">{{$finding.CodeSnippet}}</div>
            {{end}}
        </div>
        {{end}}
    </div>
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return "", err
	}

	data := struct {
		GeneratedAt time.Time
		Summary     FindingStats
		Findings    []*Finding
	}{
		GeneratedAt: time.Now(),
		Summary:     e.generateSummary(findings),
		Findings:    findings,
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}