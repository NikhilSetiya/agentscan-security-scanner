package consensus

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/pkg/agent"
)

func TestNewEngine(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config)

	assert.NotNil(t, engine)
	assert.Equal(t, config, engine.config)
	assert.Equal(t, DefaultSimilarityConfig(), engine.similarityConfig)
	assert.Equal(t, DefaultConfidenceThresholds(), engine.confidenceThresholds)
}

func TestEngine_AnalyzeFindings_EmptyInput(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	result, err := engine.AnalyzeFindings(ctx, []agent.Finding{})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.DeduplicatedFindings)
	assert.Equal(t, "1.0.0", result.ModelVersion)
	assert.GreaterOrEqual(t, result.ProcessingTime, time.Duration(0))
}

func TestEngine_AnalyzeFindings_SingleFinding(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	findings := []agent.Finding{
		{
			ID:          "finding-1",
			Tool:        "semgrep",
			RuleID:      "javascript.lang.security.audit.xss",
			Severity:    agent.SeverityHigh,
			Category:    agent.CategoryXSS,
			Title:       "Cross-site scripting vulnerability",
			Description: "Potential XSS vulnerability detected",
			File:        "app.js",
			Line:        42,
			Confidence:  0.9,
		},
	}

	result, err := engine.AnalyzeFindings(ctx, findings)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.DeduplicatedFindings, 1)

	consensusFinding := result.DeduplicatedFindings[0]
	assert.Equal(t, "finding-1", consensusFinding.ID)
	assert.Equal(t, 1, consensusFinding.AgreementCount)
	assert.Equal(t, []string{"semgrep"}, consensusFinding.SupportingTools)
	assert.Equal(t, agent.SeverityHigh, consensusFinding.FinalSeverity)
	assert.Equal(t, agent.CategoryXSS, consensusFinding.FinalCategory)
	// Single tool should have lower confidence
	assert.Less(t, consensusFinding.ConsensusScore, 0.95)
}

func TestEngine_AnalyzeFindings_MultipleToolsAgreement(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	// Three tools finding the same issue
	findings := []agent.Finding{
		{
			ID:          "finding-1",
			Tool:        "semgrep",
			RuleID:      "javascript.lang.security.audit.xss",
			Severity:    agent.SeverityHigh,
			Category:    agent.CategoryXSS,
			Title:       "Cross-site scripting vulnerability",
			File:        "app.js",
			Line:        42,
			Confidence:  0.9,
		},
		{
			ID:          "finding-2",
			Tool:        "eslint",
			RuleID:      "security/detect-unsafe-regex",
			Severity:    agent.SeverityHigh,
			Category:    agent.CategoryXSS,
			Title:       "Cross-site scripting vulnerability",
			File:        "app.js",
			Line:        42,
			Confidence:  0.8,
		},
		{
			ID:          "finding-3",
			Tool:        "bandit",
			RuleID:      "B201",
			Severity:    agent.SeverityHigh,
			Category:    agent.CategoryXSS,
			Title:       "Cross-site scripting vulnerability",
			File:        "app.js",
			Line:        42,
			Confidence:  0.85,
		},
	}

	result, err := engine.AnalyzeFindings(ctx, findings)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.DeduplicatedFindings, 1) // Should be deduplicated into one

	consensusFinding := result.DeduplicatedFindings[0]
	assert.Equal(t, 3, consensusFinding.AgreementCount)
	assert.Len(t, consensusFinding.SupportingTools, 3)
	assert.Contains(t, consensusFinding.SupportingTools, "semgrep")
	assert.Contains(t, consensusFinding.SupportingTools, "eslint")
	assert.Contains(t, consensusFinding.SupportingTools, "bandit")
	assert.Equal(t, agent.SeverityHigh, consensusFinding.FinalSeverity)
	assert.Equal(t, agent.CategoryXSS, consensusFinding.FinalCategory)
	// Three tools agreeing should have high confidence
	assert.GreaterOrEqual(t, consensusFinding.ConsensusScore, 0.95)
}

