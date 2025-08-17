package consensus

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

// ToolReliabilityScorer tracks and calculates reliability scores for security tools
type ToolReliabilityScorer interface {
	// UpdateReliability updates reliability based on user feedback
	UpdateReliability(ctx context.Context, tool string, feedback UserFeedback) error
	
	// GetReliability returns the current reliability score for a tool
	GetReliability(ctx context.Context, tool string) float64
	
	// GetAllReliabilities returns reliability scores for all tools
	GetAllReliabilities(ctx context.Context) map[string]float64
	
	// CalculateFalsePositiveRate calculates false positive rate for a tool
	CalculateFalsePositiveRate(ctx context.Context, tool string) float64
	
	// GetToolStats returns detailed statistics for a tool
	GetToolStats(ctx context.Context, tool string) ToolStats
}

// ToolStats represents detailed statistics for a security tool
type ToolStats struct {
	Tool                string    `json:"tool"`
	TotalFindings       int       `json:"total_findings"`
	ConfirmedFindings   int       `json:"confirmed_findings"`
	FalsePositives      int       `json:"false_positives"`
	IgnoredFindings     int       `json:"ignored_findings"`
	FixedFindings       int       `json:"fixed_findings"`
	ReliabilityScore    float64   `json:"reliability_score"`
	FalsePositiveRate   float64   `json:"false_positive_rate"`
	ConfidenceAccuracy  float64   `json:"confidence_accuracy"`
	LastUpdated         time.Time `json:"last_updated"`
	
	// Severity-specific stats
	HighSeverityStats   SeverityStats `json:"high_severity_stats"`
	MediumSeverityStats SeverityStats `json:"medium_severity_stats"`
	LowSeverityStats    SeverityStats `json:"low_severity_stats"`
	
	// Category-specific stats
	CategoryStats       map[string]SeverityStats `json:"category_stats"`
}

// SeverityStats represents statistics for a specific severity level
type SeverityStats struct {
	Total          int     `json:"total"`
	Confirmed      int     `json:"confirmed"`
	FalsePositives int     `json:"false_positives"`
	Accuracy       float64 `json:"accuracy"`
}

// ReliabilityTracker tracks tool reliability over time
type ReliabilityTracker struct {
	mu                sync.RWMutex
	toolStats         map[string]*ToolStatsInternal
	decayFactor       float64   // Factor for time-based decay of old data
	minSampleSize     int       // Minimum samples needed for reliable scoring
	confidenceWindow  time.Duration // Time window for confidence calculations
}

// ToolStatsInternal represents internal tool statistics with more detailed tracking
type ToolStatsInternal struct {
	Tool              string
	FeedbackHistory   []TimestampedFeedback
	ConfidenceHistory []ConfidenceRecord
	LastUpdated       time.Time
	
	// Cached calculations
	cachedReliability    float64
	cachedFPRate         float64
	cacheValidUntil      time.Time
}

// TimestampedFeedback represents user feedback with timestamp
type TimestampedFeedback struct {
	Feedback  UserFeedback
	Timestamp time.Time
	Severity  agent.Severity
	Category  agent.VulnCategory
	Confidence float64
}

// ConfidenceRecord tracks confidence vs actual outcome
type ConfidenceRecord struct {
	PredictedConfidence float64
	ActualOutcome       bool // true if confirmed, false if false positive
	Timestamp          time.Time
	Severity           agent.Severity
	Category           agent.VulnCategory
}

// NewReliabilityTracker creates a new tool reliability tracker
func NewReliabilityTracker() *ReliabilityTracker {
	return &ReliabilityTracker{
		toolStats:        make(map[string]*ToolStatsInternal),
		decayFactor:      0.95, // 5% decay per time period
		minSampleSize:    10,   // Need at least 10 samples for reliable scoring
		confidenceWindow: 30 * 24 * time.Hour, // 30 days
	}
}

