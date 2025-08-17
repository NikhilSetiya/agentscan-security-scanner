package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

// MLModel defines the interface for machine learning models used in consensus scoring
type MLModel interface {
	// Predict calculates an improved consensus score using ML
	Predict(ctx context.Context, features MLFeatures) (float64, error)
	
	// Train updates the model with new training data
	Train(ctx context.Context, trainingData []TrainingExample) error
	
	// GetModelInfo returns information about the current model
	GetModelInfo() ModelInfo
	
	// SaveModel persists the model to storage
	SaveModel(ctx context.Context, path string) error
	
	// LoadModel loads a model from storage
	LoadModel(ctx context.Context, path string) error
}

// MLFeatures represents features extracted from findings for ML prediction
type MLFeatures struct {
	// Basic finding features
	ToolCount           int     `json:"tool_count"`           // Number of tools that found this issue
	AverageConfidence   float64 `json:"average_confidence"`   // Average confidence from all tools
	MaxConfidence       float64 `json:"max_confidence"`       // Maximum confidence from any tool
	MinConfidence       float64 `json:"min_confidence"`       // Minimum confidence from any tool
	ConfidenceVariance  float64 `json:"confidence_variance"`  // Variance in confidence scores
	
	// Tool reliability features
	WeightedReliability float64 `json:"weighted_reliability"` // Weighted average of tool reliability scores
	MaxReliability      float64 `json:"max_reliability"`      // Maximum reliability of any tool
	MinReliability      float64 `json:"min_reliability"`      // Minimum reliability of any tool
	
	// Severity and category features
	SeverityConsensus   float64 `json:"severity_consensus"`   // Agreement on severity (0-1)
	CategoryConsensus   float64 `json:"category_consensus"`   // Agreement on category (0-1)
	HighSeverityRatio   float64 `json:"high_severity_ratio"`  // Ratio of tools reporting high severity
	
	// Historical features
	HistoricalAccuracy  float64 `json:"historical_accuracy"`  // Historical accuracy for similar findings
	FalsePositiveRate   float64 `json:"false_positive_rate"`  // Historical false positive rate
	SimilarFindingCount int     `json:"similar_finding_count"` // Number of similar historical findings
	
	// Rule and file features
	RuleReliability     float64 `json:"rule_reliability"`     // Reliability of the specific rule
	FileRiskScore       float64 `json:"file_risk_score"`      // Risk score of the file
	LineComplexity      float64 `json:"line_complexity"`      // Complexity of the code line
	
	// Context features
	ProjectMaturity     float64 `json:"project_maturity"`     // Maturity score of the project
	RecentChanges       int     `json:"recent_changes"`       // Number of recent changes to the file
	TestCoverage        float64 `json:"test_coverage"`        // Test coverage of the file
}

// TrainingExample represents a training example for the ML model
type TrainingExample struct {
	Features    MLFeatures `json:"features"`
	TrueLabel   float64    `json:"true_label"`   // True consensus score (0-1)
	UserAction  string     `json:"user_action"`  // User action taken (fixed, ignored, false_positive)
	Confidence  float64    `json:"confidence"`   // User confidence in their action
	Timestamp   time.Time  `json:"timestamp"`
}

// ModelInfo contains information about the ML model
type ModelInfo struct {
	Version         string    `json:"version"`
	TrainingSize    int       `json:"training_size"`
	Accuracy        float64   `json:"accuracy"`
	LastTrained     time.Time `json:"last_trained"`
	FeatureWeights  map[string]float64 `json:"feature_weights"`
	ModelType       string    `json:"model_type"`
}

// LinearRegressionModel implements a simple linear regression model for consensus scoring
type LinearRegressionModel struct {
	weights      map[string]float64
	bias         float64
	version      string
	trainingSize int
	accuracy     float64
	lastTrained  time.Time
	learningRate float64
	regularization float64
}

