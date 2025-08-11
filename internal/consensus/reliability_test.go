package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/pkg/agent"
)

func TestReliabilityTracker_UpdateReliability(t *testing.T) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	feedback := UserFeedback{
		FindingID:  "finding-1",
		UserID:     "user-1",
		Action:     "confirmed",
		Comment:    "This is a real vulnerability",
		Confidence: 0.9,
		Timestamp:  time.Now(),
	}

	err := tracker.UpdateReliability(ctx, "semgrep", feedback)
	require.NoError(t, err)

	// Check that reliability is updated
	reliability := tracker.GetReliability(ctx, "semgrep")
	assert.GreaterOrEqual(t, reliability, 0.0)
	assert.LessOrEqual(t, reliability, 1.0)
}

func TestReliabilityTracker_GetReliability(t *testing.T) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Test unknown tool (should return default)
	reliability := tracker.GetReliability(ctx, "unknown-tool")
	assert.Equal(t, 0.5, reliability)

	// Add some feedback
	feedbacks := []UserFeedback{
		{
			FindingID:  "finding-1",
			Action:     "confirmed",
			Confidence: 0.9,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-2",
			Action:     "confirmed",
			Confidence: 0.8,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-3",
			Action:     "false_positive",
			Confidence: 0.7,
			Timestamp:  time.Now(),
		},
	}

	for _, fb := range feedbacks {
		err := tracker.UpdateReliability(ctx, "semgrep", fb)
		require.NoError(t, err)
	}

	// Should have some reliability score
	reliability = tracker.GetReliability(ctx, "semgrep")
	assert.Greater(t, reliability, 0.0)
	assert.Less(t, reliability, 1.0)
}

func TestReliabilityTracker_CalculateFalsePositiveRate(t *testing.T) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Add mixed feedback
	feedbacks := []UserFeedback{
		{
			FindingID:  "finding-1",
			Action:     "confirmed",
			Confidence: 0.9,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-2",
			Action:     "false_positive",
			Confidence: 0.8,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-3",
			Action:     "confirmed",
			Confidence: 0.7,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-4",
			Action:     "false_positive",
			Confidence: 0.6,
			Timestamp:  time.Now(),
		},
	}

	for _, fb := range feedbacks {
		err := tracker.UpdateReliability(ctx, "semgrep", fb)
		require.NoError(t, err)
	}

	fpRate := tracker.CalculateFalsePositiveRate(ctx, "semgrep")
	assert.Equal(t, 0.5, fpRate) // 2 false positives out of 4 total
}

func TestReliabilityTracker_GetToolStats(t *testing.T) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Test unknown tool
	stats := tracker.GetToolStats(ctx, "unknown-tool")
	assert.Equal(t, "unknown-tool", stats.Tool)
	assert.Equal(t, 0.5, stats.ReliabilityScore)

	// Add feedback with different actions
	feedbacks := []UserFeedback{
		{
			FindingID:  "finding-1",
			Action:     "confirmed",
			Confidence: 0.9,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-2",
			Action:     "false_positive",
			Confidence: 0.8,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-3",
			Action:     "fixed",
			Confidence: 0.7,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-4",
			Action:     "ignored",
			Confidence: 0.6,
			Timestamp:  time.Now(),
		},
	}

	for _, fb := range feedbacks {
		err := tracker.UpdateReliability(ctx, "semgrep", fb)
		require.NoError(t, err)
	}

	stats = tracker.GetToolStats(ctx, "semgrep")
	assert.Equal(t, "semgrep", stats.Tool)
	assert.Equal(t, 4, stats.TotalFindings)
	assert.Equal(t, 1, stats.ConfirmedFindings)
	assert.Equal(t, 1, stats.FalsePositives)
	assert.Equal(t, 1, stats.FixedFindings)
	assert.Equal(t, 1, stats.IgnoredFindings)
	assert.Greater(t, stats.ReliabilityScore, 0.0)
	assert.Equal(t, 0.25, stats.FalsePositiveRate) // 1 out of 4
}

