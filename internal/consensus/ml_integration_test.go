package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/pkg/agent"
)

func TestEngine_MLIntegration(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)

	// Verify ML components are initialized
	assert.NotNil(t, engine.mlModel)
	assert.NotNil(t, engine.reliabilityTracker)
	assert.NotNil(t, engine.calibrator)
	assert.NotNil(t, engine.featureExtractor)
	assert.True(t, engine.enableML)
}

func TestEngine_UpdateModelWithFeedback(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)
	ctx := context.Background()

	// Create user feedback
	feedback := []UserFeedback{
		{
			FindingID:  "finding-1",
			UserID:     "user-1",
			Action:     "confirmed",
			Comment:    "This is a real vulnerability",
			Confidence: 0.9,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-2",
			UserID:     "user-1",
			Action:     "false_positive",
			Comment:    "This is not a real issue",
			Confidence: 0.8,
			Timestamp:  time.Now(),
		},
	}

	// Update model with feedback
	err := engine.UpdateModel(ctx, feedback)
	require.NoError(t, err)

	// Verify model info is updated
	modelInfo := engine.GetMLModelInfo()
	assert.Equal(t, "linear_regression", modelInfo.ModelType)
	assert.Greater(t, modelInfo.TrainingSize, 0)
}

func TestEngine_GetToolReliabilities(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)
	ctx := context.Background()

	// Initially should be empty
	reliabilities := engine.GetToolReliabilities(ctx)
	assert.Empty(t, reliabilities)

	// Add some feedback
	feedback := UserFeedback{
		FindingID:  "finding-1",
		UserID:     "user-1",
		Action:     "confirmed",
		Confidence: 0.9,
		Timestamp:  time.Now(),
	}

	err := engine.reliabilityTracker.UpdateReliability(ctx, "semgrep", feedback)
	require.NoError(t, err)

	// Should now have reliability data
	reliabilities = engine.GetToolReliabilities(ctx)
	assert.Contains(t, reliabilities, "semgrep")
	assert.GreaterOrEqual(t, reliabilities["semgrep"], 0.0)
	assert.LessOrEqual(t, reliabilities["semgrep"], 1.0)
}

func TestEngine_GetToolStats(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)
	ctx := context.Background()

	// Test unknown tool
	stats := engine.GetToolStats(ctx, "unknown-tool")
	assert.Equal(t, "unknown-tool", stats.Tool)
	assert.Equal(t, 0.5, stats.ReliabilityScore)

	// Add feedback for a tool
	feedback := UserFeedback{
		FindingID:  "finding-1",
		UserID:     "user-1",
		Action:     "confirmed",
		Confidence: 0.9,
		Timestamp:  time.Now(),
	}

	err := engine.reliabilityTracker.UpdateReliability(ctx, "semgrep", feedback)
	require.NoError(t, err)

	// Get updated stats
	stats = engine.GetToolStats(ctx, "semgrep")
	assert.Equal(t, "semgrep", stats.Tool)
	assert.Greater(t, stats.TotalFindings, 0)
	assert.Greater(t, stats.ReliabilityScore, 0.0)
}

func TestEngine_GetCalibrationStats(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)
	ctx := context.Background()

	// Initially should have no data
	stats := engine.GetCalibrationStats(ctx)
	assert.Equal(t, 0, stats.TotalPredictions)

	// Add some calibration data
	prediction := ConfidencePrediction{
		ID:                   "pred-1",
		RawConfidence:        0.8,
		CalibratedConfidence: 0.75,
		Context: CalibrationContext{
			Tool:     "semgrep",
			Severity: agent.SeverityHigh,
		},
		Timestamp: time.Now(),
		FindingID: "finding-1",
	}

	err := engine.calibrator.UpdateCalibration(ctx, prediction, true)
	require.NoError(t, err)

	// Should now have calibration data
	stats = engine.GetCalibrationStats(ctx)
	assert.Greater(t, stats.TotalPredictions, 0)
}

