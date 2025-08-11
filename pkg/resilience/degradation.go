package resilience

import (
	"fmt"
	"sync"
	"time"

	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/logging"
)

// DegradationLevel represents the level of service degradation
type DegradationLevel int

const (
	// LevelNormal - all services are operational
	LevelNormal DegradationLevel = iota
	// LevelPartial - some services are degraded but core functionality works
	LevelPartial
	// LevelSevere - significant degradation, only essential services work
	LevelSevere
	// LevelCritical - system is barely functional
	LevelCritical
)

func (l DegradationLevel) String() string {
	switch l {
	case LevelNormal:
		return "NORMAL"
	case LevelPartial:
		return "PARTIAL"
	case LevelSevere:
		return "SEVERE"
	case LevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Name         string
	Healthy      bool
	LastCheck    time.Time
	ErrorCount   int
	ResponseTime time.Duration
	Message      string
}

// DegradationManager manages service degradation based on health status
type DegradationManager struct {
	services map[string]*ServiceHealth
	mutex    sync.RWMutex
	logger   *logging.Logger

	// Configuration
	checkInterval     time.Duration
	unhealthyThreshold int
	degradationRules  map[string]DegradationLevel
}

// NewDegradationManager creates a new degradation manager
func NewDegradationManager() *DegradationManager {
	return &DegradationManager{
		services:           make(map[string]*ServiceHealth),
		logger:             logging.GetLogger(),
		checkInterval:      30 * time.Second,
		unhealthyThreshold: 3,
		degradationRules:   make(map[string]DegradationLevel),
	}
}

// RegisterService registers a service for health monitoring
func (dm *DegradationManager) RegisterService(name string, degradationLevel DegradationLevel) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.services[name] = &ServiceHealth{
		Name:      name,
		Healthy:   true,
		LastCheck: time.Now(),
	}
	dm.degradationRules[name] = degradationLevel
}

// UpdateServiceHealth updates the health status of a service
func (dm *DegradationManager) UpdateServiceHealth(name string, healthy bool, responseTime time.Duration, message string) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	service, exists := dm.services[name]
	if !exists {
		dm.logger.Warn("Attempted to update health for unregistered service", "service", name)
		return
	}

	service.LastCheck = time.Now()
	service.ResponseTime = responseTime
	service.Message = message

	if healthy {
		service.Healthy = true
		service.ErrorCount = 0
	} else {
		service.ErrorCount++
		if service.ErrorCount >= dm.unhealthyThreshold {
			service.Healthy = false
		}
	}

	dm.logger.Debug("Service health updated",
		"service", name,
		"healthy", service.Healthy,
		"error_count", service.ErrorCount,
		"response_time", responseTime,
		"message", message,
	)
}

// GetCurrentDegradationLevel returns the current system degradation level
func (dm *DegradationManager) GetCurrentDegradationLevel() DegradationLevel {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	maxLevel := LevelNormal
	unhealthyServices := 0
	totalServices := len(dm.services)

	for name, service := range dm.services {
		if !service.Healthy {
			unhealthyServices++
			if level, exists := dm.degradationRules[name]; exists && level > maxLevel {
				maxLevel = level
			}
		}
	}

	// Apply additional rules based on percentage of unhealthy services
	if totalServices > 0 {
		unhealthyPercentage := float64(unhealthyServices) / float64(totalServices)
		if unhealthyPercentage >= 0.75 {
			if maxLevel < LevelCritical {
				maxLevel = LevelCritical
			}
		} else if unhealthyPercentage >= 0.5 {
			if maxLevel < LevelSevere {
				maxLevel = LevelSevere
			}
		} else if unhealthyPercentage >= 0.25 {
			if maxLevel < LevelPartial {
				maxLevel = LevelPartial
			}
		}
	}

	return maxLevel
}

// GetServiceHealth returns the health status of a specific service
func (dm *DegradationManager) GetServiceHealth(name string) (*ServiceHealth, bool) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	service, exists := dm.services[name]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return &ServiceHealth{
		Name:         service.Name,
		Healthy:      service.Healthy,
		LastCheck:    service.LastCheck,
		ErrorCount:   service.ErrorCount,
		ResponseTime: service.ResponseTime,
		Message:      service.Message,
	}, true
}

// GetAllServiceHealth returns the health status of all services
func (dm *DegradationManager) GetAllServiceHealth() map[string]*ServiceHealth {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	result := make(map[string]*ServiceHealth)
	for name, service := range dm.services {
		result[name] = &ServiceHealth{
			Name:         service.Name,
			Healthy:      service.Healthy,
			LastCheck:    service.LastCheck,
			ErrorCount:   service.ErrorCount,
			ResponseTime: service.ResponseTime,
			Message:      service.Message,
		}
	}
	return result
}

