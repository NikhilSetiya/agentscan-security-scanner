package database

import (
	"sync"
	"time"

	"github.com/agentscan/agentscan/internal/domain/repositories"
)

// RepositoryMetrics collects and tracks repository performance metrics
type RepositoryMetrics struct {
	namespace string
	mu        sync.RWMutex
	
	// Operation counters
	totalOperations  int64
	createOps        int64
	readOps          int64
	updateOps        int64
	deleteOps        int64
	
	// Latency tracking
	totalLatency     time.Duration
	operationTimes   []time.Duration
	maxOperations    int // Maximum number of operations to track for percentiles
	
	// Error tracking
	totalErrors      int64
	errorsByType     map[string]int64
	
	// Timestamps
	lastOperationAt  time.Time
	startTime        time.Time
}

// NewRepositoryMetrics creates a new repository metrics collector
func NewRepositoryMetrics(namespace string) *RepositoryMetrics {
	return &RepositoryMetrics{
		namespace:     namespace,
		errorsByType:  make(map[string]int64),
		maxOperations: 1000, // Keep last 1000 operations for percentile calculation
		startTime:     time.Now(),
	}
}

// RecordOperation records a repository operation
func (rm *RepositoryMetrics) RecordOperation(operation string, duration time.Duration, success bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.totalOperations++
	rm.totalLatency += duration
	rm.lastOperationAt = time.Now()
	
	// Track operation type
	switch operation {
	case "create", "create_batch":
		rm.createOps++
	case "get", "list", "query", "exists", "count":
		rm.readOps++
	case "update", "update_batch":
		rm.updateOps++
	case "delete", "delete_batch":
		rm.deleteOps++
	}
	
	// Track latency for percentile calculation
	rm.operationTimes = append(rm.operationTimes, duration)
	if len(rm.operationTimes) > rm.maxOperations {
		// Remove oldest entries to maintain size limit
		rm.operationTimes = rm.operationTimes[1:]
	}
	
	// Track errors
	if !success {
		rm.totalErrors++
		rm.errorsByType[operation]++
	}
}

// GetMetrics returns current repository metrics
func (rm *RepositoryMetrics) GetMetrics() *repositories.RepositoryMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	metrics := &repositories.RepositoryMetrics{
		TotalOperations:  rm.totalOperations,
		CreateOperations: rm.createOps,
		ReadOperations:   rm.readOps,
		UpdateOperations: rm.updateOps,
		DeleteOperations: rm.deleteOps,
		LastOperationAt:  rm.lastOperationAt,
	}
	
	// Calculate average latency
	if rm.totalOperations > 0 {
		metrics.AverageLatency = time.Duration(int64(rm.totalLatency) / rm.totalOperations)
	}
	
	// Calculate error rate
	if rm.totalOperations > 0 {
		metrics.ErrorRate = float64(rm.totalErrors) / float64(rm.totalOperations)
	}
	
	return metrics
}

// GetPerformanceMetrics returns detailed performance metrics
func (rm *RepositoryMetrics) GetPerformanceMetrics() *repositories.PerformanceMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	metrics := &repositories.PerformanceMetrics{
		TotalOperations: rm.totalOperations,
		ErrorRate:       0,
	}
	
	if rm.totalOperations > 0 {
		// Calculate average latency
		metrics.AverageLatency = time.Duration(int64(rm.totalLatency) / rm.totalOperations)
		
		// Calculate error rate
		metrics.ErrorRate = float64(rm.totalErrors) / float64(rm.totalOperations)
		
		// Calculate throughput
		elapsed := time.Since(rm.startTime)
		if elapsed > 0 {
			metrics.ThroughputPerSecond = float64(rm.totalOperations) / elapsed.Seconds()
		}
	}
	
	// Calculate percentiles from operation times
	if len(rm.operationTimes) > 0 {
		sortedTimes := make([]time.Duration, len(rm.operationTimes))
		copy(sortedTimes, rm.operationTimes)
		
		// Simple sort for percentile calculation
		for i := 0; i < len(sortedTimes); i++ {
			for j := i + 1; j < len(sortedTimes); j++ {
				if sortedTimes[i] > sortedTimes[j] {
					sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
				}
			}
		}
		
		// Calculate percentiles
		metrics.P50Latency = rm.calculatePercentile(sortedTimes, 0.5)
		metrics.P95Latency = rm.calculatePercentile(sortedTimes, 0.95)
		metrics.P99Latency = rm.calculatePercentile(sortedTimes, 0.99)
	}
	
	return metrics
}

// calculatePercentile calculates the specified percentile from sorted durations
func (rm *RepositoryMetrics) calculatePercentile(sortedTimes []time.Duration, percentile float64) time.Duration {
	if len(sortedTimes) == 0 {
		return 0
	}
	
	index := int(float64(len(sortedTimes)) * percentile)
	if index >= len(sortedTimes) {
		index = len(sortedTimes) - 1
	}
	
	return sortedTimes[index]
}

// Reset resets all metrics
func (rm *RepositoryMetrics) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.totalOperations = 0
	rm.createOps = 0
	rm.readOps = 0
	rm.updateOps = 0
	rm.deleteOps = 0
	rm.totalLatency = 0
	rm.totalErrors = 0
	rm.operationTimes = nil
	rm.errorsByType = make(map[string]int64)
	rm.startTime = time.Now()
}

// GetErrorsByType returns error counts by operation type
func (rm *RepositoryMetrics) GetErrorsByType() map[string]int64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	result := make(map[string]int64)
	for k, v := range rm.errorsByType {
		result[k] = v
	}
	
	return result
}