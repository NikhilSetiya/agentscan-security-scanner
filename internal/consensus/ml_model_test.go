package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

func TestLinearRegressionModel_Predict(t *testing.T) {
	model := NewLinearRegressionModel()
	ctx := context.Background()

	features := MLFeatures{
		ToolCount:           3,
		AverageConfidence:   0.8,
		MaxConfidence:       0.9,
		MinConfidence:       0.7,
		WeightedReliability: 0.85,
		SeverityConsensus:   1.0,
		CategoryConsensus:   1.0,
	}

	// Test prediction with untrained model (should use heuristic)
	score, err := model.Predict(ctx, features)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
	assert.Greater(t, score, 0.5) // Should be high due to good features
}

func TestLinearRegressionModel_Train(t *testing.T) {
	model := NewLinearRegressionModel()
	ctx := context.Background()

	// Create training data
	trainingData := []TrainingExample{
		{
			Features: MLFeatures{
				ToolCount:           3,
				AverageConfidence:   0.9,
				WeightedReliability: 0.9,
				SeverityConsensus:   1.0,
			},
			TrueLabel:  1.0,
			UserAction: "confirmed",
			Confidence: 0.95,
			Timestamp:  time.Now(),
		},
		{
			Features: MLFeatures{
				ToolCount:           1,
				AverageConfidence:   0.3,
				WeightedReliability: 0.4,
				SeverityConsensus:   1.0,
			},
			TrueLabel:  0.0,
			UserAction: "false_positive",
			Confidence: 0.9,
			Timestamp:  time.Now(),
		},
		{
			Features: MLFeatures{
				ToolCount:           2,
				AverageConfidence:   0.7,
				WeightedReliability: 0.8,
				SeverityConsensus:   1.0,
			},
			TrueLabel:  1.0,
			UserAction: "fixed",
			Confidence: 0.85,
			Timestamp:  time.Now(),
		},
	}

	// Train the model
	err := model.Train(ctx, trainingData)
	require.NoError(t, err)

	// Check model info
	info := model.GetModelInfo()
	assert.Equal(t, "linear_regression", info.ModelType)
	assert.Equal(t, len(trainingData), info.TrainingSize)
	assert.Greater(t, info.Accuracy, 0.0)
	assert.NotZero(t, info.LastTrained)
}

func TestLinearRegressionModel_PredictAfterTraining(t *testing.T) {
	model := NewLinearRegressionModel()
	ctx := context.Background()

	// Train with clear patterns
	trainingData := []TrainingExample{
		// High confidence examples (should predict high)
		{
			Features: MLFeatures{
				ToolCount:           3,
				AverageConfidence:   0.9,
				WeightedReliability: 0.9,
			},
			TrueLabel: 1.0,
		},
		{
			Features: MLFeatures{
				ToolCount:           4,
				AverageConfidence:   0.85,
				WeightedReliability: 0.85,
			},
			TrueLabel: 1.0,
		},
		// Low confidence examples (should predict low)
		{
			Features: MLFeatures{
				ToolCount:           1,
				AverageConfidence:   0.3,
				WeightedReliability: 0.4,
			},
			TrueLabel: 0.0,
		},
		{
			Features: MLFeatures{
				ToolCount:           1,
				AverageConfidence:   0.2,
				WeightedReliability: 0.3,
			},
			TrueLabel: 0.0,
		},
	}

	err := model.Train(ctx, trainingData)
	require.NoError(t, err)

	// Test high confidence prediction
	highConfFeatures := MLFeatures{
		ToolCount:           3,
		AverageConfidence:   0.9,
		WeightedReliability: 0.9,
	}
	highScore, err := model.Predict(ctx, highConfFeatures)
	require.NoError(t, err)

	// Test low confidence prediction
	lowConfFeatures := MLFeatures{
		ToolCount:           1,
		AverageConfidence:   0.3,
		WeightedReliability: 0.4,
	}
	lowScore, err := model.Predict(ctx, lowConfFeatures)
	require.NoError(t, err)

	// High confidence features should predict higher than low confidence
	assert.Greater(t, highScore, lowScore)
}

