package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/pkg/agent"
)

func TestPlattCalibrator_CalibrateConfidence(t *testing.T) {
	calibrator := NewPlattCalibrator()
	ctx := context.Background()

	context := CalibrationContext{
		Tool:     "semgrep",
		Severity: agent.SeverityHigh,
		Category: agent.CategoryXSS,
	}

	// Test with no training data (should use simple calibration)
	calibrated, err := calibrator.CalibrateConfidence(ctx, 0.8, context)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, calibrated, 0.01)
	assert.LessOrEqual(t, calibrated, 0.99)
}

func TestPlattCalibrator_UpdateCalibration(t *testing.T) {
	calibrator := NewPlattCalibrator()
	ctx := context.Background()

	prediction := ConfidencePrediction{
		ID:                   "pred-1",
		RawConfidence:        0.8,
		CalibratedConfidence: 0.75,
		Context: CalibrationContext{
			Tool:     "semgrep",
			Severity: agent.SeverityHigh,
			Category: agent.CategoryXSS,
		},
		Timestamp: time.Now(),
		FindingID: "finding-1",
	}

	// Update with positive outcome
	err := calibrator.UpdateCalibration(ctx, prediction, true)
	require.NoError(t, err)

	// Update with negative outcome
	prediction.ID = "pred-2"
	prediction.FindingID = "finding-2"
	err = calibrator.UpdateCalibration(ctx, prediction, false)
	require.NoError(t, err)

	// Verify data was stored
	contextKey := calibrator.generateContextKey(prediction.Context)
	calibrator.mu.RLock()
	data, exists := calibrator.calibrationData[contextKey]
	calibrator.mu.RUnlock()

	assert.True(t, exists)
	assert.Len(t, data, 2)
}

func TestPlattCalibrator_GetCalibrationCurve(t *testing.T) {
	calibrator := NewPlattCalibrator()
	ctx := context.Background()

	context := CalibrationContext{
		Tool:     "semgrep",
		Severity: agent.SeverityHigh,
		Category: agent.CategoryXSS,
	}

	// Add calibration data across different confidence levels
	predictions := []struct {
		confidence float64
		outcome    bool
	}{
		{0.1, false}, {0.1, false}, {0.1, true},   // Low confidence
		{0.5, false}, {0.5, true}, {0.5, true},   // Medium confidence
		{0.9, true}, {0.9, true}, {0.9, false},   // High confidence
	}

	for i, pred := range predictions {
		prediction := ConfidencePrediction{
			ID:                   "pred-" + string(rune(i)),
			RawConfidence:        pred.confidence,
			CalibratedConfidence: pred.confidence,
			Context:              context,
			Timestamp:            time.Now(),
			FindingID:            "finding-" + string(rune(i)),
		}

		err := calibrator.UpdateCalibration(ctx, prediction, pred.outcome)
		require.NoError(t, err)
	}

	// Get calibration curve
	curve, err := calibrator.GetCalibrationCurve(ctx, context)
	require.NoError(t, err)

	assert.Len(t, curve.Bins, calibrator.binCount)
	assert.Greater(t, len(curve.ReliabilityDiagram), 0)
	assert.GreaterOrEqual(t, curve.BrierScore, 0.0)
	assert.GreaterOrEqual(t, curve.CalibrationError, 0.0)

	// Check that bins with data have reasonable values
	for _, bin := range curve.Bins {
		if bin.Count > 0 {
			assert.GreaterOrEqual(t, bin.MeanConfidence, 0.0)
			assert.LessOrEqual(t, bin.MeanConfidence, 1.0)
			assert.GreaterOrEqual(t, bin.ActualAccuracy, 0.0)
			assert.LessOrEqual(t, bin.ActualAccuracy, 1.0)
			assert.GreaterOrEqual(t, bin.CalibrationError, 0.0)
		}
	}
}

func TestPlattCalibrator_GetCalibrationStats(t *testing.T) {
	calibrator := NewPlattCalibrator()
	ctx := context.Background()

	// Test with no data
	stats := calibrator.GetCalibrationStats(ctx)
	assert.Equal(t, 0, stats.TotalPredictions)
	assert.Equal(t, 0.0, stats.OverallAccuracy)

	// Add some calibration data
	context := CalibrationContext{
		Tool:     "semgrep",
		Severity: agent.SeverityHigh,
	}

	predictions := []struct {
		confidence float64
		outcome    bool
	}{
		{0.8, true},   // Correct high confidence
		{0.3, false},  // Correct low confidence
		{0.9, false},  // Overconfident
		{0.2, true},   // Underconfident
	}

	for i, pred := range predictions {
		prediction := ConfidencePrediction{
			ID:                   "pred-" + string(rune(i)),
			RawConfidence:        pred.confidence,
			CalibratedConfidence: pred.confidence,
			Context:              context,
			Timestamp:            time.Now(),
			FindingID:            "finding-" + string(rune(i)),
		}

		err := calibrator.UpdateCalibration(ctx, prediction, pred.outcome)
		require.NoError(t, err)
	}

	stats = calibrator.GetCalibrationStats(ctx)
	assert.Equal(t, 4, stats.TotalPredictions)
	assert.Equal(t, 0.5, stats.OverallAccuracy) // 2 correct out of 4
	assert.Greater(t, stats.BrierScore, 0.0)
	assert.Greater(t, stats.CalibrationError, 0.0)
	assert.Greater(t, stats.Overconfidence, 0.0)   // From the 0.9 -> false case
	assert.Greater(t, stats.Underconfidence, 0.0)  // From the 0.2 -> true case
}