// UpdateReliability updates tool reliability based on user feedback
func (rt *ReliabilityTracker) UpdateReliability(ctx context.Context, tool string, feedback UserFeedback) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	// Get or create tool stats
	stats, exists := rt.toolStats[tool]
	if !exists {
		stats = &ToolStatsInternal{
			Tool:              tool,
			FeedbackHistory:   make([]TimestampedFeedback, 0),
			ConfidenceHistory: make([]ConfidenceRecord, 0),
		}
		rt.toolStats[tool] = stats
	}
	
	// Add feedback to history
	timestampedFeedback := TimestampedFeedback{
		Feedback:  feedback,
		Timestamp: feedback.Timestamp,
		Confidence: feedback.Confidence,
	}
	
	stats.FeedbackHistory = append(stats.FeedbackHistory, timestampedFeedback)
	stats.LastUpdated = time.Now()
	
	// Invalidate cache
	stats.cacheValidUntil = time.Time{}
	
	// Trim old feedback (keep only last 1000 entries)
	if len(stats.FeedbackHistory) > 1000 {
		stats.FeedbackHistory = stats.FeedbackHistory[len(stats.FeedbackHistory)-1000:]
	}
	
	return nil
}

// GetReliability returns the current reliability score for a tool
func (rt *ReliabilityTracker) GetReliability(ctx context.Context, tool string) float64 {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	
	stats, exists := rt.toolStats[tool]
	if !exists {
		return 0.5 // Default reliability for unknown tools
	}
	
	// Check if cached value is still valid
	if time.Now().Before(stats.cacheValidUntil) {
		return stats.cachedReliability
	}
	
	// Calculate reliability
	reliability := rt.calculateReliability(stats)
	
	// Update cache
	stats.cachedReliability = reliability
	stats.cacheValidUntil = time.Now().Add(time.Hour) // Cache for 1 hour
	
	return reliability
}

// GetAllReliabilities returns reliability scores for all tools
func (rt *ReliabilityTracker) GetAllReliabilities(ctx context.Context) map[string]float64 {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	
	reliabilities := make(map[string]float64)
	
	for tool := range rt.toolStats {
		reliabilities[tool] = rt.GetReliability(ctx, tool)
	}
	
	return reliabilities
}

// CalculateFalsePositiveRate calculates false positive rate for a tool
func (rt *ReliabilityTracker) CalculateFalsePositiveRate(ctx context.Context, tool string) float64 {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	
	stats, exists := rt.toolStats[tool]
	if !exists {
		return 0.0 // No data available
	}
	
	// Check if cached value is still valid
	if time.Now().Before(stats.cacheValidUntil) {
		return stats.cachedFPRate
	}
	
	// Calculate false positive rate
	fpRate := rt.calculateFalsePositiveRate(stats)
	
	// Update cache
	stats.cachedFPRate = fpRate
	
	return fpRate
}

// GetToolStats returns detailed statistics for a tool
func (rt *ReliabilityTracker) GetToolStats(ctx context.Context, tool string) ToolStats {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	
	stats, exists := rt.toolStats[tool]
	if !exists {
		return ToolStats{
			Tool:             tool,
			ReliabilityScore: 0.5,
			LastUpdated:      time.Time{},
		}
	}
	
	return rt.buildToolStats(stats)
}

// calculateReliability calculates the reliability score for a tool
func (rt *ReliabilityTracker) calculateReliability(stats *ToolStatsInternal) float64 {
	if len(stats.FeedbackHistory) < rt.minSampleSize {
		return 0.5 // Default reliability for insufficient data
	}
	
	// Apply time-based decay to give more weight to recent feedback
	now := time.Now()
	totalWeight := 0.0
	weightedScore := 0.0
	
	for _, feedback := range stats.FeedbackHistory {
		// Calculate time-based weight (more recent = higher weight)
		daysSince := now.Sub(feedback.Timestamp).Hours() / 24
		weight := math.Pow(rt.decayFactor, daysSince)
		
		// Calculate feedback score
		var score float64
		switch feedback.Feedback.Action {
		case "confirmed", "fixed":
			score = 1.0 // Positive feedback
		case "false_positive":
			score = 0.0 // Negative feedback
		case "ignored":
			score = 0.3 // Neutral/slightly negative
		default:
			score = 0.5 // Unknown action
		}
		
		// Weight by user confidence
		score *= feedback.Feedback.Confidence
		
		weightedScore += score * weight
		totalWeight += weight
	}
	
	if totalWeight == 0 {
		return 0.5
	}
	
	reliability := weightedScore / totalWeight
	
	// Apply confidence adjustment based on sample size
	confidenceAdjustment := math.Min(1.0, float64(len(stats.FeedbackHistory))/float64(rt.minSampleSize*2))
	reliability = 0.5 + (reliability-0.5)*confidenceAdjustment
	
	return math.Max(0.0, math.Min(1.0, reliability))
}