func TestReliabilityTracker_GetAllReliabilities(t *testing.T) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Add feedback for multiple tools
	tools := []string{"semgrep", "eslint", "bandit"}
	for _, tool := range tools {
		feedback := UserFeedback{
			FindingID:  "finding-" + tool,
			Action:     "confirmed",
			Confidence: 0.8,
			Timestamp:  time.Now(),
		}
		err := tracker.UpdateReliability(ctx, tool, feedback)
		require.NoError(t, err)
	}

	reliabilities := tracker.GetAllReliabilities(ctx)
	assert.Len(t, reliabilities, len(tools))

	for _, tool := range tools {
		reliability, exists := reliabilities[tool]
		assert.True(t, exists)
		assert.GreaterOrEqual(t, reliability, 0.0)
		assert.LessOrEqual(t, reliability, 1.0)
	}
}

func TestReliabilityTracker_TimeDecay(t *testing.T) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Add old feedback (should have less weight)
	oldFeedback := UserFeedback{
		FindingID:  "finding-old",
		Action:     "false_positive",
		Confidence: 0.9,
		Timestamp:  time.Now().AddDate(0, 0, -30), // 30 days ago
	}

	// Add recent feedback (should have more weight)
	recentFeedback := UserFeedback{
		FindingID:  "finding-recent",
		Action:     "confirmed",
		Confidence: 0.9,
		Timestamp:  time.Now(),
	}

	err := tracker.UpdateReliability(ctx, "semgrep", oldFeedback)
	require.NoError(t, err)

	err = tracker.UpdateReliability(ctx, "semgrep", recentFeedback)
	require.NoError(t, err)

	// Recent positive feedback should outweigh old negative feedback
	reliability := tracker.GetReliability(ctx, "semgrep")
	assert.GreaterOrEqual(t, reliability, 0.5) // Should be at least neutral due to time decay
}

func TestReliabilityTracker_AddConfidenceRecord(t *testing.T) {
	tracker := NewReliabilityTracker()

	// Add confidence records
	tracker.AddConfidenceRecord("semgrep", 0.9, true, agent.SeverityHigh, agent.CategoryXSS)
	tracker.AddConfidenceRecord("semgrep", 0.3, false, agent.SeverityLow, agent.CategoryOther)
	tracker.AddConfidenceRecord("semgrep", 0.8, true, agent.SeverityMedium, agent.CategorySQLInjection)

	// Get tool stats to verify confidence accuracy calculation
	stats := tracker.GetToolStats(context.Background(), "semgrep")
	assert.Greater(t, stats.ConfidenceAccuracy, 0.0)
	assert.LessOrEqual(t, stats.ConfidenceAccuracy, 1.0)
}

func TestReliabilityTracker_GetReliabilityTrend(t *testing.T) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Add feedback over time
	for i := 0; i < 5; i++ {
		feedback := UserFeedback{
			FindingID:  "finding-" + string(rune(i)),
			Action:     "confirmed",
			Confidence: 0.8,
			Timestamp:  time.Now().AddDate(0, 0, -i), // i days ago
		}
		err := tracker.UpdateReliability(ctx, "semgrep", feedback)
		require.NoError(t, err)
	}

	// Get reliability trend for last 7 days
	trend := tracker.GetReliabilityTrend(ctx, "semgrep", 7)
	assert.Len(t, trend, 7)

	// Verify trend points
	for _, point := range trend {
		assert.GreaterOrEqual(t, point.Reliability, 0.0)
		assert.LessOrEqual(t, point.Reliability, 1.0)
		assert.NotZero(t, point.Date)
	}
}