func TestPlattCalibrator_GenerateContextKey(t *testing.T) {
	calibrator := NewPlattCalibrator()

	tests := []struct {
		name     string
		context  CalibrationContext
		expected string
	}{
		{
			name: "full context",
			context: CalibrationContext{
				Tool:     "semgrep",
				Severity: agent.SeverityHigh,
				Category: agent.CategoryXSS,
				RuleID:   "test-rule",
			},
			expected: "semgrep_high_xss_test-rule",
		},
		{
			name: "partial context",
			context: CalibrationContext{
				Tool:     "eslint",
				Severity: agent.SeverityMedium,
			},
			expected: "eslint_medium",
		},
		{
			name: "tool only",
			context: CalibrationContext{
				Tool: "bandit",
			},
			expected: "bandit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := calibrator.generateContextKey(tt.context)
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestPlattCalibrator_SimpleCalibration(t *testing.T) {
	calibrator := NewPlattCalibrator()

	// Test with no data
	calibrated := calibrator.simpleCalibration(0.8, CalibrationContext{Tool: "unknown"})
	assert.GreaterOrEqual(t, calibrated, 0.0)
	assert.LessOrEqual(t, calibrated, 1.0)
	assert.Less(t, calibrated, 0.8) // Should be reduced due to uncertainty

	// Add some data for a tool
	contextKey := "semgrep"
	calibrator.calibrationData[contextKey] = []CalibrationRecord{
		{
			RawConfidence:   0.7,
			ActualOutcome:   true,
			Timestamp:       time.Now(),
		},
		{
			RawConfidence:   0.8,
			ActualOutcome:   false,
			Timestamp:       time.Now(),
		},
		{
			RawConfidence:   0.9,
			ActualOutcome:   true,
			Timestamp:       time.Now(),
		},
	}

	// Test with available data
	calibrated = calibrator.simpleCalibration(0.8, CalibrationContext{Tool: "semgrep"})
	assert.GreaterOrEqual(t, calibrated, 0.01)
	assert.LessOrEqual(t, calibrated, 0.99)
}

func TestPlattCalibrator_IsotonicCalibration(t *testing.T) {
	calibrator := NewPlattCalibrator()

	// Test with empty data
	calibrated := calibrator.isotonicCalibration(0.8, []CalibrationRecord{})
	assert.Equal(t, 0.8, calibrated)

	// Test with single data point
	data := []CalibrationRecord{
		{
			RawConfidence: 0.7,
			ActualOutcome: true,
		},
	}
	calibrated = calibrator.isotonicCalibration(0.8, data)
	assert.Equal(t, 1.0, calibrated) // Should return the actual outcome

	// Test with multiple data points
	data = []CalibrationRecord{
		{
			RawConfidence: 0.3,
			ActualOutcome: false,
		},
		{
			RawConfidence: 0.7,
			ActualOutcome: true,
		},
		{
			RawConfidence: 0.9,
			ActualOutcome: true,
		},
	}

	// Test interpolation between points
	calibrated = calibrator.isotonicCalibration(0.8, data)
	assert.GreaterOrEqual(t, calibrated, 0.01)
	assert.LessOrEqual(t, calibrated, 0.99)
	assert.Greater(t, calibrated, 0.5) // Should be closer to true given the pattern
}

func TestPlattCalibrator_FitSigmoid(t *testing.T) {
	calibrator := NewPlattCalibrator()

	// Test with simple linear relationship
	confidences := []float64{0.1, 0.3, 0.5, 0.7, 0.9}
	outcomes := []float64{0.0, 0.0, 0.5, 1.0, 1.0}

	a, b := calibrator.fitSigmoid(confidences, outcomes)

	// Should have reasonable parameters
	assert.NotEqual(t, 0.0, a)
	assert.NotEqual(t, 0.0, b)

	// Test with empty data
	a, b = calibrator.fitSigmoid([]float64{}, []float64{})
	assert.Equal(t, 1.0, a)
	assert.Equal(t, 0.0, b)
}

func TestPlattCalibrator_CleanOldData(t *testing.T) {
	calibrator := NewPlattCalibrator()
	calibrator.maxAge = 24 * time.Hour // 1 day

	contextKey := "semgrep_high"

	// Add old and new data
	calibrator.calibrationData[contextKey] = []CalibrationRecord{
		{
			RawConfidence: 0.8,
			ActualOutcome: true,
			Timestamp:     time.Now().Add(-48 * time.Hour), // 2 days old
		},
		{
			RawConfidence: 0.7,
			ActualOutcome: false,
			Timestamp:     time.Now().Add(-12 * time.Hour), // 12 hours old
		},
		{
			RawConfidence: 0.9,
			ActualOutcome: true,
			Timestamp:     time.Now(), // Current
		},
	}

	// Clean old data
	calibrator.cleanOldData(contextKey)

	// Should only have recent data
	data := calibrator.calibrationData[contextKey]
	assert.Len(t, data, 2) // Old data should be removed

	for _, record := range data {
		assert.True(t, record.Timestamp.After(time.Now().Add(-25*time.Hour)))
	}
}

func TestPlattCalibrator_RecomputeCalibration(t *testing.T) {
	calibrator := NewPlattCalibrator()
	calibrator.minSampleSize = 3

	contextKey := "semgrep_high"

	// Add sufficient data for recomputation
	calibrator.calibrationData[contextKey] = []CalibrationRecord{
		{RawConfidence: 0.2, ActualOutcome: false},
		{RawConfidence: 0.5, ActualOutcome: false},
		{RawConfidence: 0.8, ActualOutcome: true},
		{RawConfidence: 0.9, ActualOutcome: true},
	}

	err := calibrator.recomputeCalibration(contextKey)
	require.NoError(t, err)

	// Check that parameters were computed
	params, exists := calibrator.sigmoidParams[contextKey]
	assert.True(t, exists)
	assert.NotEqual(t, 0.0, params.A)
	assert.Equal(t, 4, params.SampleCount)
	assert.NotZero(t, params.LastUpdated)
}

func TestPlattCalibrator_GetCalibrationCurveForTool(t *testing.T) {
	calibrator := NewPlattCalibrator()
	ctx := context.Background()

	// Add some data for a tool
	context := CalibrationContext{Tool: "semgrep"}
	prediction := ConfidencePrediction{
		ID:                   "pred-1",
		RawConfidence:        0.8,
		CalibratedConfidence: 0.75,
		Context:              context,
		Timestamp:            time.Now(),
		FindingID:            "finding-1",
	}

	err := calibrator.UpdateCalibration(ctx, prediction, true)
	require.NoError(t, err)

	// Get calibration curve for the tool
	curve, err := calibrator.GetCalibrationCurveForTool(ctx, "semgrep")
	require.NoError(t, err)

	assert.Len(t, curve.Bins, calibrator.binCount)
}

func TestPlattCalibrator_GetReliabilityDiagram(t *testing.T) {
	calibrator := NewPlattCalibrator()
	ctx := context.Background()

	context := CalibrationContext{Tool: "semgrep"}

	// Add some calibration data
	predictions := []struct {
		confidence float64
		outcome    bool
	}{
		{0.2, false},
		{0.5, true},
		{0.8, true},
	}

	for i, pred := range predictions {
		prediction := ConfidencePrediction{
			ID:                   "pred-" + string(rune(i)),
			RawConfidence:        pred.confidence,
			CalibratedConfidence: pred.confidence,
			Context:              context,
			Timestamp:            time.Now(),
			FindingID:            "finding-" + string(rune(i)),
		}

		err := calibrator.UpdateCalibration(ctx, prediction, pred.outcome)
		require.NoError(t, err)
	}

	// Get reliability diagram
	points, err := calibrator.GetReliabilityDiagram(ctx, context)
	require.NoError(t, err)

	assert.Greater(t, len(points), 0)
	for _, point := range points {
		assert.GreaterOrEqual(t, point.X, 0.0)
		assert.LessOrEqual(t, point.X, 1.0)
		assert.GreaterOrEqual(t, point.Y, 0.0)
		assert.LessOrEqual(t, point.Y, 1.0)
	}
}

// Benchmark tests for calibration performance
func BenchmarkPlattCalibrator_CalibrateConfidence(b *testing.B) {
	calibrator := NewPlattCalibrator()
	ctx := context.Background()

	context := CalibrationContext{
		Tool:     "semgrep",
		Severity: agent.SeverityHigh,
		Category: agent.CategoryXSS,
	}

	// Add some training data
	for i := 0; i < 50; i++ {
		prediction := ConfidencePrediction{
			ID:                   "pred-" + string(rune(i)),
			RawConfidence:        0.8,
			CalibratedConfidence: 0.75,
			Context:              context,
			Timestamp:            time.Now(),
			FindingID:            "finding-" + string(rune(i)),
		}
		calibrator.UpdateCalibration(ctx, prediction, i%2 == 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := calibrator.CalibrateConfidence(ctx, 0.8, context)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPlattCalibrator_UpdateCalibration(b *testing.B) {
	calibrator := NewPlattCalibrator()
	ctx := context.Background()

	context := CalibrationContext{
		Tool:     "semgrep",
		Severity: agent.SeverityHigh,
		Category: agent.CategoryXSS,
	}

	prediction := ConfidencePrediction{
		ID:                   "pred-1",
		RawConfidence:        0.8,
		CalibratedConfidence: 0.75,
		Context:              context,
		Timestamp:            time.Now(),
		FindingID:            "finding-1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prediction.ID = "pred-" + string(rune(i))
		prediction.FindingID = "finding-" + string(rune(i))
		err := calibrator.UpdateCalibration(ctx, prediction, i%2 == 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}