package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/agentscan/agentscan/pkg/errors"
)

// AgentManager manages the lifecycle of security agents
type AgentManager struct {
	agents    map[string]agent.SecurityAgent
	configs   map[string]agent.AgentConfig
	health    map[string]AgentHealth
	mu        sync.RWMutex
	startTime time.Time
}

// AgentHealth represents the health status of an agent
type AgentHealth struct {
	Status       string    `json:"status"`
	LastCheck    time.Time `json:"last_check"`
	LastError    string    `json:"last_error,omitempty"`
	CheckCount   int64     `json:"check_count"`
	FailureCount int64     `json:"failure_count"`
}

// AgentStatus represents possible agent statuses
const (
	AgentStatusHealthy   = "healthy"
	AgentStatusUnhealthy = "unhealthy"
	AgentStatusUnknown   = "unknown"
)

// NewAgentManager creates a new agent manager
func NewAgentManager() *AgentManager {
	return &AgentManager{
		agents:    make(map[string]agent.SecurityAgent),
		configs:   make(map[string]agent.AgentConfig),
		health:    make(map[string]AgentHealth),
		startTime: time.Now(),
	}
}

// RegisterAgent registers a new security agent
func (am *AgentManager) RegisterAgent(name string, securityAgent agent.SecurityAgent) error {
	if name == "" {
		return errors.NewValidationError("agent name cannot be empty")
	}
	if securityAgent == nil {
		return errors.NewValidationError("agent cannot be nil")
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	// Check if agent is already registered
	if _, exists := am.agents[name]; exists {
		return errors.NewValidationError(fmt.Sprintf("agent %s is already registered", name))
	}

	// Get agent configuration
	config := securityAgent.GetConfig()

	// Register the agent
	am.agents[name] = securityAgent
	am.configs[name] = config
	am.health[name] = AgentHealth{
		Status:    AgentStatusUnknown,
		LastCheck: time.Now(),
	}

	return nil
}

// UnregisterAgent removes an agent from the manager
func (am *AgentManager) UnregisterAgent(name string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.agents[name]; !exists {
		return errors.NewNotFoundError("agent")
	}

	delete(am.agents, name)
	delete(am.configs, name)
	delete(am.health, name)

	return nil
}

// GetAgent retrieves an agent by name
func (am *AgentManager) GetAgent(name string) (agent.SecurityAgent, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	securityAgent, exists := am.agents[name]
	if !exists {
		return nil, errors.NewNotFoundError("agent")
	}

	return securityAgent, nil
}

// ListAgents returns a list of all registered agents
func (am *AgentManager) ListAgents() []string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	agents := make([]string, 0, len(am.agents))
	for name := range am.agents {
		agents = append(agents, name)
	}

	return agents
}

// GetAgentConfig retrieves the configuration for an agent
func (am *AgentManager) GetAgentConfig(name string) (agent.AgentConfig, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	config, exists := am.configs[name]
	if !exists {
		return agent.AgentConfig{}, errors.NewNotFoundError("agent")
	}

	return config, nil
}

// GetAgentHealth retrieves the health status for an agent
func (am *AgentManager) GetAgentHealth(name string) (AgentHealth, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	health, exists := am.health[name]
	if !exists {
		return AgentHealth{}, errors.NewNotFoundError("agent")
	}

	return health, nil
}

// HealthCheck performs a health check on a specific agent
func (am *AgentManager) HealthCheck(ctx context.Context, name string) error {
	am.mu.RLock()
	securityAgent, exists := am.agents[name]
	am.mu.RUnlock()

	if !exists {
		return errors.NewNotFoundError("agent")
	}

	// Perform health check with timeout
	healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := securityAgent.HealthCheck(healthCtx)

	// Update health status
	am.mu.Lock()
	health := am.health[name]
	health.LastCheck = time.Now()
	health.CheckCount++

	if err != nil {
		health.Status = AgentStatusUnhealthy
		health.LastError = err.Error()
		health.FailureCount++
	} else {
		health.Status = AgentStatusHealthy
		health.LastError = ""
	}

	am.health[name] = health
	am.mu.Unlock()

	return err
}

// HealthCheckAll performs health checks on all registered agents
func (am *AgentManager) HealthCheckAll(ctx context.Context) error {
	am.mu.RLock()
	agentNames := make([]string, 0, len(am.agents))
	for name := range am.agents {
		agentNames = append(agentNames, name)
	}
	am.mu.RUnlock()

	var lastError error
	for _, name := range agentNames {
		if err := am.HealthCheck(ctx, name); err != nil {
			lastError = err
			// Continue checking other agents
		}
	}

	return lastError
}

// ExecuteScan executes a scan using the specified agent
func (am *AgentManager) ExecuteScan(ctx context.Context, agentName string, config agent.ScanConfig) (*agent.ScanResult, error) {
	securityAgent, err := am.GetAgent(agentName)
	if err != nil {
		return nil, err
	}

	// Check agent health before execution
	if err := am.HealthCheck(ctx, agentName); err != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("agent %s is unhealthy", agentName)).WithCause(err)
	}

	// Execute the scan
	result, err := securityAgent.Scan(ctx, config)
	if err != nil {
		// Update health status on scan failure
		am.mu.Lock()
		health := am.health[agentName]
		health.FailureCount++
		health.LastError = err.Error()
		am.health[agentName] = health
		am.mu.Unlock()

		return nil, errors.NewInternalError(fmt.Sprintf("scan failed for agent %s", agentName)).WithCause(err)
	}

	return result, nil
}