func TestReliabilityTracker_SeveritySpecificStats(t *testing.T) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Add feedback with different severities
	feedbacks := []struct {
		action   string
		severity agent.Severity
	}{
		{"confirmed", agent.SeverityHigh},
		{"confirmed", agent.SeverityHigh},
		{"false_positive", agent.SeverityHigh},
		{"confirmed", agent.SeverityMedium},
		{"false_positive", agent.SeverityMedium},
		{"false_positive", agent.SeverityMedium},
		{"confirmed", agent.SeverityLow},
	}

	for i, fb := range feedbacks {
		feedback := UserFeedback{
			FindingID:  "finding-" + string(rune(i)),
			Action:     fb.action,
			Confidence: 0.8,
			Timestamp:  time.Now(),
		}

		// Create timestamped feedback with severity
		tracker.mu.Lock()
		if _, exists := tracker.toolStats["semgrep"]; !exists {
			tracker.toolStats["semgrep"] = &ToolStatsInternal{
				Tool:              "semgrep",
				FeedbackHistory:   make([]TimestampedFeedback, 0),
				ConfidenceHistory: make([]ConfidenceRecord, 0),
			}
		}

		timestampedFb := TimestampedFeedback{
			Feedback:  feedback,
			Timestamp: feedback.Timestamp,
			Severity:  fb.severity,
			Confidence: feedback.Confidence,
		}

		tracker.toolStats["semgrep"].FeedbackHistory = append(
			tracker.toolStats["semgrep"].FeedbackHistory,
			timestampedFb,
		)
		tracker.mu.Unlock()
	}

	stats := tracker.GetToolStats(ctx, "semgrep")

	// Check high severity stats (2 confirmed, 1 false positive)
	assert.Equal(t, 3, stats.HighSeverityStats.Total)
	assert.Equal(t, 2, stats.HighSeverityStats.Confirmed)
	assert.Equal(t, 1, stats.HighSeverityStats.FalsePositives)
	assert.InDelta(t, 0.67, stats.HighSeverityStats.Accuracy, 0.01)

	// Check medium severity stats (1 confirmed, 2 false positives)
	assert.Equal(t, 3, stats.MediumSeverityStats.Total)
	assert.Equal(t, 1, stats.MediumSeverityStats.Confirmed)
	assert.Equal(t, 2, stats.MediumSeverityStats.FalsePositives)
	assert.InDelta(t, 0.33, stats.MediumSeverityStats.Accuracy, 0.01)

	// Check low severity stats (1 confirmed, 0 false positives)
	assert.Equal(t, 1, stats.LowSeverityStats.Total)
	assert.Equal(t, 1, stats.LowSeverityStats.Confirmed)
	assert.Equal(t, 0, stats.LowSeverityStats.FalsePositives)
	assert.Equal(t, 1.0, stats.LowSeverityStats.Accuracy)
}

func TestReliabilityTracker_DataTrimming(t *testing.T) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Add more than 1000 feedback entries to test trimming
	for i := 0; i < 1100; i++ {
		feedback := UserFeedback{
			FindingID:  "finding-" + string(rune(i%100)), // Cycle through finding IDs
			Action:     "confirmed",
			Confidence: 0.8,
			Timestamp:  time.Now().Add(time.Duration(-i) * time.Minute),
		}
		err := tracker.UpdateReliability(ctx, "semgrep", feedback)
		require.NoError(t, err)
	}

	// Check that feedback history is trimmed to 1000 entries
	tracker.mu.RLock()
	stats := tracker.toolStats["semgrep"]
	assert.LessOrEqual(t, len(stats.FeedbackHistory), 1000)
	tracker.mu.RUnlock()
}

// Benchmark tests for reliability tracker performance
func BenchmarkReliabilityTracker_UpdateReliability(b *testing.B) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	feedback := UserFeedback{
		FindingID:  "finding-1",
		Action:     "confirmed",
		Confidence: 0.8,
		Timestamp:  time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := tracker.UpdateReliability(ctx, "semgrep", feedback)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReliabilityTracker_GetReliability(b *testing.B) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Add some initial data
	for i := 0; i < 100; i++ {
		feedback := UserFeedback{
			FindingID:  "finding-" + string(rune(i)),
			Action:     "confirmed",
			Confidence: 0.8,
			Timestamp:  time.Now(),
		}
		tracker.UpdateReliability(ctx, "semgrep", feedback)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.GetReliability(ctx, "semgrep")
	}
}

func BenchmarkReliabilityTracker_GetToolStats(b *testing.B) {
	tracker := NewReliabilityTracker()
	ctx := context.Background()

	// Add some initial data
	for i := 0; i < 100; i++ {
		feedback := UserFeedback{
			FindingID:  "finding-" + string(rune(i)),
			Action:     "confirmed",
			Confidence: 0.8,
			Timestamp:  time.Now(),
		}
		tracker.UpdateReliability(ctx, "semgrep", feedback)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.GetToolStats(ctx, "semgrep")
	}
}