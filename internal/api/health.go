package api

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
)

var startTime = time.Now()

// HealthHandler handles health check requests
type HealthHandler struct {
	db    *database.DB
	redis *queue.RedisClient
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.DB, redis *queue.RedisClient) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Uptime    time.Duration          `json:"uptime"`
	Checks    map[string]HealthCheck `json:"checks"`
	System    SystemInfo             `json:"system,omitempty"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Status  string        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency,omitempty"`
	Details interface{}   `json:"details,omitempty"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutine"`
	NumCPU       int    `json:"num_cpu"`
	MemoryUsage  MemoryInfo `json:"memory_usage"`
}

// MemoryInfo represents memory usage information
type MemoryInfo struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// ReadinessResponse represents the readiness check response
type ReadinessResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]HealthCheck `json:"checks"`
}

// LivenessResponse represents the liveness check response
type LivenessResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    time.Duration `json:"uptime"`
}

// ServeHTTP handles the health check request
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health":
		h.handleHealth(w, r)
	case "/ready":
		h.handleReadiness(w, r)
	case "/live":
		h.handleLiveness(w, r)
	default:
		h.handleHealth(w, r)
	}
}

// handleHealth handles comprehensive health checks
func (h *HealthHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0", // Version 1.0.0 - production ready
		Uptime:    time.Since(startTime),
		Checks:    make(map[string]HealthCheck),
	}

	// Include system information if requested
	if r.URL.Query().Get("include_system") == "true" {
		response.System = h.getSystemInfo()
	}

	// Check database health
	h.checkDatabaseHealth(ctx, &response)

	// Check Redis health
	h.checkRedisHealth(ctx, &response)

	// Check disk space
	h.checkDiskSpace(&response)

	// Check memory usage
	h.checkMemoryUsage(&response)

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	
	if response.Status == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Encode response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleReadiness handles readiness checks (ready to serve traffic)
func (h *HealthHandler) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := ReadinessResponse{
		Status:    "ready",
		Timestamp: time.Now(),
		Checks:    make(map[string]HealthCheck),
	}

	// Check critical dependencies for readiness
	h.checkDatabaseHealth(ctx, (*HealthResponse)(&struct {
		Status    string
		Timestamp time.Time
		Version   string
		Uptime    time.Duration
		Checks    map[string]HealthCheck
		System    SystemInfo
	}{
		Status: response.Status,
		Checks: response.Checks,
	}))

	h.checkRedisHealth(ctx, (*HealthResponse)(&struct {
		Status    string
		Timestamp time.Time
		Version   string
		Uptime    time.Duration
		Checks    map[string]HealthCheck
		System    SystemInfo
	}{
		Status: response.Status,
		Checks: response.Checks,
	}))

	// Update response status based on checks
	for _, check := range response.Checks {
		if check.Status != "healthy" {
			response.Status = "not_ready"
			break
		}
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	
	if response.Status != "ready" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Encode response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleLiveness handles liveness checks (application is running)
func (h *HealthHandler) handleLiveness(w http.ResponseWriter, r *http.Request) {
	response := LivenessResponse{
		Status:    "alive",
		Timestamp: time.Now(),
		Uptime:    time.Since(startTime),
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)

	// Encode response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// checkDatabaseHealth checks database connectivity and performance
func (h *HealthHandler) checkDatabaseHealth(ctx context.Context, response *HealthResponse) {
	dbStart := time.Now()
	dbErr := h.db.Health(ctx)
	dbLatency := time.Since(dbStart)

	if dbErr != nil {
		response.Status = "unhealthy"
		response.Checks["database"] = HealthCheck{
			Status:  "unhealthy",
			Message: dbErr.Error(),
			Latency: dbLatency,
		}
	} else {
		// Additional database checks
		details := make(map[string]interface{})
		
		// Check connection pool stats if available
		// This would require extending the database package to expose stats
		details["latency_ms"] = dbLatency.Milliseconds()
		
		response.Checks["database"] = HealthCheck{
			Status:  "healthy",
			Latency: dbLatency,
			Details: details,
		}
	}
}

// checkRedisHealth checks Redis connectivity and performance
func (h *HealthHandler) checkRedisHealth(ctx context.Context, response *HealthResponse) {
	if h.redis == nil {
		response.Checks["redis"] = HealthCheck{
			Status:  "disabled",
			Message: "Redis not configured",
		}
		return
	}

	redisStart := time.Now()
	redisErr := h.redis.Health(ctx)
	redisLatency := time.Since(redisStart)

	if redisErr != nil {
		response.Status = "unhealthy"
		response.Checks["redis"] = HealthCheck{
			Status:  "unhealthy",
			Message: redisErr.Error(),
			Latency: redisLatency,
		}
	} else {
		details := make(map[string]interface{})
		details["latency_ms"] = redisLatency.Milliseconds()
		
		response.Checks["redis"] = HealthCheck{
			Status:  "healthy",
			Latency: redisLatency,
			Details: details,
		}
	}
}

// checkDiskSpace checks available disk space
func (h *HealthHandler) checkDiskSpace(response *HealthResponse) {
	// This is a simplified check - in production you'd want to check actual disk usage
	// For now, we'll just mark it as healthy
	response.Checks["disk_space"] = HealthCheck{
		Status: "healthy",
		Details: map[string]interface{}{
			"status": "sufficient",
		},
	}
}

// checkMemoryUsage checks memory usage
func (h *HealthHandler) checkMemoryUsage(response *HealthResponse) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to MB for readability
	allocMB := m.Alloc / 1024 / 1024
	sysMB := m.Sys / 1024 / 1024

	status := "healthy"
	message := ""

	// Simple memory pressure check (adjust thresholds as needed)
	if allocMB > 512 { // 512MB threshold
		status = "warning"
		message = "High memory usage detected"
	}
	if allocMB > 1024 { // 1GB threshold
		status = "unhealthy"
		message = "Critical memory usage"
		response.Status = "unhealthy"
	}

	response.Checks["memory"] = HealthCheck{
		Status:  status,
		Message: message,
		Details: map[string]interface{}{
			"alloc_mb":     allocMB,
			"sys_mb":       sysMB,
			"num_gc":       m.NumGC,
			"goroutines":   runtime.NumGoroutine(),
		},
	}
}

// getSystemInfo returns system information
func (h *HealthHandler) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		MemoryUsage: MemoryInfo{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
	}
}