// NewLinearRegressionModel creates a new linear regression model
func NewLinearRegressionModel() *LinearRegressionModel {
	return &LinearRegressionModel{
		weights:        make(map[string]float64),
		bias:           0.5, // Start with neutral bias
		version:        "1.0.0",
		learningRate:   0.01,
		regularization: 0.001,
	}
}

// Predict calculates consensus score using the trained linear model
func (m *LinearRegressionModel) Predict(ctx context.Context, features MLFeatures) (float64, error) {
	if len(m.weights) == 0 {
		// If model is not trained, use simple heuristic
		return m.simpleHeuristic(features), nil
	}
	
	// Convert features to map for easier processing
	featureMap := m.featuresToMap(features)
	
	// Calculate weighted sum
	score := m.bias
	for featureName, featureValue := range featureMap {
		if weight, exists := m.weights[featureName]; exists {
			score += weight * featureValue
		}
	}
	
	// Apply sigmoid to ensure output is between 0 and 1
	score = 1.0 / (1.0 + math.Exp(-score))
	
	// Ensure score is within bounds
	if score < 0.0 {
		score = 0.0
	} else if score > 1.0 {
		score = 1.0
	}
	
	return score, nil
}

// Train updates the model using gradient descent
func (m *LinearRegressionModel) Train(ctx context.Context, trainingData []TrainingExample) error {
	if len(trainingData) == 0 {
		return fmt.Errorf("no training data provided")
	}
	
	// Initialize weights if not already done
	if len(m.weights) == 0 {
		m.initializeWeights(trainingData[0].Features)
	}
	
	// Perform multiple epochs of training
	epochs := 100
	for epoch := 0; epoch < epochs; epoch++ {
		totalLoss := 0.0
		
		for _, example := range trainingData {
			// Forward pass
			predicted, err := m.Predict(ctx, example.Features)
			if err != nil {
				return fmt.Errorf("prediction error during training: %w", err)
			}
			
			// Calculate loss (mean squared error)
			loss := math.Pow(predicted-example.TrueLabel, 2)
			totalLoss += loss
			
			// Backward pass - calculate gradients
			error := predicted - example.TrueLabel
			featureMap := m.featuresToMap(example.Features)
			
			// Update weights
			for featureName, featureValue := range featureMap {
				gradient := error * featureValue
				// Add L2 regularization
				gradient += m.regularization * m.weights[featureName]
				m.weights[featureName] -= m.learningRate * gradient
			}
			
			// Update bias
			m.bias -= m.learningRate * error
		}
		
		// Early stopping if loss is very small
		avgLoss := totalLoss / float64(len(trainingData))
		if avgLoss < 0.001 {
			break
		}
	}
	
	// Update model metadata
	m.trainingSize = len(trainingData)
	m.lastTrained = time.Now()
	
	// Calculate accuracy on training data
	m.accuracy = m.calculateAccuracy(ctx, trainingData)
	
	return nil
}

// GetModelInfo returns information about the current model
func (m *LinearRegressionModel) GetModelInfo() ModelInfo {
	return ModelInfo{
		Version:        m.version,
		TrainingSize:   m.trainingSize,
		Accuracy:       m.accuracy,
		LastTrained:    m.lastTrained,
		FeatureWeights: m.weights,
		ModelType:      "linear_regression",
	}
}

// SaveModel persists the model to storage (simplified JSON serialization)
func (m *LinearRegressionModel) SaveModel(ctx context.Context, path string) error {
	modelData := map[string]interface{}{
		"weights":        m.weights,
		"bias":           m.bias,
		"version":        m.version,
		"training_size":  m.trainingSize,
		"accuracy":       m.accuracy,
		"last_trained":   m.lastTrained,
		"learning_rate":  m.learningRate,
		"regularization": m.regularization,
	}
	
	// In a real implementation, this would save to a file or database
	// For now, we'll just serialize to JSON to validate the structure
	_, err := json.Marshal(modelData)
	if err != nil {
		return fmt.Errorf("failed to serialize model: %w", err)
	}
	
	return nil
}