func TestEngine_TrainModelWithHistoricalData(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)
	ctx := context.Background()

	// Create historical data
	historicalData := []HistoricalFinding{
		{
			Finding: agent.Finding{
				ID:         "hist-1",
				Tool:       "semgrep",
				RuleID:     "test-rule",
				Severity:   agent.SeverityHigh,
				Category:   agent.CategoryXSS,
				Confidence: 0.9,
			},
			WasFalsePositive: false,
			UserAction:       "confirmed",
			Timestamp:        time.Now().AddDate(0, 0, -30),
		},
		{
			Finding: agent.Finding{
				ID:         "hist-2",
				Tool:       "eslint",
				RuleID:     "test-rule-2",
				Severity:   agent.SeverityMedium,
				Category:   agent.CategoryXSS,
				Confidence: 0.3,
			},
			WasFalsePositive: true,
			UserAction:       "false_positive",
			Timestamp:        time.Now().AddDate(0, 0, -20),
		},
	}

	// Train model with historical data
	err := engine.TrainModelWithHistoricalData(ctx, historicalData)
	require.NoError(t, err)

	// Verify model was trained
	modelInfo := engine.GetMLModelInfo()
	assert.Greater(t, modelInfo.TrainingSize, 0)
	assert.NotZero(t, modelInfo.LastTrained)
}

func TestEngine_MLEnhancedConsensusScoring(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)
	ctx := context.Background()

	// Create findings with confidence values
	findings := []agent.Finding{
		{
			ID:         "finding-1",
			Tool:       "semgrep",
			RuleID:     "test-rule",
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
			Title:      "XSS vulnerability",
			File:       "app.js",
			Line:       42,
			Confidence: 0.9,
		},
		{
			ID:         "finding-2",
			Tool:       "eslint",
			RuleID:     "test-rule-2",
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
			Title:      "XSS vulnerability",
			File:       "app.js",
			Line:       42,
			Confidence: 0.8,
		},
		{
			ID:         "finding-3",
			Tool:       "bandit",
			RuleID:     "test-rule-3",
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
			Title:      "XSS vulnerability",
			File:       "app.js",
			Line:       42,
			Confidence: 0.85,
		},
	}

	// Train the model with some data first
	trainingData := []TrainingExample{
		{
			Features: MLFeatures{
				ToolCount:           3,
				AverageConfidence:   0.85,
				WeightedReliability: 0.9,
				SeverityConsensus:   1.0,
				CategoryConsensus:   1.0,
			},
			TrueLabel:  1.0,
			UserAction: "confirmed",
			Confidence: 0.95,
			Timestamp:  time.Now(),
		},
	}

	err := engine.mlModel.Train(ctx, trainingData)
	require.NoError(t, err)

	// Analyze findings with ML enhancement
	result, err := engine.AnalyzeFindings(ctx, findings)
	require.NoError(t, err)

	assert.Len(t, result.DeduplicatedFindings, 1) // Should be deduplicated
	consensusFinding := result.DeduplicatedFindings[0]

	// Should have high confidence due to multiple tool agreement and ML enhancement
	assert.GreaterOrEqual(t, consensusFinding.ConsensusScore, 0.90)
	assert.Equal(t, 3, consensusFinding.AgreementCount)
	assert.Len(t, consensusFinding.SupportingTools, 3)
}

func TestEngine_CalculateMLEnhancedScore(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)

	findings := []agent.Finding{
		{
			ID:         "finding-1",
			Tool:       "semgrep",
			Confidence: 0.9,
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
		},
		{
			ID:         "finding-2",
			Tool:       "eslint",
			Confidence: 0.8,
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
		},
	}

	tools := []string{"semgrep", "eslint"}
	toolReliabilities := map[string]float64{
		"semgrep": 0.9,
		"eslint":  0.8,
	}

	// Test ML enhanced scoring
	score, err := engine.calculateMLEnhancedScore(context.Background(), findings, tools, toolReliabilities)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

func TestEngine_FeedbackToLabel(t *testing.T) {
	engine := NewEngine(DefaultConfig())

	tests := []struct {
		action   string
		expected float64
	}{
		{"confirmed", 1.0},
		{"fixed", 1.0},
		{"false_positive", 0.0},
		{"ignored", 0.3},
		{"unknown", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			label := engine.feedbackToLabel(tt.action)
			assert.Equal(t, tt.expected, label)
		})
	}
}

func TestEngine_MLDisabled(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = false
	engine := NewEngine(config)
	ctx := context.Background()

	// ML components should still be initialized but not used
	assert.NotNil(t, engine.mlModel)
	assert.NotNil(t, engine.reliabilityTracker)
	assert.NotNil(t, engine.calibrator)
	assert.False(t, engine.enableML)

	// Create findings
	findings := []agent.Finding{
		{
			ID:         "finding-1",
			Tool:       "semgrep",
			Confidence: 0.9,
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
		},
	}

	// Analyze findings - should work without ML enhancement
	result, err := engine.AnalyzeFindings(ctx, findings)
	require.NoError(t, err)
	assert.Len(t, result.DeduplicatedFindings, 1)

	// Update model should be no-op when ML is disabled
	feedback := []UserFeedback{
		{
			FindingID:  "finding-1",
			Action:     "confirmed",
			Confidence: 0.9,
			Timestamp:  time.Now(),
		},
	}

	err = engine.UpdateModel(ctx, feedback)
	require.NoError(t, err) // Should not error, but should be no-op
}

