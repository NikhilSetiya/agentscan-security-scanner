package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// FunnelEvent represents a user action in the conversion funnel
type FunnelEvent struct {
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id"`
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Properties map[string]interface{} `json:"properties"`
	Source     string                 `json:"source"`     // github, vscode, web, api
	Campaign   string                 `json:"campaign"`   // beta, marketplace, organic
}

// FunnelStage represents a stage in the conversion funnel
type FunnelStage struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

// FunnelMetrics represents conversion metrics for a funnel stage
type FunnelMetrics struct {
	Stage           string    `json:"stage"`
	TotalUsers      int       `json:"total_users"`
	ConvertedUsers  int       `json:"converted_users"`
	ConversionRate  float64   `json:"conversion_rate"`
	DropoffRate     float64   `json:"dropoff_rate"`
	AverageTime     time.Duration `json:"average_time"`
	LastUpdated     time.Time `json:"last_updated"`
}

// FunnelTracker handles conversion funnel tracking and analytics
type FunnelTracker struct {
	redis  *redis.Client
	stages []FunnelStage
}

// NewFunnelTracker creates a new funnel tracker
func NewFunnelTracker(redisClient *redis.Client) *FunnelTracker {
	stages := []FunnelStage{
		{Name: "landing", Description: "User visits landing page", Order: 1},
		{Name: "signup_started", Description: "User starts signup process", Order: 2},
		{Name: "email_verified", Description: "User verifies email", Order: 3},
		{Name: "onboarding_started", Description: "User starts onboarding", Order: 4},
		{Name: "first_repo_connected", Description: "User connects first repository", Order: 5},
		{Name: "first_scan_triggered", Description: "User triggers first scan", Order: 6},
		{Name: "first_scan_completed", Description: "First scan completes successfully", Order: 7},
		{Name: "vscode_extension_installed", Description: "User installs VS Code extension", Order: 8},
		{Name: "github_action_added", Description: "User adds GitHub Action", Order: 9},
		{Name: "active_user", Description: "User becomes active (5+ scans)", Order: 10},
		{Name: "power_user", Description: "User becomes power user (50+ scans)", Order: 11},
	}

	return &FunnelTracker{
		redis:  redisClient,
		stages: stages,
	}
}

