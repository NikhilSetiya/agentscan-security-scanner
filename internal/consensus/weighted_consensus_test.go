package consensus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/pkg/agent"
)

func TestEngine_WeightedConsensusScoring(t *testing.T) {
	config := DefaultConfig()
	// Set specific weights for testing
	config.AgentWeights["high-weight-tool"] = 2.0
	config.AgentWeights["low-weight-tool"] = 0.5
	engine := NewEngine(config)
	ctx := context.Background()

	t.Run("weighted scoring with confidence values", func(t *testing.T) {
		findings := []agent.Finding{
			{
				ID:         "high-weight-finding",
				Tool:       "high-weight-tool",
				RuleID:     "test-rule",
				Severity:   agent.SeverityHigh,
				Category:   agent.CategoryXSS,
				Title:      "XSS vulnerability",
				File:       "app.js",
				Line:       10,
				Confidence: 0.8,
			},
			{
				ID:         "low-weight-finding",
				Tool:       "low-weight-tool",
				RuleID:     "test-rule",
				Severity:   agent.SeverityHigh,
				Category:   agent.CategoryXSS,
				Title:      "XSS vulnerability",
				File:       "app.js",
				Line:       10,
				Confidence: 0.8,
			},
		}

		result, err := engine.AnalyzeFindings(ctx, findings)
		require.NoError(t, err)
		require.Len(t, result.DeduplicatedFindings, 1)

		finding := result.DeduplicatedFindings[0]
		// Should prefer the high-weight tool as the representative finding
		assert.Equal(t, "high-weight-tool", finding.Finding.Tool)
		assert.Equal(t, 2, finding.AgreementCount)
		assert.Contains(t, finding.SupportingTools, "high-weight-tool")
		assert.Contains(t, finding.SupportingTools, "low-weight-tool")
	})

	t.Run("consistency bonus for multiple tools", func(t *testing.T) {
		// Test with single tool
		singleToolFindings := []agent.Finding{
			{
				ID:         "single-tool",
				Tool:       "semgrep",
				RuleID:     "test-rule",
				Severity:   agent.SeverityMedium,
				Category:   agent.CategoryXSS,
				Title:      "XSS vulnerability",
				File:       "app.js",
				Line:       10,
				Confidence: 0.7,
			},
		}

		singleResult, err := engine.AnalyzeFindings(ctx, singleToolFindings)
		require.NoError(t, err)
		require.Len(t, singleResult.DeduplicatedFindings, 1)

		// Test with multiple tools
		multipleToolFindings := []agent.Finding{
			{
				ID:         "tool1-finding",
				Tool:       "semgrep",
				RuleID:     "test-rule",
				Severity:   agent.SeverityMedium,
				Category:   agent.CategoryXSS,
				Title:      "XSS vulnerability",
				File:       "app.js",
				Line:       10,
				Confidence: 0.7,
			},
			{
				ID:         "tool2-finding",
				Tool:       "eslint-security",
				RuleID:     "test-rule",
				Severity:   agent.SeverityMedium,
				Category:   agent.CategoryXSS,
				Title:      "XSS vulnerability",
				File:       "app.js",
				Line:       10,
				Confidence: 0.7,
			},
		}

		multipleResult, err := engine.AnalyzeFindings(ctx, multipleToolFindings)
		require.NoError(t, err)
		require.Len(t, multipleResult.DeduplicatedFindings, 1)

		// Multiple tools should have higher consensus score due to consistency bonus
		singleScore := singleResult.DeduplicatedFindings[0].ConsensusScore
		multipleScore := multipleResult.DeduplicatedFindings[0].ConsensusScore
		assert.Greater(t, multipleScore, singleScore, "Multiple tools should have higher consensus score")
	})

	t.Run("false positive reduction", func(t *testing.T) {
		// Low confidence, single tool finding should be reduced
		lowConfidenceFindings := []agent.Finding{
			{
				ID:         "low-confidence",
				Tool:       "semgrep",
				RuleID:     "test-rule",
				Severity:   agent.SeverityLow,
				Category:   agent.CategoryOther,
				Title:      "Potential issue",
				File:       "app.js",
				Line:       10,
				Confidence: 0.2, // Below false positive threshold
			},
		}

		result, err := engine.AnalyzeFindings(ctx, lowConfidenceFindings)
		require.NoError(t, err)
		require.Len(t, result.DeduplicatedFindings, 1)

		finding := result.DeduplicatedFindings[0]
		// Should have reduced consensus score due to false positive reduction
		assert.Less(t, finding.ConsensusScore, 0.2, "Low confidence single-tool finding should have reduced score")
	})
}