func TestEngine_MLIntegrationWithCalibration(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)
	ctx := context.Background()

	// Add calibration data
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

	err := engine.calibrator.UpdateCalibration(ctx, prediction, true)
	require.NoError(t, err)

	// Create findings
	findings := []agent.Finding{
		{
			ID:         "finding-1",
			Tool:       "semgrep",
			RuleID:     "test-rule",
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
			Confidence: 0.8,
		},
	}

	// Analyze findings - should apply calibration
	result, err := engine.AnalyzeFindings(ctx, findings)
	require.NoError(t, err)
	assert.Len(t, result.DeduplicatedFindings, 1)

	// The consensus score should be influenced by calibration
	consensusFinding := result.DeduplicatedFindings[0]
	assert.GreaterOrEqual(t, consensusFinding.ConsensusScore, 0.0)
	assert.LessOrEqual(t, consensusFinding.ConsensusScore, 1.0)
}

func TestEngine_MLIntegrationWithReliabilityTracking(t *testing.T) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)
	ctx := context.Background()

	// Add reliability feedback
	feedback := UserFeedback{
		FindingID:  "finding-1",
		UserID:     "user-1",
		Action:     "confirmed",
		Confidence: 0.9,
		Timestamp:  time.Now(),
	}

	err := engine.reliabilityTracker.UpdateReliability(ctx, "semgrep", feedback)
	require.NoError(t, err)

	// Create findings
	findings := []agent.Finding{
		{
			ID:         "finding-1",
			Tool:       "semgrep",
			Confidence: 0.8,
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
		},
		{
			ID:         "finding-2",
			Tool:       "eslint",
			Confidence: 0.7,
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
		},
	}

	// Analyze findings - should use dynamic reliability scores
	result, err := engine.AnalyzeFindings(ctx, findings)
	require.NoError(t, err)
	assert.Len(t, result.DeduplicatedFindings, 1)

	// The consensus score should be influenced by tool reliability
	consensusFinding := result.DeduplicatedFindings[0]
	assert.GreaterOrEqual(t, consensusFinding.ConsensusScore, 0.0)
	assert.LessOrEqual(t, consensusFinding.ConsensusScore, 1.0)
}

// Integration benchmark tests
func BenchmarkEngine_MLEnhancedAnalysis(b *testing.B) {
	config := DefaultConfig()
	config.EnableLearning = true
	engine := NewEngine(config)
	ctx := context.Background()

	// Train the model with some data
	trainingData := []TrainingExample{
		{
			Features: MLFeatures{
				ToolCount:           2,
				AverageConfidence:   0.8,
				WeightedReliability: 0.85,
			},
			TrueLabel: 1.0,
		},
	}
	engine.mlModel.Train(ctx, trainingData)

	findings := []agent.Finding{
		{
			ID:         "finding-1",
			Tool:       "semgrep",
			Confidence: 0.9,
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
		},
		{
			ID:         "finding-2",
			Tool:       "eslint",
			Confidence: 0.8,
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.AnalyzeFindings(ctx, findings)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEngine_TraditionalVsMLAnalysis(b *testing.B) {
	findings := []agent.Finding{
		{
			ID:         "finding-1",
			Tool:       "semgrep",
			Confidence: 0.9,
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
		},
		{
			ID:         "finding-2",
			Tool:       "eslint",
			Confidence: 0.8,
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
		},
	}

	b.Run("Traditional", func(b *testing.B) {
		config := DefaultConfig()
		config.EnableLearning = false
		engine := NewEngine(config)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.AnalyzeFindings(ctx, findings)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ML Enhanced", func(b *testing.B) {
		config := DefaultConfig()
		config.EnableLearning = true
		engine := NewEngine(config)
		ctx := context.Background()

		// Train the model
		trainingData := []TrainingExample{
			{
				Features: MLFeatures{
					ToolCount:           2,
					AverageConfidence:   0.85,
					WeightedReliability: 0.9,
				},
				TrueLabel: 1.0,
			},
		}
		engine.mlModel.Train(ctx, trainingData)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.AnalyzeFindings(ctx, findings)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}