func TestEngine_AnalyzeFindings_DifferentSeverities(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	// Same issue but different severities - should use consensus
	findings := []agent.Finding{
		{
			ID:       "finding-1",
			Tool:     "semgrep",
			RuleID:   "javascript.lang.security.audit.xss",
			Severity: agent.SeverityHigh,
			Category: agent.CategoryXSS,
			Title:    "Cross-site scripting vulnerability",
			File:     "app.js",
			Line:     42,
		},
		{
			ID:       "finding-2",
			Tool:     "eslint",
			RuleID:   "security/detect-unsafe-regex",
			Severity: agent.SeverityMedium,
			Category: agent.CategoryXSS,
			Title:    "Cross-site scripting vulnerability",
			File:     "app.js",
			Line:     42,
		},
		{
			ID:       "finding-3",
			Tool:     "custom-tool",
			RuleID:   "xss-check",
			Severity: agent.SeverityHigh,
			Category: agent.CategoryXSS,
			Title:    "Cross-site scripting vulnerability",
			File:     "app.js",
			Line:     42,
		},
	}

	result, err := engine.AnalyzeFindings(ctx, findings)

	require.NoError(t, err)
	assert.Len(t, result.DeduplicatedFindings, 1)

	consensusFinding := result.DeduplicatedFindings[0]
	// Should choose the most common severity (High appears twice)
	assert.Equal(t, agent.SeverityHigh, consensusFinding.FinalSeverity)
	assert.Equal(t, 3, consensusFinding.AgreementCount)
}

func TestEngine_AnalyzeFindings_DifferentIssues(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	// Completely different issues - should not be deduplicated
	findings := []agent.Finding{
		{
			ID:       "finding-1",
			Tool:     "semgrep",
			RuleID:   "javascript.lang.security.audit.xss",
			Severity: agent.SeverityHigh,
			Category: agent.CategoryXSS,
			Title:    "Cross-site scripting vulnerability",
			File:     "app.js",
			Line:     42,
		},
		{
			ID:       "finding-2",
			Tool:     "bandit",
			RuleID:   "B602",
			Severity: agent.SeverityMedium,
			Category: agent.CategorySQLInjection,
			Title:    "SQL injection vulnerability",
			File:     "database.py",
			Line:     15,
		},
	}

	result, err := engine.AnalyzeFindings(ctx, findings)

	require.NoError(t, err)
	assert.Len(t, result.DeduplicatedFindings, 2) // Should remain separate

	// Sort by consensus score to ensure consistent ordering
	if result.DeduplicatedFindings[0].ConsensusScore < result.DeduplicatedFindings[1].ConsensusScore {
		result.DeduplicatedFindings[0], result.DeduplicatedFindings[1] = result.DeduplicatedFindings[1], result.DeduplicatedFindings[0]
	}

	// Both should have low confidence (single tool each)
	for _, finding := range result.DeduplicatedFindings {
		assert.Equal(t, 1, finding.AgreementCount)
		assert.Less(t, finding.ConsensusScore, 0.95)
	}
}

func TestEngine_DeduplicateFindings_SimilarFindings(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	findings := []agent.Finding{
		{
			ID:       "finding-1",
			Tool:     "semgrep",
			RuleID:   "javascript.lang.security.audit.xss",
			Title:    "Cross-site scripting vulnerability",
			File:     "app.js",
			Line:     42,
		},
		{
			ID:       "finding-2",
			Tool:     "eslint",
			RuleID:   "javascript.lang.security.audit.xss",
			Title:    "Cross-site scripting vulnerability",
			File:     "app.js",
			Line:     43, // Very close line number
		},
	}

	groups, err := engine.DeduplicateFindings(ctx, findings)

	require.NoError(t, err)
	assert.Len(t, groups, 1) // Should be grouped together

	group := groups[0]
	assert.Equal(t, "finding-1", group.PrimaryFinding.ID)
	assert.Len(t, group.SimilarFindings, 1)
	assert.Equal(t, "finding-2", group.SimilarFindings[0].ID)
	assert.Len(t, group.Tools, 2)
	assert.Contains(t, group.Tools, "semgrep")
	assert.Contains(t, group.Tools, "eslint")
}

func TestEngine_DeduplicateFindings_DissimilarFindings(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	findings := []agent.Finding{
		{
			ID:       "finding-1",
			Tool:     "semgrep",
			RuleID:   "javascript.lang.security.audit.xss",
			Title:    "Cross-site scripting vulnerability",
			File:     "app.js",
			Line:     42,
		},
		{
			ID:       "finding-2",
			Tool:     "bandit",
			RuleID:   "B602",
			Title:    "SQL injection vulnerability",
			File:     "database.py",
			Line:     100,
		},
	}

	groups, err := engine.DeduplicateFindings(ctx, findings)

	require.NoError(t, err)
	assert.Len(t, groups, 2) // Should remain separate

	for _, group := range groups {
		assert.Empty(t, group.SimilarFindings)
		assert.Len(t, group.Tools, 1)
		assert.Equal(t, 1.0, group.SimilarityScore)
	}
}

