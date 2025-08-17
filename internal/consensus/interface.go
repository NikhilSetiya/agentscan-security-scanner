package consensus

import (
	"context"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

// ConsensusEngine defines the interface for result deduplication and consensus scoring
type ConsensusEngine interface {
	// AnalyzeFindings processes multiple agent results and returns consensus findings
	AnalyzeFindings(ctx context.Context, findings []agent.Finding) (*ConsensusResult, error)

	// UpdateModel trains the consensus model with user feedback
	UpdateModel(ctx context.Context, feedback []UserFeedback) error

	// GetConfidenceScore calculates confidence for a finding based on consensus context
	GetConfidenceScore(ctx context.Context, finding agent.Finding, context ConsensusContext) (float64, error)

	// DeduplicateFindings removes duplicate findings using semantic similarity
	DeduplicateFindings(ctx context.Context, findings []agent.Finding) ([]FindingGroup, error)

	// GetStats returns consensus engine statistics
	GetStats() ConsensusStats
}

// ConsensusResult represents the result of consensus analysis
type ConsensusResult struct {
	DeduplicatedFindings []ConsensusFinding `json:"deduplicated_findings"`
	Statistics          ConsensusStats     `json:"statistics"`
	ModelVersion        string             `json:"model_version"`
	ProcessingTime      time.Duration      `json:"processing_time"`
}

// ConsensusFinding represents a finding with consensus information
type ConsensusFinding struct {
	agent.Finding
	ConsensusScore    float64   `json:"consensus_score"`    // 0.0 - 1.0
	AgreementCount    int       `json:"agreement_count"`    // Number of tools that found this
	DisagreementCount int       `json:"disagreement_count"` // Number of tools that disagree
	SupportingTools   []string  `json:"supporting_tools"`   // Tools that found this finding
	ConflictingTools  []string  `json:"conflicting_tools"`  // Tools that disagree on severity/category
	SimilarFindings   []string  `json:"similar_findings"`   // IDs of similar findings that were merged
	FinalSeverity     agent.Severity `json:"final_severity"` // Consensus-determined severity
	FinalCategory     agent.VulnCategory `json:"final_category"` // Consensus-determined category
}

// FindingGroup represents a group of similar findings
type FindingGroup struct {
	ID               string          `json:"id"`
	PrimaryFinding   agent.Finding   `json:"primary_finding"`   // The representative finding
	SimilarFindings  []agent.Finding `json:"similar_findings"`  // Other similar findings
	SimilarityScore  float64         `json:"similarity_score"`  // Average similarity score
	Tools            []string        `json:"tools"`             // All tools that found similar issues
}

// ConsensusContext provides context for confidence scoring
type ConsensusContext struct {
	TotalAgents      int                    `json:"total_agents"`       // Total number of agents run
	AgentReliability map[string]float64     `json:"agent_reliability"`  // Historical reliability per agent
	HistoricalData   []HistoricalFinding    `json:"historical_data"`    // Past findings for learning
	UserFeedback     []UserFeedback         `json:"user_feedback"`      // User feedback on similar findings
}

// HistoricalFinding represents past finding data for learning
type HistoricalFinding struct {
	Finding         agent.Finding `json:"finding"`
	WasFalsePositive bool         `json:"was_false_positive"`
	UserAction      string        `json:"user_action"` // fixed, ignored, false_positive
	Timestamp       time.Time     `json:"timestamp"`
}

// UserFeedback represents user feedback on findings
type UserFeedback struct {
	FindingID   string    `json:"finding_id"`
	UserID      string    `json:"user_id"`
	Action      string    `json:"action"`      // fixed, ignored, false_positive, confirmed
	Comment     string    `json:"comment"`
	Confidence  float64   `json:"confidence"`  // User's confidence in their feedback
	Timestamp   time.Time `json:"timestamp"`
}

// ConsensusStats represents statistics about consensus analysis
type ConsensusStats struct {
	TotalFindings        int                    `json:"total_findings"`
	DeduplicatedFindings int                    `json:"deduplicated_findings"`
	HighConfidenceCount  int                    `json:"high_confidence_count"`
	MediumConfidenceCount int                   `json:"medium_confidence_count"`
	LowConfidenceCount   int                    `json:"low_confidence_count"`
	AverageConfidence    float64                `json:"average_confidence"`
	ProcessingTime       time.Duration          `json:"processing_time"`
	SimilarityMatches    int                    `json:"similarity_matches"`
	ToolAgreementRates   map[string]float64     `json:"tool_agreement_rates"`
	CategoryDistribution map[string]int         `json:"category_distribution"`
	SeverityDistribution map[string]int         `json:"severity_distribution"`
}

// SimilarityConfig configures similarity matching
type SimilarityConfig struct {
	MinSimilarityThreshold float64 `json:"min_similarity_threshold"` // Minimum similarity to consider findings similar
	FilePathWeight         float64 `json:"file_path_weight"`         // Weight for file path similarity
	RuleIDWeight           float64 `json:"rule_id_weight"`           // Weight for rule ID similarity
	MessageWeight          float64 `json:"message_weight"`           // Weight for message similarity
	LocationWeight         float64 `json:"location_weight"`          // Weight for location similarity
}

// DefaultSimilarityConfig returns default similarity configuration
func DefaultSimilarityConfig() SimilarityConfig {
	return SimilarityConfig{
		MinSimilarityThreshold: 0.7,  // Lower threshold for better grouping
		FilePathWeight:         0.3,
		RuleIDWeight:           0.3,
		MessageWeight:          0.3,
		LocationWeight:         0.1,
	}
}

// ConfidenceThresholds defines thresholds for confidence levels
type ConfidenceThresholds struct {
	High   float64 `json:"high"`   // >= 0.95
	Medium float64 `json:"medium"` // >= 0.7
	Low    float64 `json:"low"`    // < 0.7
}

// DefaultConfidenceThresholds returns default confidence thresholds
func DefaultConfidenceThresholds() ConfidenceThresholds {
	return ConfidenceThresholds{
		High:   0.95,
		Medium: 0.7,
		Low:    0.0,
	}
}