// TrackEvent records a funnel event
func (ft *FunnelTracker) TrackEvent(ctx context.Context, event FunnelEvent) error {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Store individual event
	eventKey := fmt.Sprintf("funnel:events:%s:%s", event.UserID, event.SessionID)
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	pipe := ft.redis.Pipeline()

	// Store event with expiration (30 days)
	pipe.LPush(ctx, eventKey, eventData)
	pipe.Expire(ctx, eventKey, 30*24*time.Hour)

	// Update stage counters
	stageKey := fmt.Sprintf("funnel:stage:%s", event.Event)
	pipe.SAdd(ctx, stageKey, event.UserID)
	pipe.Expire(ctx, stageKey, 30*24*time.Hour)

	// Update daily counters
	dateKey := event.Timestamp.Format("2006-01-02")
	dailyKey := fmt.Sprintf("funnel:daily:%s:%s", dateKey, event.Event)
	pipe.SAdd(ctx, dailyKey, event.UserID)
	pipe.Expire(ctx, dailyKey, 90*24*time.Hour)

	// Update source tracking
	if event.Source != "" {
		sourceKey := fmt.Sprintf("funnel:source:%s:%s", event.Source, event.Event)
		pipe.SAdd(ctx, sourceKey, event.UserID)
		pipe.Expire(ctx, sourceKey, 30*24*time.Hour)
	}

	// Update campaign tracking
	if event.Campaign != "" {
		campaignKey := fmt.Sprintf("funnel:campaign:%s:%s", event.Campaign, event.Event)
		pipe.SAdd(ctx, campaignKey, event.UserID)
		pipe.Expire(ctx, campaignKey, 30*24*time.Hour)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	log.Printf("Tracked funnel event: %s for user %s", event.Event, event.UserID)
	return nil
}

// GetFunnelMetrics calculates conversion metrics for all stages
func (ft *FunnelTracker) GetFunnelMetrics(ctx context.Context) ([]FunnelMetrics, error) {
	var metrics []FunnelMetrics

	for i, stage := range ft.stages {
		stageKey := fmt.Sprintf("funnel:stage:%s", stage.Name)
		
		// Get total users for this stage
		totalUsers, err := ft.redis.SCard(ctx, stageKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get stage count for %s: %w", stage.Name, err)
		}

		var convertedUsers int64 = 0
		var conversionRate float64 = 0
		var dropoffRate float64 = 0

		// Calculate conversion rate to next stage
		if i < len(ft.stages)-1 {
			nextStage := ft.stages[i+1]
			nextStageKey := fmt.Sprintf("funnel:stage:%s", nextStage.Name)
			
			convertedUsers, err = ft.redis.SCard(ctx, nextStageKey).Result()
			if err != nil {
				return nil, fmt.Errorf("failed to get next stage count for %s: %w", nextStage.Name, err)
			}

			if totalUsers > 0 {
				conversionRate = float64(convertedUsers) / float64(totalUsers) * 100
				dropoffRate = 100 - conversionRate
			}
		}

		// Calculate average time (simplified - would need more complex logic for real implementation)
		averageTime := time.Duration(0)

		metric := FunnelMetrics{
			Stage:          stage.Name,
			TotalUsers:     int(totalUsers),
			ConvertedUsers: int(convertedUsers),
			ConversionRate: conversionRate,
			DropoffRate:    dropoffRate,
			AverageTime:    averageTime,
			LastUpdated:    time.Now(),
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetSourceMetrics returns conversion metrics by traffic source
func (ft *FunnelTracker) GetSourceMetrics(ctx context.Context, stage string) (map[string]int, error) {
	sources := []string{"github", "vscode", "web", "api", "organic", "referral"}
	sourceMetrics := make(map[string]int)

	for _, source := range sources {
		sourceKey := fmt.Sprintf("funnel:source:%s:%s", source, stage)
		count, err := ft.redis.SCard(ctx, sourceKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get source metrics for %s: %w", source, err)
		}
		sourceMetrics[source] = int(count)
	}

	return sourceMetrics, nil
}

// GetCampaignMetrics returns conversion metrics by campaign
func (ft *FunnelTracker) GetCampaignMetrics(ctx context.Context, stage string) (map[string]int, error) {
	campaigns := []string{"beta", "marketplace", "organic", "referral", "paid"}
	campaignMetrics := make(map[string]int)

	for _, campaign := range campaigns {
		campaignKey := fmt.Sprintf("funnel:campaign:%s:%s", campaign, stage)
		count, err := ft.redis.SCard(ctx, campaignKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get campaign metrics for %s: %w", campaign, err)
		}
		campaignMetrics[campaign] = int(count)
	}

	return campaignMetrics, nil
}

// GetDailyMetrics returns daily conversion metrics for a specific stage
func (ft *FunnelTracker) GetDailyMetrics(ctx context.Context, stage string, days int) (map[string]int, error) {
	dailyMetrics := make(map[string]int)

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		dailyKey := fmt.Sprintf("funnel:daily:%s:%s", date, stage)
		
		count, err := ft.redis.SCard(ctx, dailyKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get daily metrics for %s: %w", date, err)
		}
		
		dailyMetrics[date] = int(count)
	}

	return dailyMetrics, nil
}

// GetUserJourney returns the complete journey for a specific user
func (ft *FunnelTracker) GetUserJourney(ctx context.Context, userID string) ([]FunnelEvent, error) {
	// Get all sessions for the user
	pattern := fmt.Sprintf("funnel:events:%s:*", userID)
	keys, err := ft.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user event keys: %w", err)
	}

	var allEvents []FunnelEvent

	for _, key := range keys {
		events, err := ft.redis.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			continue
		}

		for _, eventData := range events {
			var event FunnelEvent
			if err := json.Unmarshal([]byte(eventData), &event); err != nil {
				continue
			}
			allEvents = append(allEvents, event)
		}
	}

	// Sort events by timestamp
	for i := 0; i < len(allEvents)-1; i++ {
		for j := i + 1; j < len(allEvents); j++ {
			if allEvents[i].Timestamp.After(allEvents[j].Timestamp) {
				allEvents[i], allEvents[j] = allEvents[j], allEvents[i]
			}
		}
	}

	return allEvents, nil
}

// GetCohortAnalysis performs cohort analysis for user retention
func (ft *FunnelTracker) GetCohortAnalysis(ctx context.Context, startDate, endDate time.Time) (map[string]map[string]float64, error) {
	cohorts := make(map[string]map[string]float64)

	// Simplified cohort analysis - would need more complex logic for real implementation
	current := startDate
	for current.Before(endDate) {
		cohortDate := current.Format("2006-01-02")
		
		// Get users who signed up on this date
		signupKey := fmt.Sprintf("funnel:daily:%s:signup_started", cohortDate)
		signupUsers, err := ft.redis.SMembers(ctx, signupKey).Result()
		if err != nil {
			current = current.AddDate(0, 0, 1)
			continue
		}

		if len(signupUsers) == 0 {
			current = current.AddDate(0, 0, 1)
			continue
		}

		cohortMetrics := make(map[string]float64)
		
		// Calculate retention for each week
		for week := 0; week < 12; week++ {
			retentionDate := current.AddDate(0, 0, week*7).Format("2006-01-02")
			activeKey := fmt.Sprintf("funnel:daily:%s:active_user", retentionDate)
			
			activeUsers, err := ft.redis.SMembers(ctx, activeKey).Result()
			if err != nil {
				continue
			}

			// Count how many signup users are still active
			retainedCount := 0
			for _, signupUser := range signupUsers {
				for _, activeUser := range activeUsers {
					if signupUser == activeUser {
						retainedCount++
						break
					}
				}
			}

			retentionRate := float64(retainedCount) / float64(len(signupUsers)) * 100
			cohortMetrics[fmt.Sprintf("week_%d", week)] = retentionRate
		}

		cohorts[cohortDate] = cohortMetrics
		current = current.AddDate(0, 0, 1)
	}

	return cohorts, nil
}

// Predefined funnel events for easy tracking
const (
	EventLanding                = "landing"
	EventSignupStarted          = "signup_started"
	EventEmailVerified          = "email_verified"
	EventOnboardingStarted      = "onboarding_started"
	EventFirstRepoConnected     = "first_repo_connected"
	EventFirstScanTriggered     = "first_scan_triggered"
	EventFirstScanCompleted     = "first_scan_completed"
	EventVSCodeExtensionInstalled = "vscode_extension_installed"
	EventGitHubActionAdded      = "github_action_added"
	EventActiveUser             = "active_user"
	EventPowerUser              = "power_user"
)

// Helper functions for common tracking scenarios

// TrackSignup tracks when a user starts the signup process
func (ft *FunnelTracker) TrackSignup(ctx context.Context, userID, sessionID, source, campaign string) error {
	return ft.TrackEvent(ctx, FunnelEvent{
		UserID:    userID,
		SessionID: sessionID,
		Event:     EventSignupStarted,
		Source:    source,
		Campaign:  campaign,
		Properties: map[string]interface{}{
			"signup_method": "email",
		},
	})
}

// TrackFirstScan tracks when a user completes their first scan
func (ft *FunnelTracker) TrackFirstScan(ctx context.Context, userID, sessionID string, scanType string, findingsCount int) error {
	return ft.TrackEvent(ctx, FunnelEvent{
		UserID:    userID,
		SessionID: sessionID,
		Event:     EventFirstScanCompleted,
		Properties: map[string]interface{}{
			"scan_type":      scanType,
			"findings_count": findingsCount,
		},
	})
}

// TrackExtensionInstall tracks VS Code extension installation
func (ft *FunnelTracker) TrackExtensionInstall(ctx context.Context, userID, sessionID string) error {
	return ft.TrackEvent(ctx, FunnelEvent{
		UserID:    userID,
		SessionID: sessionID,
		Event:     EventVSCodeExtensionInstalled,
		Source:    "vscode",
		Properties: map[string]interface{}{
			"extension_version": "1.0.0",
		},
	})
}