// calculateFalsePositiveRate calculates the false positive rate for a tool
func (rt *ReliabilityTracker) calculateFalsePositiveRate(stats *ToolStatsInternal) float64 {
	if len(stats.FeedbackHistory) == 0 {
		return 0.0
	}
	
	falsePositives := 0
	total := 0
	now := time.Now()
	
	for _, feedback := range stats.FeedbackHistory {
		// Only consider recent feedback (within confidence window)
		if now.Sub(feedback.Timestamp) > rt.confidenceWindow {
			continue
		}
		
		total++
		if feedback.Feedback.Action == "false_positive" {
			falsePositives++
		}
	}
	
	if total == 0 {
		return 0.0
	}
	
	return float64(falsePositives) / float64(total)
}

// buildToolStats builds comprehensive tool statistics
func (rt *ReliabilityTracker) buildToolStats(stats *ToolStatsInternal) ToolStats {
	toolStats := ToolStats{
		Tool:               stats.Tool,
		ReliabilityScore:   rt.calculateReliability(stats),
		FalsePositiveRate:  rt.calculateFalsePositiveRate(stats),
		LastUpdated:        stats.LastUpdated,
		CategoryStats:      make(map[string]SeverityStats),
	}
	
	// Count different types of feedback
	severityCounts := map[agent.Severity]map[string]int{
		agent.SeverityHigh:   {"total": 0, "confirmed": 0, "false_positive": 0},
		agent.SeverityMedium: {"total": 0, "confirmed": 0, "false_positive": 0},
		agent.SeverityLow:    {"total": 0, "confirmed": 0, "false_positive": 0},
	}
	
	categoryCounts := make(map[agent.VulnCategory]map[string]int)
	
	for _, feedback := range stats.FeedbackHistory {
		toolStats.TotalFindings++
		
		// Count by action
		switch feedback.Feedback.Action {
		case "confirmed":
			toolStats.ConfirmedFindings++
		case "false_positive":
			toolStats.FalsePositives++
		case "ignored":
			toolStats.IgnoredFindings++
		case "fixed":
			toolStats.FixedFindings++
		}
		
		// Count by severity
		if counts, exists := severityCounts[feedback.Severity]; exists {
			counts["total"]++
			if feedback.Feedback.Action == "confirmed" || feedback.Feedback.Action == "fixed" {
				counts["confirmed"]++
			} else if feedback.Feedback.Action == "false_positive" {
				counts["false_positive"]++
			}
		}
		
		// Count by category
		if _, exists := categoryCounts[feedback.Category]; !exists {
			categoryCounts[feedback.Category] = map[string]int{"total": 0, "confirmed": 0, "false_positive": 0}
		}
		categoryCounts[feedback.Category]["total"]++
		if feedback.Feedback.Action == "confirmed" || feedback.Feedback.Action == "fixed" {
			categoryCounts[feedback.Category]["confirmed"]++
		} else if feedback.Feedback.Action == "false_positive" {
			categoryCounts[feedback.Category]["false_positive"]++
		}
	}
	
	// Calculate severity-specific stats
	toolStats.HighSeverityStats = rt.calculateSeverityStats(severityCounts[agent.SeverityHigh])
	toolStats.MediumSeverityStats = rt.calculateSeverityStats(severityCounts[agent.SeverityMedium])
	toolStats.LowSeverityStats = rt.calculateSeverityStats(severityCounts[agent.SeverityLow])
	
	// Calculate category-specific stats
	for category, counts := range categoryCounts {
		toolStats.CategoryStats[string(category)] = rt.calculateSeverityStats(counts)
	}
	
	// Calculate confidence accuracy
	toolStats.ConfidenceAccuracy = rt.calculateConfidenceAccuracy(stats)
	
	return toolStats
}

// calculateSeverityStats calculates accuracy stats for a severity level
func (rt *ReliabilityTracker) calculateSeverityStats(counts map[string]int) SeverityStats {
	total := counts["total"]
	confirmed := counts["confirmed"]
	falsePositives := counts["false_positive"]
	
	var accuracy float64
	if total > 0 {
		accuracy = float64(confirmed) / float64(total)
	}
	
	return SeverityStats{
		Total:          total,
		Confirmed:      confirmed,
		FalsePositives: falsePositives,
		Accuracy:       accuracy,
	}
}

