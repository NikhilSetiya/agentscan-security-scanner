package consensus

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
)

// ConfidenceCalibrator calibrates confidence scores based on historical accuracy
type ConfidenceCalibrator interface {
	// CalibrateConfidence adjusts a confidence score based on historical accuracy
	CalibrateConfidence(ctx context.Context, rawConfidence float64, context CalibrationContext) (float64, error)
	
	// UpdateCalibration updates calibration data with new feedback
	UpdateCalibration(ctx context.Context, prediction ConfidencePrediction, outcome bool) error
	
	// GetCalibrationCurve returns the calibration curve data
	GetCalibrationCurve(ctx context.Context, context CalibrationContext) (CalibrationCurve, error)
	
	// GetCalibrationStats returns calibration statistics
	GetCalibrationStats(ctx context.Context) CalibrationStats
}

// CalibrationContext provides context for confidence calibration
type CalibrationContext struct {
	Tool       string             `json:"tool"`
	Severity   agent.Severity     `json:"severity"`
	Category   agent.VulnCategory `json:"category"`
	RuleID     string             `json:"rule_id"`
	FilePath   string             `json:"file_path"`
	TimeWindow time.Duration      `json:"time_window"`
}

// ConfidencePrediction represents a confidence prediction that can be validated later
type ConfidencePrediction struct {
	ID               string             `json:"id"`
	RawConfidence    float64            `json:"raw_confidence"`
	CalibratedConfidence float64        `json:"calibrated_confidence"`
	Context          CalibrationContext `json:"context"`
	Timestamp        time.Time          `json:"timestamp"`
	FindingID        string             `json:"finding_id"`
}

// CalibrationCurve represents the relationship between predicted and actual confidence
type CalibrationCurve struct {
	Bins           []CalibrationBin `json:"bins"`
	ReliabilityDiagram []Point      `json:"reliability_diagram"`
	BrierScore     float64          `json:"brier_score"`
	CalibrationError float64        `json:"calibration_error"`
}

// CalibrationBin represents a bin in the calibration curve
type CalibrationBin struct {
	MinConfidence    float64 `json:"min_confidence"`
	MaxConfidence    float64 `json:"max_confidence"`
	MeanConfidence   float64 `json:"mean_confidence"`
	ActualAccuracy   float64 `json:"actual_accuracy"`
	Count            int     `json:"count"`
	CalibrationError float64 `json:"calibration_error"`
}

// Point represents a point in 2D space
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// CalibrationStats represents overall calibration statistics
type CalibrationStats struct {
	TotalPredictions    int     `json:"total_predictions"`
	OverallAccuracy     float64 `json:"overall_accuracy"`
	BrierScore          float64 `json:"brier_score"`
	CalibrationError    float64 `json:"calibration_error"`
	Overconfidence      float64 `json:"overconfidence"`
	Underconfidence     float64 `json:"underconfidence"`
	LastUpdated         time.Time `json:"last_updated"`
}

// PlattCalibrator implements Platt scaling for confidence calibration
type PlattCalibrator struct {
	mu                sync.RWMutex
	calibrationData   map[string][]CalibrationRecord
	sigmoidParams     map[string]SigmoidParams
	minSampleSize     int
	maxAge            time.Duration
	binCount          int
}

// CalibrationRecord represents a single calibration data point
type CalibrationRecord struct {
	RawConfidence        float64            `json:"raw_confidence"`
	CalibratedConfidence float64            `json:"calibrated_confidence"`
	ActualOutcome        bool               `json:"actual_outcome"`
	Context              CalibrationContext `json:"context"`
	Timestamp            time.Time          `json:"timestamp"`
	Weight               float64            `json:"weight"`
}

// SigmoidParams represents parameters for sigmoid calibration
type SigmoidParams struct {
	A           float64   `json:"a"`           // Slope parameter
	B           float64   `json:"b"`           // Intercept parameter
	LastUpdated time.Time `json:"last_updated"`
	SampleCount int       `json:"sample_count"`
}

// NewPlattCalibrator creates a new Platt scaling calibrator
func NewPlattCalibrator() *PlattCalibrator {
	return &PlattCalibrator{
		calibrationData: make(map[string][]CalibrationRecord),
		sigmoidParams:   make(map[string]SigmoidParams),
		minSampleSize:   20,  // Minimum samples needed for calibration
		maxAge:          90 * 24 * time.Hour, // 90 days
		binCount:        10,  // Number of bins for calibration curve
	}
}