// LoadModel loads a model from storage
func (m *LinearRegressionModel) LoadModel(ctx context.Context, path string) error {
	// In a real implementation, this would load from a file or database
	// For now, we'll initialize with default values
	m.initializeWeights(MLFeatures{})
	m.lastTrained = time.Now()
	return nil
}

// initializeWeights initializes model weights with small random values
func (m *LinearRegressionModel) initializeWeights(features MLFeatures) {
	featureMap := m.featuresToMap(features)
	
	for featureName := range featureMap {
		// Initialize with small random weights
		m.weights[featureName] = (math.Mod(float64(time.Now().UnixNano()), 1000) - 500) / 10000.0
	}
}

// featuresToMap converts MLFeatures struct to map for easier processing
func (m *LinearRegressionModel) featuresToMap(features MLFeatures) map[string]float64 {
	return map[string]float64{
		"tool_count":            float64(features.ToolCount),
		"average_confidence":    features.AverageConfidence,
		"max_confidence":        features.MaxConfidence,
		"min_confidence":        features.MinConfidence,
		"confidence_variance":   features.ConfidenceVariance,
		"weighted_reliability":  features.WeightedReliability,
		"max_reliability":       features.MaxReliability,
		"min_reliability":       features.MinReliability,
		"severity_consensus":    features.SeverityConsensus,
		"category_consensus":    features.CategoryConsensus,
		"high_severity_ratio":   features.HighSeverityRatio,
		"historical_accuracy":   features.HistoricalAccuracy,
		"false_positive_rate":   features.FalsePositiveRate,
		"similar_finding_count": float64(features.SimilarFindingCount),
		"rule_reliability":      features.RuleReliability,
		"file_risk_score":       features.FileRiskScore,
		"line_complexity":       features.LineComplexity,
		"project_maturity":      features.ProjectMaturity,
		"recent_changes":        float64(features.RecentChanges),
		"test_coverage":         features.TestCoverage,
	}
}

// simpleHeuristic provides a fallback scoring method when model is not trained
func (m *LinearRegressionModel) simpleHeuristic(features MLFeatures) float64 {
	score := 0.5 // Base score
	
	// Tool count factor (more tools = higher confidence)
	if features.ToolCount >= 3 {
		score += 0.3
	} else if features.ToolCount >= 2 {
		score += 0.15
	}
	
	// Confidence factor
	score += features.AverageConfidence * 0.2
	
	// Reliability factor
	score += features.WeightedReliability * 0.15
	
	// Consensus factors
	score += features.SeverityConsensus * 0.1
	score += features.CategoryConsensus * 0.1
	
	// Historical accuracy factor
	if features.HistoricalAccuracy > 0 {
		score += (features.HistoricalAccuracy - 0.5) * 0.2
	}
	
	// False positive penalty
	score -= features.FalsePositiveRate * 0.3
	
	// Ensure score is within bounds
	if score < 0.0 {
		score = 0.0
	} else if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// calculateAccuracy calculates model accuracy on given data
func (m *LinearRegressionModel) calculateAccuracy(ctx context.Context, data []TrainingExample) float64 {
	if len(data) == 0 {
		return 0.0
	}
	
	totalError := 0.0
	for _, example := range data {
		predicted, err := m.Predict(ctx, example.Features)
		if err != nil {
			continue
		}
		
		// Calculate absolute error
		error := math.Abs(predicted - example.TrueLabel)
		totalError += error
	}
	
	// Convert average error to accuracy (1 - normalized error)
	avgError := totalError / float64(len(data))
	accuracy := 1.0 - avgError
	
	if accuracy < 0.0 {
		accuracy = 0.0
	}
	
	return accuracy
}

// FeatureExtractor extracts ML features from findings and context
type FeatureExtractor struct {
	toolReliability map[string]float64
	ruleReliability map[string]float64
	fileRiskScores  map[string]float64
}

// NewFeatureExtractor creates a new feature extractor
func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{
		toolReliability: make(map[string]float64),
		ruleReliability: make(map[string]float64),
		fileRiskScores:  make(map[string]float64),
	}
}

