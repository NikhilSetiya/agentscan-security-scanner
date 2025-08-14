package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// PlanType represents different subscription plans
type PlanType string

const (
	PlanFree       PlanType = "free"
	PlanPro        PlanType = "pro"
	PlanTeam       PlanType = "team"
	PlanEnterprise PlanType = "enterprise"
)

// Usage represents usage metrics for a user/organization
type Usage struct {
	UserID           string    `json:"user_id"`
	OrganizationID   string    `json:"organization_id,omitempty"`
	Plan             PlanType  `json:"plan"`
	ScansThisMonth   int       `json:"scans_this_month"`
	ScansLimit       int       `json:"scans_limit"`
	PrivateRepos     int       `json:"private_repos"`
	PrivateReposLimit int      `json:"private_repos_limit"`
	LastReset        time.Time `json:"last_reset"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// PlanLimits defines the limits for each plan
type PlanLimits struct {
	ScansPerMonth      int  `json:"scans_per_month"`
	PrivateRepos       int  `json:"private_repos"`
	UnlimitedPublic    bool `json:"unlimited_public"`
	PrioritySupport    bool `json:"priority_support"`
	AdvancedFeatures   bool `json:"advanced_features"`
	CustomIntegrations bool `json:"custom_integrations"`
	SLA                bool `json:"sla"`
	Watermark          bool `json:"watermark"`
}

// GetPlanLimits returns the limits for a specific plan
func GetPlanLimits(plan PlanType) PlanLimits {
	switch plan {
	case PlanFree:
		return PlanLimits{
			ScansPerMonth:      100,
			PrivateRepos:       0,
			UnlimitedPublic:    true,
			PrioritySupport:    false,
			AdvancedFeatures:   false,
			CustomIntegrations: false,
			SLA:                false,
			Watermark:          true,
		}
	case PlanPro:
		return PlanLimits{
			ScansPerMonth:      1000,
			PrivateRepos:       5,
			UnlimitedPublic:    true,
			PrioritySupport:    true,
			AdvancedFeatures:   true,
			CustomIntegrations: false,
			SLA:                false,
			Watermark:          false,
		}
	case PlanTeam:
		return PlanLimits{
			ScansPerMonth:      5000,
			PrivateRepos:       25,
			UnlimitedPublic:    true,
			PrioritySupport:    true,
			AdvancedFeatures:   true,
			CustomIntegrations: true,
			SLA:                false,
			Watermark:          false,
		}
	case PlanEnterprise:
		return PlanLimits{
			ScansPerMonth:      -1, // Unlimited
			PrivateRepos:       -1, // Unlimited
			UnlimitedPublic:    true,
			PrioritySupport:    true,
			AdvancedFeatures:   true,
			CustomIntegrations: true,
			SLA:                true,
			Watermark:          false,
		}
	default:
		return GetPlanLimits(PlanFree)
	}
}

// FreemiumManager handles freemium model logic
type FreemiumManager struct {
	// In a real implementation, this would use a database
	usageStore map[string]*Usage
}

// NewFreemiumManager creates a new freemium manager
func NewFreemiumManager() *FreemiumManager {
	return &FreemiumManager{
		usageStore: make(map[string]*Usage),
	}
}

// CheckScanPermission checks if a user can perform a scan
func (fm *FreemiumManager) CheckScanPermission(ctx context.Context, userID string, repoURL string, isPrivate bool) (*ScanPermission, error) {
	usage, err := fm.GetUsage(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	limits := GetPlanLimits(usage.Plan)
	
	permission := &ScanPermission{
		Allowed:   true,
		Plan:      usage.Plan,
		Watermark: limits.Watermark,
		Message:   "",
	}

	// Check if it's a private repo and user has access
	if isPrivate {
		if limits.PrivateRepos == 0 {
			permission.Allowed = false
			permission.Message = "Private repository scanning requires a paid plan. Upgrade to Pro for $9/month."
			permission.UpgradeURL = "https://agentscan.dev/upgrade?utm_source=scan_limit&utm_medium=api&utm_campaign=freemium"
			return permission, nil
		}
		
		if limits.PrivateRepos > 0 && usage.PrivateRepos >= limits.PrivateRepos {
			permission.Allowed = false
			permission.Message = fmt.Sprintf("Private repository limit reached (%d/%d). Upgrade your plan for more repositories.", usage.PrivateRepos, limits.PrivateRepos)
			permission.UpgradeURL = "https://agentscan.dev/upgrade?utm_source=repo_limit&utm_medium=api&utm_campaign=freemium"
			return permission, nil
		}
	}

	// Check monthly scan limits (only for non-unlimited plans)
	if limits.ScansPerMonth > 0 {
		// Reset monthly usage if needed
		if fm.shouldResetUsage(usage) {
			usage.ScansThisMonth = 0
			usage.LastReset = time.Now()
		}

		if usage.ScansThisMonth >= limits.ScansPerMonth {
			permission.Allowed = false
			permission.Message = fmt.Sprintf("Monthly scan limit reached (%d/%d). Upgrade for unlimited scans.", usage.ScansThisMonth, limits.ScansPerMonth)
			permission.UpgradeURL = "https://agentscan.dev/upgrade?utm_source=scan_limit&utm_medium=api&utm_campaign=freemium"
			return permission, nil
		}
	}

	// For free plans, add watermark to public repos
	if usage.Plan == PlanFree && !isPrivate {
		permission.Watermark = true
		permission.Message = "Free plan includes AgentScan branding. Upgrade to Pro to remove watermarks."
	}

	return permission, nil
}

// RecordScan records a scan usage
func (fm *FreemiumManager) RecordScan(ctx context.Context, userID string, repoURL string, isPrivate bool) error {
	usage, err := fm.GetUsage(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get usage: %w", err)
	}

	// Reset monthly usage if needed
	if fm.shouldResetUsage(usage) {
		usage.ScansThisMonth = 0
		usage.LastReset = time.Now()
	}

	// Increment scan count
	usage.ScansThisMonth++
	usage.UpdatedAt = time.Now()

	// Track private repo if it's a new one
	if isPrivate {
		// In a real implementation, you'd track unique repositories
		// For now, we'll just increment the counter
		usage.PrivateRepos++
	}

	return fm.UpdateUsage(ctx, usage)
}

// GetUsage retrieves usage information for a user
func (fm *FreemiumManager) GetUsage(ctx context.Context, userID string) (*Usage, error) {
	if usage, exists := fm.usageStore[userID]; exists {
		return usage, nil
	}

	// Create new usage record for new user
	usage := &Usage{
		UserID:            userID,
		Plan:              PlanFree,
		ScansThisMonth:    0,
		ScansLimit:        GetPlanLimits(PlanFree).ScansPerMonth,
		PrivateRepos:      0,
		PrivateReposLimit: GetPlanLimits(PlanFree).PrivateRepos,
		LastReset:         time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	fm.usageStore[userID] = usage
	return usage, nil
}

// UpdateUsage updates usage information
func (fm *FreemiumManager) UpdateUsage(ctx context.Context, usage *Usage) error {
	usage.UpdatedAt = time.Now()
	fm.usageStore[usage.UserID] = usage
	return nil
}

// UpgradePlan upgrades a user's plan
func (fm *FreemiumManager) UpgradePlan(ctx context.Context, userID string, newPlan PlanType) error {
	usage, err := fm.GetUsage(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get usage: %w", err)
	}

	oldPlan := usage.Plan
	usage.Plan = newPlan
	
	limits := GetPlanLimits(newPlan)
	usage.ScansLimit = limits.ScansPerMonth
	usage.PrivateReposLimit = limits.PrivateRepos
	usage.UpdatedAt = time.Now()

	// Log the upgrade
	fmt.Printf("User %s upgraded from %s to %s", userID, oldPlan, newPlan)

	return fm.UpdateUsage(ctx, usage)
}

// shouldResetUsage checks if monthly usage should be reset
func (fm *FreemiumManager) shouldResetUsage(usage *Usage) bool {
	now := time.Now()
	lastReset := usage.LastReset
	
	// Reset if it's a new month
	return now.Year() > lastReset.Year() || 
		   (now.Year() == lastReset.Year() && now.Month() > lastReset.Month())
}

// ScanPermission represents the result of a permission check
type ScanPermission struct {
	Allowed    bool     `json:"allowed"`
	Plan       PlanType `json:"plan"`
	Watermark  bool     `json:"watermark"`
	Message    string   `json:"message"`
	UpgradeURL string   `json:"upgrade_url,omitempty"`
}

// ApplyWatermark adds watermark to scan results for free plans
func (fm *FreemiumManager) ApplyWatermark(findings []types.Finding, plan PlanType) []types.Finding {
	limits := GetPlanLimits(plan)
	
	if !limits.Watermark {
		return findings
	}

	// Add watermark to each finding
	for i := range findings {
		if findings[i].Description != "" {
			findings[i].Description += "\n\n---\n🛡️ Secured by AgentScan - Upgrade to Pro to remove this watermark: https://agentscan.dev/upgrade"
		}
		
		// Add watermark to fix suggestions
		if findings[i].FixSuggestion != "" {
			findings[i].FixSuggestion += "\n\nℹ️ Get unlimited scans and advanced features with AgentScan Pro: https://agentscan.dev/upgrade"
		}
	}

	return findings
}

// GetPlanFeatures returns a human-readable description of plan features
func GetPlanFeatures(plan PlanType) map[string]interface{} {
	limits := GetPlanLimits(plan)
	
	features := map[string]interface{}{
		"plan":                plan,
		"scans_per_month":     limits.ScansPerMonth,
		"private_repos":       limits.PrivateRepos,
		"unlimited_public":    limits.UnlimitedPublic,
		"priority_support":    limits.PrioritySupport,
		"advanced_features":   limits.AdvancedFeatures,
		"custom_integrations": limits.CustomIntegrations,
		"sla":                 limits.SLA,
		"watermark":           limits.Watermark,
	}

	// Add human-readable descriptions
	switch plan {
	case PlanFree:
		features["description"] = "Perfect for open source projects and getting started"
		features["price"] = "$0/month"
		features["highlights"] = []string{
			"Unlimited public repository scanning",
			"100 scans per month",
			"Community support",
			"Basic security scanning",
		}
	case PlanPro:
		features["description"] = "Ideal for individual developers and small teams"
		features["price"] = "$9/month"
		features["highlights"] = []string{
			"Everything in Free",
			"1,000 scans per month",
			"5 private repositories",
			"Priority support",
			"Advanced features",
			"No watermarks",
		}
	case PlanTeam:
		features["description"] = "Built for growing teams and organizations"
		features["price"] = "$29/month"
		features["highlights"] = []string{
			"Everything in Pro",
			"5,000 scans per month",
			"25 private repositories",
			"Custom integrations",
			"Team management",
			"Advanced analytics",
		}
	case PlanEnterprise:
		features["description"] = "Enterprise-grade security for large organizations"
		features["price"] = "Custom pricing"
		features["highlights"] = []string{
			"Everything in Team",
			"Unlimited scans",
			"Unlimited private repositories",
			"SLA guarantee",
			"Dedicated support",
			"Custom deployment options",
		}
	}

	return features
}

// GenerateUpgradeIncentive creates personalized upgrade messaging
func (fm *FreemiumManager) GenerateUpgradeIncentive(ctx context.Context, userID string) (*UpgradeIncentive, error) {
	usage, err := fm.GetUsage(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	if usage.Plan != PlanFree {
		return nil, nil // No incentive needed for paid plans
	}

	limits := GetPlanLimits(usage.Plan)
	
	incentive := &UpgradeIncentive{
		UserID:      userID,
		CurrentPlan: usage.Plan,
		Message:     "",
		Benefits:    []string{},
		UpgradeURL:  "https://agentscan.dev/upgrade?utm_source=incentive&utm_medium=api&utm_campaign=freemium",
	}

	// Calculate usage percentage
	usagePercent := float64(usage.ScansThisMonth) / float64(limits.ScansPerMonth) * 100

	if usagePercent >= 80 {
		incentive.Message = "You're using 80% of your monthly scans! Upgrade to Pro for unlimited scanning."
		incentive.Benefits = []string{
			"10x more scans (1,000/month)",
			"Private repository scanning",
			"Priority support",
			"Remove watermarks",
		}
	} else if usage.PrivateRepos > 0 {
		incentive.Message = "Unlock private repository scanning with AgentScan Pro!"
		incentive.Benefits = []string{
			"Scan up to 5 private repositories",
			"Advanced security features",
			"Priority support",
			"Professional reporting",
		}
	} else {
		incentive.Message = "Take your security to the next level with AgentScan Pro!"
		incentive.Benefits = []string{
			"10x more monthly scans",
			"Private repository support",
			"Advanced features",
			"Priority support",
		}
	}

	return incentive, nil
}

// UpgradeIncentive represents an upgrade incentive message
type UpgradeIncentive struct {
	UserID      string   `json:"user_id"`
	CurrentPlan PlanType `json:"current_plan"`
	Message     string   `json:"message"`
	Benefits    []string `json:"benefits"`
	UpgradeURL  string   `json:"upgrade_url"`
}