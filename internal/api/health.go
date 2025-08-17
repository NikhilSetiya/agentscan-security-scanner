package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
)

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
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Status  string        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency,omitempty"`
}

// ServeHTTP handles the health check request
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0", // Version 1.0.0 - production ready
		Checks:    make(map[string]HealthCheck),
	}

	// Check database health
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
		response.Checks["database"] = HealthCheck{
			Status:  "healthy",
			Latency: dbLatency,
		}
	}

	// Check Redis health
	if h.redis != nil {
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
			response.Checks["redis"] = HealthCheck{
				Status:  "healthy",
				Latency: redisLatency,
			}
		}
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
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