// CalibrateConfidence adjusts a confidence score based on historical accuracy
func (pc *PlattCalibrator) CalibrateConfidence(ctx context.Context, rawConfidence float64, context CalibrationContext) (float64, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	// Generate context key for lookup
	contextKey := pc.generateContextKey(context)
	
	// Check if we have calibration parameters for this context
	params, exists := pc.sigmoidParams[contextKey]
	if !exists || params.SampleCount < pc.minSampleSize {
		// Fall back to isotonic regression or simple adjustment
		return pc.simpleCalibration(rawConfidence, context), nil
	}
	
	// Apply Platt scaling: P(y=1|f) = 1 / (1 + exp(A*f + B))
	// where f is the raw confidence score
	logit := params.A*rawConfidence + params.B
	calibratedConfidence := 1.0 / (1.0 + math.Exp(-logit))
	
	// Ensure the result is within bounds
	calibratedConfidence = math.Max(0.01, math.Min(0.99, calibratedConfidence))
	
	return calibratedConfidence, nil
}

// UpdateCalibration updates calibration data with new feedback
func (pc *PlattCalibrator) UpdateCalibration(ctx context.Context, prediction ConfidencePrediction, outcome bool) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	contextKey := pc.generateContextKey(prediction.Context)
	
	// Create calibration record
	record := CalibrationRecord{
		RawConfidence:        prediction.RawConfidence,
		CalibratedConfidence: prediction.CalibratedConfidence,
		ActualOutcome:        outcome,
		Context:              prediction.Context,
		Timestamp:            time.Now(),
		Weight:               1.0, // Could be adjusted based on user confidence
	}
	
	// Add to calibration data
	if _, exists := pc.calibrationData[contextKey]; !exists {
		pc.calibrationData[contextKey] = make([]CalibrationRecord, 0)
	}
	
	pc.calibrationData[contextKey] = append(pc.calibrationData[contextKey], record)
	
	// Clean old data
	pc.cleanOldData(contextKey)
	
	// Recompute calibration parameters if we have enough data
	if len(pc.calibrationData[contextKey]) >= pc.minSampleSize {
		err := pc.recomputeCalibration(contextKey)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// GetCalibrationCurve returns the calibration curve data
func (pc *PlattCalibrator) GetCalibrationCurve(ctx context.Context, context CalibrationContext) (CalibrationCurve, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	contextKey := pc.generateContextKey(context)
	data, exists := pc.calibrationData[contextKey]
	if !exists || len(data) == 0 {
		return CalibrationCurve{}, nil
	}
	
	// Create bins
	bins := make([]CalibrationBin, pc.binCount)
	binSize := 1.0 / float64(pc.binCount)
	
	for i := 0; i < pc.binCount; i++ {
		bins[i] = CalibrationBin{
			MinConfidence: float64(i) * binSize,
			MaxConfidence: float64(i+1) * binSize,
		}
	}
	
	// Populate bins with data
	for _, record := range data {
		binIndex := int(record.CalibratedConfidence * float64(pc.binCount))
		if binIndex >= pc.binCount {
			binIndex = pc.binCount - 1
		}
		
		bins[binIndex].Count++
		bins[binIndex].MeanConfidence += record.CalibratedConfidence
		if record.ActualOutcome {
			bins[binIndex].ActualAccuracy += 1.0
		}
	}
	
	// Calculate final statistics for each bin
	reliabilityPoints := make([]Point, 0)
	totalCalibrationError := 0.0
	totalBrierScore := 0.0
	totalCount := 0
	
	for i := range bins {
		if bins[i].Count > 0 {
			bins[i].MeanConfidence /= float64(bins[i].Count)
			bins[i].ActualAccuracy /= float64(bins[i].Count)
			bins[i].CalibrationError = math.Abs(bins[i].MeanConfidence - bins[i].ActualAccuracy)
			
			reliabilityPoints = append(reliabilityPoints, Point{
				X: bins[i].MeanConfidence,
				Y: bins[i].ActualAccuracy,
			})
			
			totalCalibrationError += bins[i].CalibrationError * float64(bins[i].Count)
			totalCount += bins[i].Count
		}
	}
	
	// Calculate overall metrics
	overallCalibrationError := 0.0
	if totalCount > 0 {
		overallCalibrationError = totalCalibrationError / float64(totalCount)
	}
	
	// Calculate Brier score
	for _, record := range data {
		actualScore := 0.0
		if record.ActualOutcome {
			actualScore = 1.0
		}
		brierContribution := math.Pow(record.CalibratedConfidence-actualScore, 2)
		totalBrierScore += brierContribution
	}
	
	brierScore := 0.0
	if len(data) > 0 {
		brierScore = totalBrierScore / float64(len(data))
	}
	
	return CalibrationCurve{
		Bins:             bins,
		ReliabilityDiagram: reliabilityPoints,
		BrierScore:       brierScore,
		CalibrationError: overallCalibrationError,
	}, nil
}

// GetCalibrationStats returns calibration statistics
func (pc *PlattCalibrator) GetCalibrationStats(ctx context.Context) CalibrationStats {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	totalPredictions := 0
	totalCorrect := 0
	totalBrierScore := 0.0
	totalCalibrationError := 0.0
	overconfidenceSum := 0.0
	underconfidenceSum := 0.0
	
	for _, data := range pc.calibrationData {
		for _, record := range data {
			totalPredictions++
			
			actualScore := 0.0
			if record.ActualOutcome {
				actualScore = 1.0
				totalCorrect++
			}
			
			// Brier score contribution
			brierContribution := math.Pow(record.CalibratedConfidence-actualScore, 2)
			totalBrierScore += brierContribution
			
			// Calibration error contribution
			calibrationError := math.Abs(record.CalibratedConfidence - actualScore)
			totalCalibrationError += calibrationError
			
			// Over/under confidence
			if record.CalibratedConfidence > actualScore {
				overconfidenceSum += record.CalibratedConfidence - actualScore
			} else if record.CalibratedConfidence < actualScore {
				underconfidenceSum += actualScore - record.CalibratedConfidence
			}
		}
	}
	
	stats := CalibrationStats{
		TotalPredictions: totalPredictions,
		LastUpdated:      time.Now(),
	}
	
	if totalPredictions > 0 {
		stats.OverallAccuracy = float64(totalCorrect) / float64(totalPredictions)
		stats.BrierScore = totalBrierScore / float64(totalPredictions)
		stats.CalibrationError = totalCalibrationError / float64(totalPredictions)
		stats.Overconfidence = overconfidenceSum / float64(totalPredictions)
		stats.Underconfidence = underconfidenceSum / float64(totalPredictions)
	}
	
	return stats
}

// generateContextKey generates a key for the calibration context
func (pc *PlattCalibrator) generateContextKey(context CalibrationContext) string {
	// Create a hierarchical key that allows for fallback
	// Most specific: tool + severity + category + rule
	// Less specific: tool + severity + category
	// Least specific: tool + severity
	
	key := context.Tool
	if context.Severity != "" {
		key += "_" + string(context.Severity)
	}
	if context.Category != "" {
		key += "_" + string(context.Category)
	}
	if context.RuleID != "" {
		key += "_" + context.RuleID
	}
	
	return key
}

// simpleCalibration provides a fallback calibration method
func (pc *PlattCalibrator) simpleCalibration(rawConfidence float64, context CalibrationContext) float64 {
	// Look for any available data for this tool
	toolKey := context.Tool
	
	var relevantData []CalibrationRecord
	for key, data := range pc.calibrationData {
		if len(key) >= len(toolKey) && key[:len(toolKey)] == toolKey {
			relevantData = append(relevantData, data...)
		}
	}
	
	if len(relevantData) == 0 {
		// No data available, return raw confidence with slight adjustment
		return rawConfidence * 0.9 // Slightly reduce confidence when uncertain
	}
	
	// Simple isotonic regression approximation
	return pc.isotonicCalibration(rawConfidence, relevantData)
}

// isotonicCalibration performs simple isotonic regression
func (pc *PlattCalibrator) isotonicCalibration(rawConfidence float64, data []CalibrationRecord) float64 {
	if len(data) == 0 {
		return rawConfidence
	}
	
	// Sort data by raw confidence
	sort.Slice(data, func(i, j int) bool {
		return data[i].RawConfidence < data[j].RawConfidence
	})
	
	// Find the appropriate range
	var lowerBound, upperBound CalibrationRecord
	found := false
	
	for i, record := range data {
		if record.RawConfidence >= rawConfidence {
			if i > 0 {
				lowerBound = data[i-1]
				upperBound = record
			} else {
				lowerBound = record
				upperBound = record
			}
			found = true
			break
		}
	}
	
	if !found {
		// Use the last record
		lowerBound = data[len(data)-1]
		upperBound = data[len(data)-1]
	}
	
	// Calculate calibrated confidence using linear interpolation
	if lowerBound.RawConfidence == upperBound.RawConfidence {
		actualOutcome := 0.0
		if lowerBound.ActualOutcome {
			actualOutcome = 1.0
		}
		return actualOutcome
	}
	
	// Linear interpolation
	ratio := (rawConfidence - lowerBound.RawConfidence) / (upperBound.RawConfidence - lowerBound.RawConfidence)
	
	lowerOutcome := 0.0
	if lowerBound.ActualOutcome {
		lowerOutcome = 1.0
	}
	
	upperOutcome := 0.0
	if upperBound.ActualOutcome {
		upperOutcome = 1.0
	}
	
	calibratedConfidence := lowerOutcome + ratio*(upperOutcome-lowerOutcome)
	
	return math.Max(0.01, math.Min(0.99, calibratedConfidence))
}

// recomputeCalibration recomputes Platt scaling parameters
func (pc *PlattCalibrator) recomputeCalibration(contextKey string) error {
	data := pc.calibrationData[contextKey]
	if len(data) < pc.minSampleSize {
		return nil
	}
	
	// Prepare data for Platt scaling
	confidences := make([]float64, len(data))
	outcomes := make([]float64, len(data))
	
	for i, record := range data {
		confidences[i] = record.RawConfidence
		if record.ActualOutcome {
			outcomes[i] = 1.0
		} else {
			outcomes[i] = 0.0
		}
	}
	
	// Fit sigmoid parameters using maximum likelihood estimation
	// This is a simplified version - in practice, you'd use proper optimization
	a, b := pc.fitSigmoid(confidences, outcomes)
	
	pc.sigmoidParams[contextKey] = SigmoidParams{
		A:           a,
		B:           b,
		LastUpdated: time.Now(),
		SampleCount: len(data),
	}
	
	return nil
}

// fitSigmoid fits sigmoid parameters using a simplified method
func (pc *PlattCalibrator) fitSigmoid(confidences, outcomes []float64) (float64, float64) {
	// Simplified sigmoid fitting using least squares
	// In practice, you'd use proper maximum likelihood estimation
	
	n := len(confidences)
	if n == 0 {
		return 1.0, 0.0
	}
	
	// Calculate means
	meanX := 0.0
	meanY := 0.0
	for i := 0; i < n; i++ {
		meanX += confidences[i]
		meanY += outcomes[i]
	}
	meanX /= float64(n)
	meanY /= float64(n)
	
	// Calculate slope (simplified)
	numerator := 0.0
	denominator := 0.0
	
	for i := 0; i < n; i++ {
		numerator += (confidences[i] - meanX) * (outcomes[i] - meanY)
		denominator += (confidences[i] - meanX) * (confidences[i] - meanX)
	}
	
	a := 1.0 // Default slope
	if denominator != 0 {
		a = numerator / denominator
	}
	
	// Calculate intercept
	b := meanY - a*meanX
	
	return a, b
}

// cleanOldData removes calibration data older than maxAge
func (pc *PlattCalibrator) cleanOldData(contextKey string) {
	data := pc.calibrationData[contextKey]
	cutoff := time.Now().Add(-pc.maxAge)
	
	filtered := make([]CalibrationRecord, 0)
	for _, record := range data {
		if record.Timestamp.After(cutoff) {
			filtered = append(filtered, record)
		}
	}
	
	pc.calibrationData[contextKey] = filtered
	
	// Remove context if no data remains
	if len(filtered) == 0 {
		delete(pc.calibrationData, contextKey)
		delete(pc.sigmoidParams, contextKey)
	}
}

// GetCalibrationCurveForTool returns calibration curve for a specific tool
func (pc *PlattCalibrator) GetCalibrationCurveForTool(ctx context.Context, tool string) (CalibrationCurve, error) {
	context := CalibrationContext{
		Tool: tool,
	}
	return pc.GetCalibrationCurve(ctx, context)
}

// GetReliabilityDiagram returns reliability diagram data for visualization
func (pc *PlattCalibrator) GetReliabilityDiagram(ctx context.Context, context CalibrationContext) ([]Point, error) {
	curve, err := pc.GetCalibrationCurve(ctx, context)
	if err != nil {
		return nil, err
	}
	
	return curve.ReliabilityDiagram, nil
}