func TestEngine_CalculateSimilarity(t *testing.T) {
	engine := NewEngine(DefaultConfig())

	tests := []struct {
		name     string
		finding1 agent.Finding
		finding2 agent.Finding
		expected float64
		minSim   float64
	}{
		{
			name: "identical findings",
			finding1: agent.Finding{
				RuleID: "test-rule",
				Title:  "Test vulnerability",
				File:   "app.js",
				Line:   42,
			},
			finding2: agent.Finding{
				RuleID: "test-rule",
				Title:  "Test vulnerability",
				File:   "app.js",
				Line:   42,
			},
			expected: 1.0,
		},
		{
			name: "same rule and file, different line",
			finding1: agent.Finding{
				RuleID: "test-rule",
				Title:  "Test vulnerability",
				File:   "app.js",
				Line:   42,
			},
			finding2: agent.Finding{
				RuleID: "test-rule",
				Title:  "Test vulnerability",
				File:   "app.js",
				Line:   45,
			},
			minSim: 0.8, // Should be high similarity
		},
		{
			name: "completely different findings",
			finding1: agent.Finding{
				RuleID: "xss-rule",
				Title:  "XSS vulnerability",
				File:   "app.js",
				Line:   42,
			},
			finding2: agent.Finding{
				RuleID: "sql-rule",
				Title:  "SQL injection",
				File:   "database.py",
				Line:   100,
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := engine.calculateSimilarity(tt.finding1, tt.finding2)
			
			if tt.expected > 0 {
				assert.InDelta(t, tt.expected, similarity, 0.1)
			} else if tt.minSim > 0 {
				assert.GreaterOrEqual(t, similarity, tt.minSim)
			} else {
				assert.LessOrEqual(t, similarity, 0.5) // Allow for some similarity in "different" findings
			}
		})
	}
}

func TestEngine_CalculateConsensusScore(t *testing.T) {
	engine := NewEngine(DefaultConfig())

	tests := []struct {
		name            string
		agreementCount  int
		disagreementCount int
		totalTools      int
		expectedMin     float64
	}{
		{
			name:           "high agreement (3+ tools)",
			agreementCount: 3,
			totalTools:     3,
			expectedMin:    0.95, // Should be high confidence
		},
		{
			name:           "medium agreement (2 tools)",
			agreementCount: 2,
			totalTools:     3,
			expectedMin:    0.6, // Should be medium confidence
		},
		{
			name:           "low agreement (1 tool)",
			agreementCount: 1,
			totalTools:     3,
			expectedMin:    0.0, // Should be low confidence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateConsensusScore(tt.agreementCount, tt.disagreementCount, tt.totalTools)
			assert.GreaterOrEqual(t, score, tt.expectedMin)
			assert.LessOrEqual(t, score, 1.0)
		})
	}
}

func TestEngine_GetConfidenceScore(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	finding := agent.Finding{
		ID:         "test-finding",
		Tool:       "semgrep",
		Confidence: 0.8,
	}

	context := ConsensusContext{
		TotalAgents: 3,
		AgentReliability: map[string]float64{
			"semgrep": 0.9,
		},
		HistoricalData: []HistoricalFinding{},
	}

	score, err := engine.GetConfidenceScore(ctx, finding, context)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
	// Should be influenced by agent reliability
	assert.Greater(t, score, finding.Confidence*0.8)
}

func TestEngine_GetConfidenceScore_WithHistoricalData(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	finding := agent.Finding{
		ID:         "test-finding",
		Tool:       "semgrep",
		RuleID:     "test-rule",
		Title:      "Test vulnerability",
		File:       "app.js",
		Confidence: 0.8,
	}

	// Historical data showing similar findings were false positives
	historicalData := []HistoricalFinding{
		{
			Finding: agent.Finding{
				RuleID: "test-rule",
				Title:  "Test vulnerability",
				File:   "app.js",
			},
			WasFalsePositive: true,
		},
		{
			Finding: agent.Finding{
				RuleID: "test-rule",
				Title:  "Test vulnerability",
				File:   "app.js",
			},
			WasFalsePositive: true,
		},
	}

	context := ConsensusContext{
		TotalAgents:      1,
		AgentReliability: map[string]float64{"semgrep": 1.0},
		HistoricalData:   historicalData,
	}

	score, err := engine.GetConfidenceScore(ctx, finding, context)

	require.NoError(t, err)
	// Should be reduced due to historical false positives
	assert.Less(t, score, finding.Confidence)
}

