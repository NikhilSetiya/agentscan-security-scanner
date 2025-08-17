package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/benchmark"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/cache"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/monitoring"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
)

// PerformanceHandler handles performance monitoring and benchmarking endpoints
type PerformanceHandler struct {
	monitoring *monitoring.Service
	benchmark  *benchmark.Service
	statsCache *cache.StatsCache
}

// NewPerformanceHandler creates a new performance handler
func NewPerformanceHandler(monitoring *monitoring.Service, benchmark *benchmark.Service, statsCache *cache.StatsCache) *PerformanceHandler {
	return &PerformanceHandler{
		monitoring: monitoring,
		benchmark:  benchmark,
		statsCache: statsCache,
	}
}

// GetSystemMetrics returns current system performance metrics
func (h *PerformanceHandler) GetSystemMetrics(c *gin.Context) {
	metrics := h.monitoring.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
	})
}

// GetResourceAlerts returns current resource alerts
func (h *PerformanceHandler) GetResourceAlerts(c *gin.Context) {
	alerts := h.monitoring.GetResourceAlerts()
	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
	})
}

// GetCacheStats returns cache performance statistics
func (h *PerformanceHandler) GetCacheStats(c *gin.Context) {
	// Get system metrics from cache
	systemMetrics, err := h.statsCache.GetSystemMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve cache stats",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cache_stats": systemMetrics,
	})
}

// StartLoadTest initiates a load test
func (h *PerformanceHandler) StartLoadTest(c *gin.Context) {
	var params benchmark.LoadTestParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid load test parameters",
		})
		return
	}

	result, err := h.benchmark.RunLoadTest(c.Request.Context(), &params)
	if err != nil {
		if errors.IsValidation(err) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start load test",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetLoadTestResult retrieves load test results
func (h *PerformanceHandler) GetLoadTestResult(c *gin.Context) {
	testID := c.Param("testId")
	if testID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Test ID is required",
		})
		return
	}

	result, err := h.benchmark.GetLoadTestResult(c.Request.Context(), testID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Load test not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve load test result",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// StartBenchmark initiates a performance benchmark
func (h *PerformanceHandler) StartBenchmark(c *gin.Context) {
	var params benchmark.BenchmarkParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid benchmark parameters",
		})
		return
	}

	result, err := h.benchmark.RunBenchmark(c.Request.Context(), &params)
	if err != nil {
		if errors.IsValidation(err) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start benchmark",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetBenchmarkResult retrieves benchmark results
func (h *PerformanceHandler) GetBenchmarkResult(c *gin.Context) {
	testID := c.Param("testId")
	if testID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Test ID is required",
		})
		return
	}

	result, err := h.benchmark.GetBenchmarkResult(c.Request.Context(), testID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Benchmark not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve benchmark result",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetDashboardStats returns cached dashboard statistics
func (h *PerformanceHandler) GetDashboardStats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User authentication required",
		})
		return
	}

	stats, err := h.statsCache.GetDashboardStats(c.Request.Context(), userID)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return empty stats if not cached yet
			c.JSON(http.StatusOK, gin.H{
				"stats": nil,
				"cached": false,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve dashboard stats",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":  stats,
		"cached": true,
	})
}

// GetRepositoryStats returns cached repository statistics
func (h *PerformanceHandler) GetRepositoryStats(c *gin.Context) {
	repoID := c.Param("repoId")
	if repoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Repository ID is required",
		})
		return
	}

	stats, err := h.statsCache.GetRepositoryStats(c.Request.Context(), repoID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Repository stats not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve repository stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAgentPerformance returns agent performance metrics
func (h *PerformanceHandler) GetAgentPerformance(c *gin.Context) {
	agentName := c.Param("agentName")
	if agentName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Agent name is required",
		})
		return
	}

	metrics, err := h.statsCache.GetAgentPerformance(c.Request.Context(), agentName)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Agent performance metrics not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve agent performance metrics",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetAverageScanDuration returns average scan duration for an agent
func (h *PerformanceHandler) GetAverageScanDuration(c *gin.Context) {
	agentName := c.Param("agentName")
	if agentName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Agent name is required",
		})
		return
	}

	hoursStr := c.DefaultQuery("hours", "24")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid hours parameter",
		})
		return
	}

	avgDuration, err := h.statsCache.GetAverageScanDuration(c.Request.Context(), agentName, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate average scan duration",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_name":       agentName,
		"hours":           hours,
		"average_duration": avgDuration.String(),
		"average_duration_ms": avgDuration.Milliseconds(),
	})
}

// InvalidateCache clears performance-related caches
func (h *PerformanceHandler) InvalidateCache(c *gin.Context) {
	cacheType := c.Query("type")
	
	switch cacheType {
	case "stats":
		err := h.statsCache.InvalidateStatsCache(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to invalidate stats cache",
			})
			return
		}
	case "all":
		// Invalidate all performance-related caches
		err := h.statsCache.InvalidateStatsCache(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to invalidate caches",
			})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid cache type. Use 'stats' or 'all'",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache invalidated successfully",
		"type":    cacheType,
	})
}