func TestLinearRegressionModel_SaveLoad(t *testing.T) {
	model := NewLinearRegressionModel()
	ctx := context.Background()

	// Train the model first
	trainingData := []TrainingExample{
		{
			Features: MLFeatures{
				ToolCount:         2,
				AverageConfidence: 0.8,
			},
			TrueLabel: 1.0,
		},
	}

	err := model.Train(ctx, trainingData)
	require.NoError(t, err)

	// Test save (simplified - just checks serialization)
	err = model.SaveModel(ctx, "test_model.json")
	require.NoError(t, err)

	// Test load (simplified - just checks it doesn't error)
	err = model.LoadModel(ctx, "test_model.json")
	require.NoError(t, err)
}

func TestFeatureExtractor_ExtractFeatures(t *testing.T) {
	extractor := NewFeatureExtractor()

	// Create test finding group
	group := FindingGroup{
		PrimaryFinding: agent.Finding{
			ID:         "finding-1",
			Tool:       "semgrep",
			RuleID:     "test-rule",
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
			Title:      "Test vulnerability",
			File:       "app.js",
			Line:       42,
			Confidence: 0.9,
		},
		SimilarFindings: []agent.Finding{
			{
				ID:         "finding-2",
				Tool:       "eslint",
				RuleID:     "test-rule-2",
				Severity:   agent.SeverityHigh,
				Category:   agent.CategoryXSS,
				Title:      "Test vulnerability",
				File:       "app.js",
				Line:       43,
				Confidence: 0.8,
			},
		},
		Tools: []string{"semgrep", "eslint"},
	}

	// Create test context
	context := ConsensusContext{
		TotalAgents: 2,
		AgentReliability: map[string]float64{
			"semgrep": 0.9,
			"eslint":  0.8,
		},
		HistoricalData: []HistoricalFinding{
			{
				Finding: agent.Finding{
					RuleID: "test-rule",
					File:   "app.js",
				},
				WasFalsePositive: false,
			},
		},
	}

	toolReliability := map[string]float64{
		"semgrep": 0.9,
		"eslint":  0.8,
	}

	// Extract features
	features := extractor.ExtractFeatures(group, context, toolReliability)

	// Verify basic features
	assert.Equal(t, 2, features.ToolCount)
	assert.InDelta(t, 0.85, features.AverageConfidence, 0.001) // (0.9 + 0.8) / 2
	assert.Equal(t, 0.9, features.MaxConfidence)
	assert.Equal(t, 0.8, features.MinConfidence)
	assert.Greater(t, features.ConfidenceVariance, 0.0)

	// Verify reliability features
	assert.InDelta(t, 0.85, features.WeightedReliability, 0.001) // (0.9 + 0.8) / 2
	assert.Equal(t, 0.9, features.MaxReliability)
	assert.Equal(t, 0.8, features.MinReliability)

	// Verify consensus features
	assert.Equal(t, 1.0, features.SeverityConsensus)  // Both high severity
	assert.Equal(t, 1.0, features.CategoryConsensus)  // Both XSS
	assert.Equal(t, 1.0, features.HighSeverityRatio)  // All high severity

	// Verify historical features
	assert.Greater(t, features.HistoricalAccuracy, 0.0)
	assert.Equal(t, 1, features.SimilarFindingCount)
}

func TestFeatureExtractor_CalculateStatistics(t *testing.T) {
	extractor := NewFeatureExtractor()

	// Test mean calculation
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	mean := extractor.calculateMean(values)
	assert.Equal(t, 3.0, mean)

	// Test max calculation
	max := extractor.calculateMax(values)
	assert.Equal(t, 5.0, max)

	// Test min calculation
	min := extractor.calculateMin(values)
	assert.Equal(t, 1.0, min)

	// Test variance calculation
	variance := extractor.calculateVariance(values)
	assert.Greater(t, variance, 0.0)
	assert.InDelta(t, 2.5, variance, 0.1) // Expected variance for 1,2,3,4,5
}