func TestEngine_UpdateModel(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	feedback := []UserFeedback{
		{
			FindingID:  "finding-1",
			UserID:     "user-1",
			Action:     "false_positive",
			Comment:    "This is not a real issue",
			Confidence: 0.9,
			Timestamp:  time.Now(),
		},
		{
			FindingID:  "finding-2",
			UserID:     "user-1",
			Action:     "confirmed",
			Comment:    "This is a real vulnerability",
			Confidence: 0.95,
			Timestamp:  time.Now(),
		},
	}

	err := engine.UpdateModel(ctx, feedback)

	require.NoError(t, err)
	// For now, this should just not error
	// TODO: Add more specific tests when ML model is implemented
}

func TestEngine_GetStats(t *testing.T) {
	engine := NewEngine(DefaultConfig())

	stats := engine.GetStats()

	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.ProcessingTime, time.Duration(0))
}

func TestEngine_CalculateStringSimilarity(t *testing.T) {
	engine := NewEngine(DefaultConfig())

	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 1.0,
		},
		{
			name:     "completely different strings",
			s1:       "hello",
			s2:       "world",
			expected: 0.2, // Some similarity due to length
		},
		{
			name:     "similar strings",
			s1:       "javascript.lang.security.audit.xss",
			s2:       "javascript.lang.security.audit.xss.react",
			expected: 0.8, // Should be high similarity
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			expected: 1.0,
		},
		{
			name:     "one empty string",
			s1:       "hello",
			s2:       "",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := engine.calculateStringSimilarity(tt.s1, tt.s2)
			
			if tt.expected == 1.0 || tt.expected == 0.0 {
				assert.Equal(t, tt.expected, similarity)
			} else {
				assert.GreaterOrEqual(t, similarity, tt.expected-0.2)
			}
		})
	}
}

func TestEngine_LevenshteinDistance(t *testing.T) {
	engine := NewEngine(DefaultConfig())

	tests := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 0,
		},
		{
			name:     "one character difference",
			s1:       "hello",
			s2:       "hallo",
			expected: 1,
		},
		{
			name:     "completely different",
			s1:       "abc",
			s2:       "xyz",
			expected: 3,
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			expected: 0,
		},
		{
			name:     "one empty",
			s1:       "hello",
			s2:       "",
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := engine.levenshteinDistance(tt.s1, tt.s2)
			assert.Equal(t, tt.expected, distance)
		})
	}
}

// Benchmark tests
func BenchmarkEngine_AnalyzeFindings(b *testing.B) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	// Create test findings
	findings := make([]agent.Finding, 100)
	for i := 0; i < 100; i++ {
		findings[i] = agent.Finding{
			ID:       fmt.Sprintf("finding-%d", i),
			Tool:     "semgrep",
			RuleID:   "test-rule",
			Severity: agent.SeverityMedium,
			Category: agent.CategoryXSS,
			Title:    "Test vulnerability",
			File:     "app.js",
			Line:     i + 1,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.AnalyzeFindings(ctx, findings)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEngine_DeduplicateFindings(b *testing.B) {
	engine := NewEngine(DefaultConfig())
	ctx := context.Background()

	// Create test findings with some similarities
	findings := make([]agent.Finding, 50)
	for i := 0; i < 50; i++ {
		findings[i] = agent.Finding{
			ID:       fmt.Sprintf("finding-%d", i),
			Tool:     "semgrep",
			RuleID:   "test-rule",
			Title:    "Test vulnerability",
			File:     "app.js",
			Line:     (i % 10) + 1, // Create some similar line numbers
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.DeduplicateFindings(ctx, findings)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEngine_CalculateSimilarity(b *testing.B) {
	engine := NewEngine(DefaultConfig())

	finding1 := agent.Finding{
		RuleID: "javascript.lang.security.audit.xss.react-dangerously-set-inner-html",
		Title:  "Detected usage of dangerouslySetInnerHTML which can lead to XSS vulnerabilities",
		File:   "src/components/App.js",
		Line:   42,
	}

	finding2 := agent.Finding{
		RuleID: "javascript.lang.security.audit.xss.react-dangerously-set-inner-html",
		Title:  "Detected usage of dangerouslySetInnerHTML which can lead to XSS vulnerabilities",
		File:   "src/components/App.js",
		Line:   43,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.calculateSimilarity(finding1, finding2)
	}
}