// calculateConfidenceAccuracy calculates how well the tool's confidence correlates with actual outcomes
func (rt *ReliabilityTracker) calculateConfidenceAccuracy(stats *ToolStatsInternal) float64 {
	if len(stats.ConfidenceHistory) == 0 {
		return 0.5 // Default accuracy
	}
	
	// Calculate correlation between predicted confidence and actual outcomes
	totalError := 0.0
	count := 0
	
	for _, record := range stats.ConfidenceHistory {
		actualScore := 0.0
		if record.ActualOutcome {
			actualScore = 1.0
		}
		
		error := math.Abs(record.PredictedConfidence - actualScore)
		totalError += error
		count++
	}
	
	if count == 0 {
		return 0.5
	}
	
	// Convert average error to accuracy
	avgError := totalError / float64(count)
	accuracy := 1.0 - avgError
	
	return math.Max(0.0, math.Min(1.0, accuracy))
}

// AddConfidenceRecord adds a confidence record for accuracy tracking
func (rt *ReliabilityTracker) AddConfidenceRecord(tool string, predictedConfidence float64, actualOutcome bool, severity agent.Severity, category agent.VulnCategory) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	stats, exists := rt.toolStats[tool]
	if !exists {
		stats = &ToolStatsInternal{
			Tool:              tool,
			FeedbackHistory:   make([]TimestampedFeedback, 0),
			ConfidenceHistory: make([]ConfidenceRecord, 0),
		}
		rt.toolStats[tool] = stats
	}
	
	record := ConfidenceRecord{
		PredictedConfidence: predictedConfidence,
		ActualOutcome:       actualOutcome,
		Timestamp:          time.Now(),
		Severity:           severity,
		Category:           category,
	}
	
	stats.ConfidenceHistory = append(stats.ConfidenceHistory, record)
	
	// Trim old records (keep only last 1000 entries)
	if len(stats.ConfidenceHistory) > 1000 {
		stats.ConfidenceHistory = stats.ConfidenceHistory[len(stats.ConfidenceHistory)-1000:]
	}
}

// GetReliabilityTrend returns reliability trend over time
func (rt *ReliabilityTracker) GetReliabilityTrend(ctx context.Context, tool string, days int) []ReliabilityPoint {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	
	stats, exists := rt.toolStats[tool]
	if !exists {
		return []ReliabilityPoint{}
	}
	
	// Calculate reliability for each day
	now := time.Now()
	points := make([]ReliabilityPoint, days)
	
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)
		reliability := rt.calculateReliabilityForDate(stats, date)
		
		points[days-1-i] = ReliabilityPoint{
			Date:        date,
			Reliability: reliability,
		}
	}
	
	return points
}

// ReliabilityPoint represents a reliability score at a specific point in time
type ReliabilityPoint struct {
	Date        time.Time `json:"date"`
	Reliability float64   `json:"reliability"`
}

// calculateReliabilityForDate calculates reliability up to a specific date
func (rt *ReliabilityTracker) calculateReliabilityForDate(stats *ToolStatsInternal, date time.Time) float64 {
	relevantFeedback := make([]TimestampedFeedback, 0)
	
	// Filter feedback up to the specified date
	for _, feedback := range stats.FeedbackHistory {
		if feedback.Timestamp.Before(date) || feedback.Timestamp.Equal(date) {
			relevantFeedback = append(relevantFeedback, feedback)
		}
	}
	
	if len(relevantFeedback) < rt.minSampleSize {
		return 0.5 // Default reliability
	}
	
	// Calculate reliability using the filtered feedback
	totalWeight := 0.0
	weightedScore := 0.0
	
	for _, feedback := range relevantFeedback {
		daysSince := date.Sub(feedback.Timestamp).Hours() / 24
		weight := math.Pow(rt.decayFactor, daysSince)
		
		var score float64
		switch feedback.Feedback.Action {
		case "confirmed", "fixed":
			score = 1.0
		case "false_positive":
			score = 0.0
		case "ignored":
			score = 0.3
		default:
			score = 0.5
		}
		
		score *= feedback.Feedback.Confidence
		
		weightedScore += score * weight
		totalWeight += weight
	}
	
	if totalWeight == 0 {
		return 0.5
	}
	
	reliability := weightedScore / totalWeight
	confidenceAdjustment := math.Min(1.0, float64(len(relevantFeedback))/float64(rt.minSampleSize*2))
	reliability = 0.5 + (reliability-0.5)*confidenceAdjustment
	
	return math.Max(0.0, math.Min(1.0, reliability))
}