func TestFeatureExtractor_ConsensusCalculations(t *testing.T) {
	extractor := NewFeatureExtractor()

	// Test severity consensus with agreement
	findings := []agent.Finding{
		{Severity: agent.SeverityHigh},
		{Severity: agent.SeverityHigh},
		{Severity: agent.SeverityHigh},
	}
	consensus := extractor.calculateSeverityConsensus(findings)
	assert.Equal(t, 1.0, consensus)

	// Test severity consensus with disagreement
	findings = []agent.Finding{
		{Severity: agent.SeverityHigh},
		{Severity: agent.SeverityMedium},
		{Severity: agent.SeverityHigh},
	}
	consensus = extractor.calculateSeverityConsensus(findings)
	assert.InDelta(t, 0.67, consensus, 0.01) // 2/3 agree on high

	// Test category consensus
	findings = []agent.Finding{
		{Category: agent.CategoryXSS},
		{Category: agent.CategoryXSS},
	}
	consensus = extractor.calculateCategoryConsensus(findings)
	assert.Equal(t, 1.0, consensus)

	// Test high severity ratio
	findings = []agent.Finding{
		{Severity: agent.SeverityHigh},
		{Severity: agent.SeverityMedium},
		{Severity: agent.SeverityHigh},
	}
	ratio := extractor.calculateHighSeverityRatio(findings)
	assert.InDelta(t, 0.67, ratio, 0.01) // 2/3 are high severity
}

func TestFeatureExtractor_HistoricalAnalysis(t *testing.T) {
	extractor := NewFeatureExtractor()

	finding := agent.Finding{
		RuleID: "test-rule",
		File:   "app.js",
	}

	historical := []HistoricalFinding{
		{
			Finding: agent.Finding{
				RuleID: "test-rule",
				File:   "app.js",
			},
			WasFalsePositive: false,
		},
		{
			Finding: agent.Finding{
				RuleID: "test-rule",
				File:   "app.js",
			},
			WasFalsePositive: true,
		},
		{
			Finding: agent.Finding{
				RuleID: "other-rule",
				File:   "other.js",
			},
			WasFalsePositive: false,
		},
	}

	// Test historical accuracy calculation
	accuracy := extractor.calculateHistoricalAccuracy(finding, historical)
	assert.Equal(t, 0.5, accuracy) // 1 correct out of 2 similar findings

	// Test false positive rate calculation
	fpRate := extractor.calculateFalsePositiveRate(finding, historical)
	assert.Equal(t, 0.5, fpRate) // 1 false positive out of 2 similar findings

	// Test similar finding count
	count := extractor.countSimilarFindings(finding, historical)
	assert.Equal(t, 2, count) // 2 similar findings
}

func TestFeatureExtractor_UpdateMethods(t *testing.T) {
	extractor := NewFeatureExtractor()

	// Test updating tool reliability
	extractor.UpdateToolReliability("semgrep", 0.95)
	// Note: We don't have a getter for tool reliability in the extractor
	// This is by design as tool reliability is managed by the ReliabilityTracker

	// Test updating rule reliability
	extractor.UpdateRuleReliability("test-rule", 0.85)
	ruleReliability := extractor.getRuleReliability("test-rule")
	assert.Equal(t, 0.85, ruleReliability)

	// Test updating file risk score
	extractor.UpdateFileRiskScore("app.js", 0.7)
	riskScore := extractor.getFileRiskScore("app.js")
	assert.Equal(t, 0.7, riskScore)
}

// Benchmark tests for ML model performance
func BenchmarkLinearRegressionModel_Predict(b *testing.B) {
	model := NewLinearRegressionModel()
	ctx := context.Background()

	features := MLFeatures{
		ToolCount:           3,
		AverageConfidence:   0.8,
		WeightedReliability: 0.85,
		SeverityConsensus:   1.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := model.Predict(ctx, features)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFeatureExtractor_ExtractFeatures(b *testing.B) {
	extractor := NewFeatureExtractor()

	group := FindingGroup{
		PrimaryFinding: agent.Finding{
			Tool:       "semgrep",
			Severity:   agent.SeverityHigh,
			Category:   agent.CategoryXSS,
			Confidence: 0.9,
		},
		SimilarFindings: []agent.Finding{
			{
				Tool:       "eslint",
				Severity:   agent.SeverityHigh,
				Category:   agent.CategoryXSS,
				Confidence: 0.8,
			},
		},
		Tools: []string{"semgrep", "eslint"},
	}

	context := ConsensusContext{
		TotalAgents:      2,
		AgentReliability: map[string]float64{"semgrep": 0.9, "eslint": 0.8},
		HistoricalData:   []HistoricalFinding{},
	}

	toolReliability := map[string]float64{"semgrep": 0.9, "eslint": 0.8}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.ExtractFeatures(group, context, toolReliability)
	}
}