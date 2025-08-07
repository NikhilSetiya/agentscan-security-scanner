package consensus

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/agentscan/agentscan/pkg/errors"
)

// Engine implements the ConsensusEngine interface
type Engine struct {
	config              Config
	similarityConfig    SimilarityConfig
	confidenceThresholds ConfidenceThresholds
	stats               ConsensusStats
	startTime           time.Time
}

// Config contains consensus engine configuration
type Config struct {
	ModelVersion        string  `json:"model_version"`
	MinAgreementCount   int     `json:"min_agreement_count"`   // Minimum tools needed for high confidence
	MaxDisagreementRate float64 `json:"max_disagreement_rate"` // Maximum disagreement rate for high confidence
	EnableSimilarityMatching bool `json:"enable_similarity_matching"`
	EnableLearning      bool    `json:"enable_learning"`
}

// DefaultConfig returns default consensus engine configuration
func DefaultConfig() Config {
	return Config{
		ModelVersion:        "1.0.0",
		MinAgreementCount:   3,
		MaxDisagreementRate: 0.2,
		EnableSimilarityMatching: true,
		EnableLearning:      true,
	}
}

// NewEngine creates a new consensus engine
func NewEngine(config Config) *Engine {
	return &Engine{
		config:              config,
		similarityConfig:    DefaultSimilarityConfig(),
		confidenceThresholds: DefaultConfidenceThresholds(),
		stats:               ConsensusStats{},
		startTime:           time.Now(),
	}
}

// NewEngineWithConfig creates a new consensus engine with custom configuration
func NewEngineWithConfig(config Config, similarityConfig SimilarityConfig, confidenceThresholds ConfidenceThresholds) *Engine {
	return &Engine{
		config:              config,
		similarityConfig:    similarityConfig,
		confidenceThresholds: confidenceThresholds,
		stats:               ConsensusStats{},
		startTime:           time.Now(),
	}
}

// AnalyzeFindings processes multiple agent results and returns consensus findings
func (e *Engine) AnalyzeFindings(ctx context.Context, findings []agent.Finding) (*ConsensusResult, error) {
	startTime := time.Now()
	
	if len(findings) == 0 {
		return &ConsensusResult{
			DeduplicatedFindings: []ConsensusFinding{},
			Statistics:          e.stats,
			ModelVersion:        e.config.ModelVersion,
			ProcessingTime:      time.Since(startTime),
		}, nil
	}

	// Step 1: Deduplicate findings using similarity matching
	var findingGroups []FindingGroup
	var err error
	
	if e.config.EnableSimilarityMatching {
		findingGroups, err = e.DeduplicateFindings(ctx, findings)
		if err != nil {
			return nil, errors.NewInternalError("failed to deduplicate findings").WithCause(err)
		}
	} else {
		// If similarity matching is disabled, treat each finding as its own group
		findingGroups = make([]FindingGroup, len(findings))
		for i, finding := range findings {
			findingGroups[i] = FindingGroup{
				ID:             uuid.New().String(),
				PrimaryFinding: finding,
				SimilarFindings: []agent.Finding{},
				SimilarityScore: 1.0,
				Tools:          []string{finding.Tool},
			}
		}
	}

	// Step 2: Calculate consensus for each group
	consensusFindings := make([]ConsensusFinding, len(findingGroups))
	
	for i, group := range findingGroups {
		consensusFinding, err := e.calculateConsensus(ctx, group)
		if err != nil {
			return nil, errors.NewInternalError("failed to calculate consensus").WithCause(err)
		}
		consensusFindings[i] = consensusFinding
	}

	// Step 3: Sort by consensus score (highest first)
	sort.Slice(consensusFindings, func(i, j int) bool {
		return consensusFindings[i].ConsensusScore > consensusFindings[j].ConsensusScore
	})

	// Step 4: Update statistics
	e.updateStats(findings, consensusFindings, time.Since(startTime))

	return &ConsensusResult{
		DeduplicatedFindings: consensusFindings,
		Statistics:          e.stats,
		ModelVersion:        e.config.ModelVersion,
		ProcessingTime:      time.Since(startTime),
	}, nil
}