// IsServiceHealthy checks if a specific service is healthy
func (dm *DegradationManager) IsServiceHealthy(name string) bool {
	service, exists := dm.GetServiceHealth(name)
	return exists && service.Healthy
}

// GetHealthyServices returns a list of healthy services
func (dm *DegradationManager) GetHealthyServices() []string {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	var healthy []string
	for name, service := range dm.services {
		if service.Healthy {
			healthy = append(healthy, name)
		}
	}
	return healthy
}

// GetUnhealthyServices returns a list of unhealthy services
func (dm *DegradationManager) GetUnhealthyServices() []string {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	var unhealthy []string
	for name, service := range dm.services {
		if !service.Healthy {
			unhealthy = append(unhealthy, name)
		}
	}
	return unhealthy
}

// AgentDegradationHandler handles graceful degradation for agent failures
type AgentDegradationHandler struct {
	degradationManager *DegradationManager
	logger             *logging.Logger
	
	// Agent-specific configuration
	minRequiredAgents int
	fallbackAgents    map[string][]string // agent -> fallback agents
}

// NewAgentDegradationHandler creates a new agent degradation handler
func NewAgentDegradationHandler(minRequiredAgents int) *AgentDegradationHandler {
	return &AgentDegradationHandler{
		degradationManager: NewDegradationManager(),
		logger:             logging.GetLogger(),
		minRequiredAgents:  minRequiredAgents,
		fallbackAgents:     make(map[string][]string),
	}
}

// RegisterAgent registers an agent for health monitoring
func (adh *AgentDegradationHandler) RegisterAgent(name string, degradationLevel DegradationLevel, fallbacks []string) {
	adh.degradationManager.RegisterService(name, degradationLevel)
	if len(fallbacks) > 0 {
		adh.fallbackAgents[name] = fallbacks
	}
}

// UpdateAgentHealth updates the health status of an agent
func (adh *AgentDegradationHandler) UpdateAgentHealth(name string, healthy bool, responseTime time.Duration, message string) {
	adh.degradationManager.UpdateServiceHealth(name, healthy, responseTime, message)
}

// GetAvailableAgents returns a list of available agents for scanning
func (adh *AgentDegradationHandler) GetAvailableAgents(requestedAgents []string) ([]string, error) {
	var availableAgents []string
	var unavailableAgents []string

	// Check requested agents
	for _, agent := range requestedAgents {
		if adh.degradationManager.IsServiceHealthy(agent) {
			availableAgents = append(availableAgents, agent)
		} else {
			unavailableAgents = append(unavailableAgents, agent)
			
			// Try fallback agents
			if fallbacks, exists := adh.fallbackAgents[agent]; exists {
				for _, fallback := range fallbacks {
					if adh.degradationManager.IsServiceHealthy(fallback) {
						availableAgents = append(availableAgents, fallback)
						adh.logger.Info("Using fallback agent",
							"original", agent,
							"fallback", fallback,
						)
						break
					}
				}
			}
		}
	}

	// Check if we have minimum required agents
	if len(availableAgents) < adh.minRequiredAgents {
		return availableAgents, errors.NewInternalError(
			fmt.Sprintf("insufficient healthy agents: have %d, need %d (unavailable: %v)",
				len(availableAgents), adh.minRequiredAgents, unavailableAgents))
	}

	if len(unavailableAgents) > 0 {
		adh.logger.Warn("Some agents are unavailable",
			"unavailable", unavailableAgents,
			"available", availableAgents,
		)
	}

	return availableAgents, nil
}

// CanPerformScan checks if a scan can be performed given the current degradation level
func (adh *AgentDegradationHandler) CanPerformScan(scanType string) (bool, string) {
	level := adh.degradationManager.GetCurrentDegradationLevel()
	
	switch level {
	case LevelNormal:
		return true, ""
	case LevelPartial:
		if scanType == "full" {
			return false, "full scans are disabled during partial degradation"
		}
		return true, "operating with reduced agent set"
	case LevelSevere:
		if scanType == "full" || scanType == "comprehensive" {
			return false, "only basic scans are available during severe degradation"
		}
		return true, "operating with minimal agent set"
	case LevelCritical:
		return false, "scanning is disabled during critical system degradation"
	default:
		return false, "unknown degradation level"
	}
}

// GetDegradationStatus returns the current degradation status
func (adh *AgentDegradationHandler) GetDegradationStatus() map[string]interface{} {
	level := adh.degradationManager.GetCurrentDegradationLevel()
	healthyAgents := adh.degradationManager.GetHealthyServices()
	unhealthyAgents := adh.degradationManager.GetUnhealthyServices()
	
	return map[string]interface{}{
		"degradation_level": level.String(),
		"healthy_agents":    healthyAgents,
		"unhealthy_agents":  unhealthyAgents,
		"total_agents":      len(healthyAgents) + len(unhealthyAgents),
		"can_scan":          len(healthyAgents) >= adh.minRequiredAgents,
	}
}