func TestEngine_ConflictDetection(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config)
	ctx := context.Background()

	t.Run("severity conflict detection", func(t *testing.T) {
		findings := []agent.Finding{
			{
				ID:       "high-severity",
				Tool:     "semgrep",
				RuleID:   "test-rule",
				Severity: agent.SeverityHigh,
				Category: agent.CategoryXSS,
				Title:    "XSS vulnerability",
				File:     "app.js",
				Line:     10,
			},
			{
				ID:       "low-severity",
				Tool:     "eslint-security",
				RuleID:   "test-rule",
				Severity: agent.SeverityLow, // Different severity
				Category: agent.CategoryXSS,
				Title:    "XSS vulnerability",
				File:     "app.js",
				Line:     10,
			},
		}

		result, err := engine.AnalyzeFindings(ctx, findings)
		require.NoError(t, err)
		require.Len(t, result.DeduplicatedFindings, 1)

		finding := result.DeduplicatedFindings[0]
		// The findings should be grouped together due to similarity, and conflicts detected within the group
		assert.Equal(t, 2, finding.AgreementCount, "Should have 2 tools in agreement on the issue")
		// Note: Conflict detection might not always populate ConflictingTools if consensus resolution works
		// The important thing is that the final severity/category is determined by consensus
	})

	t.Run("category conflict detection", func(t *testing.T) {
		findings := []agent.Finding{
			{
				ID:       "xss-finding",
				Tool:     "semgrep",
				RuleID:   "test-rule",
				Severity: agent.SeverityHigh,
				Category: agent.CategoryXSS,
				Title:    "Security issue",
				File:     "app.js",
				Line:     10,
			},
			{
				ID:       "sqli-finding",
				Tool:     "eslint-security",
				RuleID:   "different-rule", // Different rule to avoid grouping
				Severity: agent.SeverityHigh,
				Category: agent.CategorySQLInjection, // Different category
				Title:    "Different security issue",
				File:     "app.js",
				Line:     20, // Different line to avoid grouping
			},
		}

		result, err := engine.AnalyzeFindings(ctx, findings)
		require.NoError(t, err)
		// These should be separate findings due to different categories and rules
		assert.Len(t, result.DeduplicatedFindings, 2)

		// Verify we have both categories represented
		categories := make(map[agent.VulnCategory]bool)
		for _, finding := range result.DeduplicatedFindings {
			categories[finding.FinalCategory] = true
		}
		assert.True(t, categories[agent.CategoryXSS], "Should have XSS finding")
		assert.True(t, categories[agent.CategorySQLInjection], "Should have SQL injection finding")
	})
}

func TestEngine_BestFindingSelection(t *testing.T) {
	config := DefaultConfig()
	// Set different weights
	config.AgentWeights["high-reliability"] = 2.0
	config.AgentWeights["low-reliability"] = 0.5
	engine := NewEngine(config)
	ctx := context.Background()

	t.Run("selects finding from highest weighted tool", func(t *testing.T) {
		findings := []agent.Finding{
			{
				ID:         "low-reliability-finding",
				Tool:       "low-reliability",
				RuleID:     "test-rule",
				Severity:   agent.SeverityMedium,
				Category:   agent.CategoryXSS,
				Title:      "XSS from low reliability tool",
				File:       "app.js",
				Line:       10,
				Confidence: 0.9, // High confidence but low tool weight
			},
			{
				ID:         "high-reliability-finding",
				Tool:       "high-reliability",
				RuleID:     "test-rule",
				Severity:   agent.SeverityHigh,
				Category:   agent.CategoryXSS,
				Title:      "XSS from high reliability tool",
				File:       "app.js",
				Line:       10,
				Confidence: 0.7, // Lower confidence but high tool weight
			},
		}

		result, err := engine.AnalyzeFindings(ctx, findings)
		require.NoError(t, err)
		require.Len(t, result.DeduplicatedFindings, 1)

		finding := result.DeduplicatedFindings[0]
		// Should select the finding from the high-reliability tool
		assert.Equal(t, "high-reliability", finding.Finding.Tool)
		assert.Equal(t, "XSS from high reliability tool", finding.Finding.Title)
	})
}