// ExecuteParallelScans executes scans using multiple agents in parallel
func (am *AgentManager) ExecuteParallelScans(ctx context.Context, agentNames []string, config agent.ScanConfig) (map[string]*agent.ScanResult, error) {
	if len(agentNames) == 0 {
		return nil, errors.NewValidationError("no agents specified")
	}

	// Create channels for results and errors
	type scanResult struct {
		agentName string
		result    *agent.ScanResult
		err       error
	}

	resultCh := make(chan scanResult, len(agentNames))
	
	// Start scans in parallel
	for _, agentName := range agentNames {
		go func(name string) {
			result, err := am.ExecuteScan(ctx, name, config)
			resultCh <- scanResult{
				agentName: name,
				result:    result,
				err:       err,
			}
		}(agentName)
	}

	// Collect results
	results := make(map[string]*agent.ScanResult)
	var lastError error

	for i := 0; i < len(agentNames); i++ {
		select {
		case res := <-resultCh:
			if res.err != nil {
				lastError = res.err
				// Continue collecting other results
			} else {
				results[res.agentName] = res.result
			}
		case <-ctx.Done():
			return results, ctx.Err()
		}
	}

	// Return results even if some agents failed
	return results, lastError
}

// GetAgentCapabilities returns the capabilities of all agents
func (am *AgentManager) GetAgentCapabilities() map[string]agent.AgentConfig {
	am.mu.RLock()
	defer am.mu.RUnlock()

	capabilities := make(map[string]agent.AgentConfig)
	for name, config := range am.configs {
		capabilities[name] = config
	}

	return capabilities
}

// GetAgentsForLanguages returns agents that support the specified languages
func (am *AgentManager) GetAgentsForLanguages(languages []string) []string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var supportedAgents []string
	
	for agentName, config := range am.configs {
		// Check if agent supports any of the requested languages
		for _, requestedLang := range languages {
			for _, supportedLang := range config.SupportedLangs {
				if requestedLang == supportedLang {
					supportedAgents = append(supportedAgents, agentName)
					goto nextAgent // Found support, move to next agent
				}
			}
		}
		nextAgent:
	}

	return supportedAgents
}

// GetAgentsForCategories returns agents that support the specified vulnerability categories
func (am *AgentManager) GetAgentsForCategories(categories []agent.VulnCategory) []string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var supportedAgents []string
	
	for agentName, config := range am.configs {
		// Check if agent supports any of the requested categories
		for _, requestedCat := range categories {
			for _, supportedCat := range config.Categories {
				if requestedCat == supportedCat {
					supportedAgents = append(supportedAgents, agentName)
					goto nextAgent // Found support, move to next agent
				}
			}
		}
		nextAgent:
	}

	return supportedAgents
}

// Health returns the overall health of the agent manager
func (am *AgentManager) Health(ctx context.Context) error {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if len(am.agents) == 0 {
		return errors.NewInternalError("no agents registered")
	}

	// Check if any agents are healthy
	healthyCount := 0
	for _, health := range am.health {
		if health.Status == AgentStatusHealthy {
			healthyCount++
		}
	}

	if healthyCount == 0 {
		return errors.NewInternalError("no healthy agents available")
	}

	return nil
}

// GetStats returns statistics about the agent manager
func (am *AgentManager) GetStats() AgentManagerStats {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := AgentManagerStats{
		TotalAgents:   len(am.agents),
		HealthyAgents: 0,
		UnhealthyAgents: 0,
		UnknownAgents: 0,
		Uptime:        time.Since(am.startTime),
		Agents:        make(map[string]AgentStats),
	}

	for name, health := range am.health {
		config := am.configs[name]
		
		agentStats := AgentStats{
			Name:           name,
			Status:         health.Status,
			LastCheck:      health.LastCheck,
			CheckCount:     health.CheckCount,
			FailureCount:   health.FailureCount,
			LastError:      health.LastError,
			SupportedLangs: config.SupportedLangs,
			Categories:     config.Categories,
			RequiresDocker: config.RequiresDocker,
		}

		stats.Agents[name] = agentStats

		switch health.Status {
		case AgentStatusHealthy:
			stats.HealthyAgents++
		case AgentStatusUnhealthy:
			stats.UnhealthyAgents++
		default:
			stats.UnknownAgents++
		}
	}

	return stats
}

// AgentManagerStats represents statistics for the agent manager
type AgentManagerStats struct {
	TotalAgents     int                    `json:"total_agents"`
	HealthyAgents   int                    `json:"healthy_agents"`
	UnhealthyAgents int                    `json:"unhealthy_agents"`
	UnknownAgents   int                    `json:"unknown_agents"`
	Uptime          time.Duration          `json:"uptime"`
	Agents          map[string]AgentStats  `json:"agents"`
}

// AgentStats represents statistics for a single agent
type AgentStats struct {
	Name           string                  `json:"name"`
	Status         string                  `json:"status"`
	LastCheck      time.Time               `json:"last_check"`
	CheckCount     int64                   `json:"check_count"`
	FailureCount   int64                   `json:"failure_count"`
	LastError      string                  `json:"last_error,omitempty"`
	SupportedLangs []string                `json:"supported_languages"`
	Categories     []agent.VulnCategory    `json:"categories"`
	RequiresDocker bool                    `json:"requires_docker"`
}