// DeduplicateFindings removes duplicate findings using semantic similarity
func (e *Engine) DeduplicateFindings(ctx context.Context, findings []agent.Finding) ([]FindingGroup, error) {
	if len(findings) == 0 {
		return []FindingGroup{}, nil
	}

	groups := make([]FindingGroup, 0)
	processed := make(map[string]bool)

	for i, finding := range findings {
		if processed[finding.ID] {
			continue
		}

		// Create a new group with this finding as primary
		group := FindingGroup{
			ID:             uuid.New().String(),
			PrimaryFinding: finding,
			SimilarFindings: []agent.Finding{},
			Tools:          []string{finding.Tool},
		}

		// Find similar findings
		for j := i + 1; j < len(findings); j++ {
			if processed[findings[j].ID] {
				continue
			}

			similarity := e.calculateSimilarity(finding, findings[j])
			if similarity >= e.similarityConfig.MinSimilarityThreshold {
				group.SimilarFindings = append(group.SimilarFindings, findings[j])
				group.Tools = append(group.Tools, findings[j].Tool)
				processed[findings[j].ID] = true
			}
		}

		// Calculate average similarity score
		if len(group.SimilarFindings) > 0 {
			totalSimilarity := 0.0
			for _, similar := range group.SimilarFindings {
				totalSimilarity += e.calculateSimilarity(finding, similar)
			}
			group.SimilarityScore = totalSimilarity / float64(len(group.SimilarFindings))
		} else {
			group.SimilarityScore = 1.0
		}

		// Remove duplicate tools
		group.Tools = e.removeDuplicateStrings(group.Tools)

		groups = append(groups, group)
		processed[finding.ID] = true
	}

	return groups, nil
}

// calculateSimilarity calculates similarity between two findings
func (e *Engine) calculateSimilarity(f1, f2 agent.Finding) float64 {
	// File path similarity
	filePathSim := e.calculateStringSimilarity(f1.File, f2.File)
	
	// Rule ID similarity
	ruleIDSim := e.calculateStringSimilarity(f1.RuleID, f2.RuleID)
	
	// Message similarity
	messageSim := e.calculateStringSimilarity(f1.Title, f2.Title)
	
	// Location similarity (line numbers)
	locationSim := e.calculateLocationSimilarity(f1.Line, f2.Line)

	// Weighted average
	similarity := (filePathSim * e.similarityConfig.FilePathWeight) +
		(ruleIDSim * e.similarityConfig.RuleIDWeight) +
		(messageSim * e.similarityConfig.MessageWeight) +
		(locationSim * e.similarityConfig.LocationWeight)

	// Additional boost for same file and similar location
	if f1.File == f2.File && abs(f1.Line-f2.Line) <= 5 {
		similarity = maxFloat(similarity, 0.85) // Boost similarity for same file + close lines
	}

	// Additional boost for same category and severity
	if f1.Category == f2.Category && f1.Severity == f2.Severity {
		similarity += 0.1 // Small boost for same category/severity
	}

	return minFloat(1.0, similarity)
}