// ExtractFeatures extracts ML features from a finding group and context
func (fe *FeatureExtractor) ExtractFeatures(
	group FindingGroup,
	context ConsensusContext,
	toolReliability map[string]float64,
) MLFeatures {
	allFindings := append([]agent.Finding{group.PrimaryFinding}, group.SimilarFindings...)
	
	features := MLFeatures{
		ToolCount: len(group.Tools),
	}
	
	// Calculate confidence statistics
	confidences := make([]float64, len(allFindings))
	for i, finding := range allFindings {
		confidences[i] = finding.Confidence
	}
	
	if len(confidences) > 0 {
		features.AverageConfidence = fe.calculateMean(confidences)
		features.MaxConfidence = fe.calculateMax(confidences)
		features.MinConfidence = fe.calculateMin(confidences)
		features.ConfidenceVariance = fe.calculateVariance(confidences)
	}
	
	// Calculate reliability statistics
	reliabilities := make([]float64, len(group.Tools))
	for i, tool := range group.Tools {
		if reliability, exists := toolReliability[tool]; exists {
			reliabilities[i] = reliability
		} else {
			reliabilities[i] = 0.5 // Default reliability
		}
	}
	
	if len(reliabilities) > 0 {
		features.WeightedReliability = fe.calculateMean(reliabilities)
		features.MaxReliability = fe.calculateMax(reliabilities)
		features.MinReliability = fe.calculateMin(reliabilities)
	}
	
	// Calculate consensus features
	features.SeverityConsensus = fe.calculateSeverityConsensus(allFindings)
	features.CategoryConsensus = fe.calculateCategoryConsensus(allFindings)
	features.HighSeverityRatio = fe.calculateHighSeverityRatio(allFindings)
	
	// Calculate historical features
	if len(context.HistoricalData) > 0 {
		features.HistoricalAccuracy = fe.calculateHistoricalAccuracy(group.PrimaryFinding, context.HistoricalData)
		features.FalsePositiveRate = fe.calculateFalsePositiveRate(group.PrimaryFinding, context.HistoricalData)
		features.SimilarFindingCount = fe.countSimilarFindings(group.PrimaryFinding, context.HistoricalData)
	}
	
	// Calculate rule and file features
	features.RuleReliability = fe.getRuleReliability(group.PrimaryFinding.RuleID)
	features.FileRiskScore = fe.getFileRiskScore(group.PrimaryFinding.File)
	features.LineComplexity = fe.calculateLineComplexity(group.PrimaryFinding)
	
	// Set default values for context features (would be calculated from project data)
	features.ProjectMaturity = 0.5
	features.RecentChanges = 0
	features.TestCoverage = 0.5
	
	return features
}

// Helper methods for feature calculation

func (fe *FeatureExtractor) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (fe *FeatureExtractor) calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func (fe *FeatureExtractor) calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func (fe *FeatureExtractor) calculateVariance(values []float64) float64 {
	if len(values) <= 1 {
		return 0.0
	}
	
	mean := fe.calculateMean(values)
	sumSquaredDiff := 0.0
	
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	
	return sumSquaredDiff / float64(len(values)-1)
}

func (fe *FeatureExtractor) calculateSeverityConsensus(findings []agent.Finding) float64 {
	if len(findings) <= 1 {
		return 1.0
	}
	
	severityCounts := make(map[agent.Severity]int)
	for _, finding := range findings {
		severityCounts[finding.Severity]++
	}
	
	// Find the most common severity
	maxCount := 0
	for _, count := range severityCounts {
		if count > maxCount {
			maxCount = count
		}
	}
	
	return float64(maxCount) / float64(len(findings))
}