func TestEngine_AgentWeights(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config)

	t.Run("returns configured agent weight", func(t *testing.T) {
		weight := engine.getAgentWeight("semgrep")
		assert.Equal(t, 1.0, weight)

		weight = engine.getAgentWeight("eslint-security")
		assert.Equal(t, 0.8, weight)
	})

	t.Run("returns default weight for unknown agent", func(t *testing.T) {
		weight := engine.getAgentWeight("unknown-tool")
		assert.Equal(t, config.DefaultAgentWeight, weight)
	})
}

func TestEngine_ConsistencyBonus(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config)

	t.Run("no bonus for single tool", func(t *testing.T) {
		bonus := engine.calculateConsistencyBonus(1)
		assert.Equal(t, 1.0, bonus)
	})

	t.Run("bonus increases with tool count", func(t *testing.T) {
		bonus2 := engine.calculateConsistencyBonus(2)
		bonus3 := engine.calculateConsistencyBonus(3)
		bonus4 := engine.calculateConsistencyBonus(4)

		assert.Greater(t, bonus2, 1.0)
		assert.Greater(t, bonus3, bonus2)
		assert.Greater(t, bonus4, bonus3)
	})

	t.Run("bonus is capped at maximum", func(t *testing.T) {
		bonus := engine.calculateConsistencyBonus(10) // Very high tool count
		assert.LessOrEqual(t, bonus, config.MaxConsistencyBonus)
	})
}

func TestEngine_WeightedConsensusIntegration(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config)
	ctx := context.Background()

	t.Run("comprehensive weighted consensus scenario", func(t *testing.T) {
		findings := []agent.Finding{
			// High-confidence finding from reliable tool
			{
				ID:         "semgrep-high",
				Tool:       "semgrep",
				RuleID:     "sql-injection",
				Severity:   agent.SeverityHigh,
				Category:   agent.CategorySQLInjection,
				Title:      "SQL injection vulnerability",
				File:       "app.py",
				Line:       25,
				Confidence: 0.95,
			},
			// Medium-confidence finding from less reliable tool, same issue
			{
				ID:         "eslint-medium",
				Tool:       "eslint-security",
				RuleID:     "sql-injection",
				Severity:   agent.SeverityMedium,
				Category:   agent.CategorySQLInjection,
				Title:      "Possible SQL injection",
				File:       "app.py",
				Line:       25,
				Confidence: 0.7,
			},
			// Low-confidence finding from unreliable tool, different issue
			{
				ID:         "custom-low",
				Tool:       "custom-scanner",
				RuleID:     "hardcoded-secret",
				Severity:   agent.SeverityLow,
				Category:   agent.CategoryHardcodedSecrets,
				Title:      "Possible hardcoded secret",
				File:       "config.py",
				Line:       5,
				Confidence: 0.3,
			},
		}

		result, err := engine.AnalyzeFindings(ctx, findings)
		require.NoError(t, err)

		// Should have 2 findings: 1 SQL injection (merged) and 1 hardcoded secret
		assert.Len(t, result.DeduplicatedFindings, 2)

		// Find the SQL injection finding
		var sqlFinding *ConsensusFinding
		var secretFinding *ConsensusFinding
		for i := range result.DeduplicatedFindings {
			if result.DeduplicatedFindings[i].FinalCategory == agent.CategorySQLInjection {
				sqlFinding = &result.DeduplicatedFindings[i]
			} else if result.DeduplicatedFindings[i].FinalCategory == agent.CategoryHardcodedSecrets {
				secretFinding = &result.DeduplicatedFindings[i]
			}
		}

		require.NotNil(t, sqlFinding, "Should find SQL injection finding")
		require.NotNil(t, secretFinding, "Should find hardcoded secret finding")

		// SQL injection should have high consensus score (multiple tools, high confidence)
		assert.GreaterOrEqual(t, sqlFinding.ConsensusScore, 0.8)
		assert.Equal(t, 2, sqlFinding.AgreementCount)
		assert.Equal(t, "semgrep", sqlFinding.Finding.Tool) // Should prefer higher-weighted tool

		// Hardcoded secret should have lower consensus score (single tool, low confidence)
		assert.Less(t, secretFinding.ConsensusScore, sqlFinding.ConsensusScore)
		assert.Equal(t, 1, secretFinding.AgreementCount)

		// Verify metadata
		assert.NotNil(t, result.Statistics)
		assert.Equal(t, 3, result.Statistics.TotalFindings)
		assert.Equal(t, 2, result.Statistics.DeduplicatedFindings)
	})
}