// calculateStringSimilarity calculates similarity between two strings using Levenshtein distance
func (e *Engine) calculateStringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Simple similarity based on common substrings and length
	longerLength := len(s1)
	if len(s2) > len(s1) {
		longerLength = len(s2)
	}
	if longerLength == 0 {
		return 1.0
	}

	// Calculate edit distance (simplified)
	editDistance := e.levenshteinDistance(s1, s2)
	return (float64(longerLength) - float64(editDistance)) / float64(longerLength)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func (e *Engine) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = minInt(
				minInt(matrix[i-1][j]+1, matrix[i][j-1]+1),      // deletion, insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// calculateLocationSimilarity calculates similarity based on line numbers
func (e *Engine) calculateLocationSimilarity(line1, line2 int) float64 {
	if line1 == line2 {
		return 1.0
	}

	// Consider lines within 5 lines as similar
	diff := abs(line1 - line2)
	if diff <= 5 {
		return 1.0 - (float64(diff) / 5.0)
	}

	return 0.0
}

// calculateConsensus calculates consensus information for a finding group
func (e *Engine) calculateConsensus(ctx context.Context, group FindingGroup) (ConsensusFinding, error) {
	allFindings := append([]agent.Finding{group.PrimaryFinding}, group.SimilarFindings...)
	
	// Count agreements and disagreements
	agreementCount := len(group.Tools)
	
	// Determine final severity and category based on consensus
	finalSeverity := e.calculateConsensusSeverity(allFindings)
	finalCategory := e.calculateConsensusCategory(allFindings)
	
	// Calculate consensus score
	consensusScore := e.calculateConsensusScore(agreementCount, 0, len(group.Tools))
	
	// Collect similar finding IDs
	similarFindingIDs := make([]string, len(group.SimilarFindings))
	for i, finding := range group.SimilarFindings {
		similarFindingIDs[i] = finding.ID
	}

	// Create consensus finding based on primary finding
	consensusFinding := ConsensusFinding{
		Finding:           group.PrimaryFinding,
		ConsensusScore:    consensusScore,
		AgreementCount:    agreementCount,
		DisagreementCount: 0, // TODO: Implement disagreement detection
		SupportingTools:   group.Tools,
		ConflictingTools:  []string{}, // TODO: Implement conflict detection
		SimilarFindings:   similarFindingIDs,
		FinalSeverity:     finalSeverity,
		FinalCategory:     finalCategory,
	}

	// Update the finding with consensus-determined values
	consensusFinding.Finding.Severity = finalSeverity
	consensusFinding.Finding.Category = finalCategory

	return consensusFinding, nil
}

// calculateConsensusSeverity determines the final severity based on all findings
func (e *Engine) calculateConsensusSeverity(findings []agent.Finding) agent.Severity {
	if len(findings) == 0 {
		return agent.SeverityMedium
	}

	// Count severity votes
	severityCounts := make(map[agent.Severity]int)
	for _, finding := range findings {
		severityCounts[finding.Severity]++
	}

	// Find the most common severity
	maxCount := 0
	var consensusSeverity agent.Severity = agent.SeverityMedium
	
	for severity, count := range severityCounts {
		if count > maxCount {
			maxCount = count
			consensusSeverity = severity
		}
	}

	return consensusSeverity
}

// calculateConsensusCategory determines the final category based on all findings
func (e *Engine) calculateConsensusCategory(findings []agent.Finding) agent.VulnCategory {
	if len(findings) == 0 {
		return agent.CategoryOther
	}

	// Count category votes
	categoryCounts := make(map[agent.VulnCategory]int)
	for _, finding := range findings {
		categoryCounts[finding.Category]++
	}

	// Find the most common category
	maxCount := 0
	var consensusCategory agent.VulnCategory = agent.CategoryOther
	
	for category, count := range categoryCounts {
		if count > maxCount {
			maxCount = count
			consensusCategory = category
		}
	}

	return consensusCategory
}

// calculateConsensusScore calculates the consensus score based on agreement/disagreement
func (e *Engine) calculateConsensusScore(agreementCount, disagreementCount, totalTools int) float64 {
	if totalTools == 0 {
		return 0.0
	}

	// Base score from agreement ratio
	agreementRatio := float64(agreementCount) / float64(totalTools)
	
	// Apply confidence thresholds based on agreement count
	if agreementCount >= e.config.MinAgreementCount {
		// High confidence if enough tools agree
		return maxFloat(agreementRatio, e.confidenceThresholds.High)
	} else if agreementCount >= 2 {
		// Medium confidence if at least 2 tools agree
		return maxFloat(agreementRatio, e.confidenceThresholds.Medium)
	} else {
		// Low confidence if only 1 tool found it
		return minFloat(agreementRatio, 0.6) // Cap single-tool findings at 0.6
	}
}

// GetConfidenceScore calculates confidence for a finding based on consensus context
func (e *Engine) GetConfidenceScore(ctx context.Context, finding agent.Finding, context ConsensusContext) (float64, error) {
	// Base confidence from the finding itself
	baseConfidence := finding.Confidence

	// Adjust based on agent reliability
	agentReliability := 1.0
	if reliability, exists := context.AgentReliability[finding.Tool]; exists {
		agentReliability = reliability
	}

	// Adjust based on historical data
	historicalAdjustment := e.calculateHistoricalAdjustment(finding, context.HistoricalData)

	// Combine factors
	finalConfidence := baseConfidence * agentReliability * historicalAdjustment

	// Ensure confidence is within bounds
	return maxFloat(0.0, minFloat(1.0, finalConfidence)), nil
}

// calculateHistoricalAdjustment adjusts confidence based on historical data
func (e *Engine) calculateHistoricalAdjustment(finding agent.Finding, historical []HistoricalFinding) float64 {
	if len(historical) == 0 {
		return 1.0
	}

	// Find similar historical findings
	similarCount := 0
	falsePositiveCount := 0

	for _, hist := range historical {
		if e.calculateSimilarity(finding, hist.Finding) > 0.8 {
			similarCount++
			if hist.WasFalsePositive {
				falsePositiveCount++
			}
		}
	}

	if similarCount == 0 {
		return 1.0
	}

	// Reduce confidence if historically many false positives
	falsePositiveRate := float64(falsePositiveCount) / float64(similarCount)
	return 1.0 - (falsePositiveRate * 0.3) // Reduce by up to 30%
}

// UpdateModel trains the consensus model with user feedback
func (e *Engine) UpdateModel(ctx context.Context, feedback []UserFeedback) error {
	if !e.config.EnableLearning {
		return nil
	}

	// TODO: Implement machine learning model updates
	// For now, we'll just log that we received feedback
	
	for _, fb := range feedback {
		// Update internal statistics based on feedback
		switch fb.Action {
		case "false_positive":
			// Reduce confidence for similar findings in the future
		case "confirmed":
			// Increase confidence for similar findings
		case "fixed":
			// Mark as legitimate finding
		}
	}

	return nil
}

// GetStats returns consensus engine statistics
func (e *Engine) GetStats() ConsensusStats {
	e.stats.ProcessingTime = time.Since(e.startTime)
	return e.stats
}

// updateStats updates internal statistics
func (e *Engine) updateStats(originalFindings []agent.Finding, consensusFindings []ConsensusFinding, processingTime time.Duration) {
	e.stats.TotalFindings = len(originalFindings)
	e.stats.DeduplicatedFindings = len(consensusFindings)
	e.stats.ProcessingTime = processingTime

	// Count confidence levels
	e.stats.HighConfidenceCount = 0
	e.stats.MediumConfidenceCount = 0
	e.stats.LowConfidenceCount = 0
	totalConfidence := 0.0

	categoryDist := make(map[string]int)
	severityDist := make(map[string]int)

	for _, finding := range consensusFindings {
		totalConfidence += finding.ConsensusScore
		
		if finding.ConsensusScore >= e.confidenceThresholds.High {
			e.stats.HighConfidenceCount++
		} else if finding.ConsensusScore >= e.confidenceThresholds.Medium {
			e.stats.MediumConfidenceCount++
		} else {
			e.stats.LowConfidenceCount++
		}

		categoryDist[string(finding.FinalCategory)]++
		severityDist[string(finding.FinalSeverity)]++
	}

	if len(consensusFindings) > 0 {
		e.stats.AverageConfidence = totalConfidence / float64(len(consensusFindings))
	}

	e.stats.CategoryDistribution = categoryDist
	e.stats.SeverityDistribution = severityDist
}

// Helper functions

func (e *Engine) removeDuplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}