func (fe *FeatureExtractor) calculateCategoryConsensus(findings []agent.Finding) float64 {
	if len(findings) <= 1 {
		return 1.0
	}
	
	categoryCounts := make(map[agent.VulnCategory]int)
	for _, finding := range findings {
		categoryCounts[finding.Category]++
	}
	
	// Find the most common category
	maxCount := 0
	for _, count := range categoryCounts {
		if count > maxCount {
			maxCount = count
		}
	}
	
	return float64(maxCount) / float64(len(findings))
}

func (fe *FeatureExtractor) calculateHighSeverityRatio(findings []agent.Finding) float64 {
	if len(findings) == 0 {
		return 0.0
	}
	
	highSeverityCount := 0
	for _, finding := range findings {
		if finding.Severity == agent.SeverityHigh {
			highSeverityCount++
		}
	}
	
	return float64(highSeverityCount) / float64(len(findings))
}

func (fe *FeatureExtractor) calculateHistoricalAccuracy(finding agent.Finding, historical []HistoricalFinding) float64 {
	similarFindings := fe.findSimilarHistoricalFindings(finding, historical)
	if len(similarFindings) == 0 {
		return 0.5 // Default accuracy
	}
	
	correctCount := 0
	for _, hist := range similarFindings {
		// Consider it correct if it wasn't a false positive
		if !hist.WasFalsePositive {
			correctCount++
		}
	}
	
	return float64(correctCount) / float64(len(similarFindings))
}

func (fe *FeatureExtractor) calculateFalsePositiveRate(finding agent.Finding, historical []HistoricalFinding) float64 {
	similarFindings := fe.findSimilarHistoricalFindings(finding, historical)
	if len(similarFindings) == 0 {
		return 0.0 // No historical data
	}
	
	falsePositiveCount := 0
	for _, hist := range similarFindings {
		if hist.WasFalsePositive {
			falsePositiveCount++
		}
	}
	
	return float64(falsePositiveCount) / float64(len(similarFindings))
}

func (fe *FeatureExtractor) countSimilarFindings(finding agent.Finding, historical []HistoricalFinding) int {
	return len(fe.findSimilarHistoricalFindings(finding, historical))
}

func (fe *FeatureExtractor) findSimilarHistoricalFindings(finding agent.Finding, historical []HistoricalFinding) []HistoricalFinding {
	var similar []HistoricalFinding
	
	for _, hist := range historical {
		// Simple similarity check based on rule ID and file
		if hist.Finding.RuleID == finding.RuleID && hist.Finding.File == finding.File {
			similar = append(similar, hist)
		}
	}
	
	return similar
}

func (fe *FeatureExtractor) getRuleReliability(ruleID string) float64 {
	if reliability, exists := fe.ruleReliability[ruleID]; exists {
		return reliability
	}
	return 0.5 // Default reliability
}

func (fe *FeatureExtractor) getFileRiskScore(filePath string) float64 {
	if score, exists := fe.fileRiskScores[filePath]; exists {
		return score
	}
	return 0.5 // Default risk score
}

func (fe *FeatureExtractor) calculateLineComplexity(finding agent.Finding) float64 {
	// Simple complexity estimation based on code snippet length
	if finding.Code != "" {
		length := len(finding.Code)
		// Normalize to 0-1 range (assuming max complexity at 200 characters)
		complexity := float64(length) / 200.0
		if complexity > 1.0 {
			complexity = 1.0
		}
		return complexity
	}
	return 0.5 // Default complexity
}

// UpdateToolReliability updates the reliability score for a tool
func (fe *FeatureExtractor) UpdateToolReliability(tool string, reliability float64) {
	fe.toolReliability[tool] = reliability
}

// UpdateRuleReliability updates the reliability score for a rule
func (fe *FeatureExtractor) UpdateRuleReliability(ruleID string, reliability float64) {
	fe.ruleReliability[ruleID] = reliability
}

// UpdateFileRiskScore updates the risk score for a file
func (fe *FeatureExtractor) UpdateFileRiskScore(filePath string, riskScore float64) {
	fe.fileRiskScores[filePath] = riskScore
}