package findings

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFindingStatus_String(t *testing.T) {
	tests := []struct {
		status   FindingStatus
		expected string
	}{
		{FindingStatusOpen, "open"},
		{FindingStatusFixed, "fixed"},
		{FindingStatusIgnored, "ignored"},
		{FindingStatusFalsePositive, "false_positive"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestExportFormat_String(t *testing.T) {
	tests := []struct {
		format   ExportFormat
		expected string
	}{
		{ExportFormatJSON, "json"},
		{ExportFormatPDF, "pdf"},
		{ExportFormatCSV, "csv"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.format))
		})
	}
}

func TestFindingFilter_Validation(t *testing.T) {
	filter := FindingFilter{
		Severity:      []string{"high", "medium"},
		Status:        []FindingStatus{FindingStatusOpen, FindingStatusFixed},
		Tool:          []string{"semgrep", "eslint"},
		MinConfidence: &[]float64{0.8}[0],
	}

	assert.Equal(t, []string{"high", "medium"}, filter.Severity)
	assert.Equal(t, []FindingStatus{FindingStatusOpen, FindingStatusFixed}, filter.Status)
	assert.Equal(t, []string{"semgrep", "eslint"}, filter.Tool)
	assert.Equal(t, 0.8, *filter.MinConfidence)
}

func TestFindingUpdate_Validation(t *testing.T) {
	status := FindingStatusFixed
	suggestion := "Use parameterized queries"
	
	update := FindingUpdate{
		Status:        &status,
		FixSuggestion: &suggestion,
	}

	assert.Equal(t, FindingStatusFixed, *update.Status)
	assert.Equal(t, "Use parameterized queries", *update.FixSuggestion)
}

func TestExportRequest_Validation(t *testing.T) {
	scanJobID := uuid.New()
	
	request := ExportRequest{
		ScanJobID: scanJobID,
		Format:    ExportFormatJSON,
		Filter: FindingFilter{
			Severity: []string{"high"},
		},
	}

	assert.Equal(t, scanJobID, request.ScanJobID)
	assert.Equal(t, ExportFormatJSON, request.Format)
	assert.Equal(t, []string{"high"}, request.Filter.Severity)
}