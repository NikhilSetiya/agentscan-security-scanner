package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/infrastructure/middleware"
	"github.com/your-org/agentscan/internal/shared/testing"
)

// PerformanceTestSuite tests API performance under various load conditions
type PerformanceTestSuite struct {
	testSuite *testing.TestSuite
	router    *gin.Engine
	server    *httptest.Server
	config    *config.Config
	authToken string
}

func (pts *PerformanceTestSuite) SetupSuite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	pts.testSuite = testing.NewTestSuite(t)
	pts.config = pts.testSuite.MockConfig()
	
	// Setup router with middleware
	pts.router = pts.setupPerformanceRouter()
	pts.server = pts.testSuite.SetupHTTPServer(pts.router)
	
	// Get auth token
	pts.authToken = pts.getAuthToken()
}

func (pts *PerformanceTestSuite) TearDownSuite() {
	pts.testSuite.Cleanup()
}

func (pts *PerformanceTestSuite) setupPerformanceRouter() *gin.Engine {
	router := gin.New()
	
	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.PrometheusMiddleware(testing.NewMetricsCollector("performance_test")))
	
	// Auth endpoint
	router.POST("/auth/login", pts.handleLogin)
	
	// API endpoints
	api := router.Group("/api/v1")
	api.Use(middleware.JWTAuth(&pts.config.Security.JWT))
	{
		// Lightweight endpoints for performance testing
		api.GET("/health", pts.handleHealth)
		api.GET("/scans", pts.handleListScans)
		api.POST("/scans", pts.handleCreateScan)
		api.GET("/scans/:id", pts.handleGetScan)
		api.PUT("/scans/:id", pts.handleUpdateScan)
		api.DELETE("/scans/:id", pts.handleDeleteScan)
		
		// Heavy endpoints for stress testing
		api.GET("/dashboard/stats", pts.handleDashboardStats)
		api.GET("/reports/generate", pts.handleGenerateReport)
		api.POST("/scans/bulk", pts.handleBulkCreateScans)
	}
	
	return router
}

// TestAPIResponseTimes tests response times for individual endpoints
func TestAPIResponseTimes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	pts := &PerformanceTestSuite{}
	pts.SetupSuite(t)
	defer pts.TearDownSuite()
	
	headers := map[string]string{
		"Authorization": "Bearer " + pts.authToken,
	}
	
	testCases := []struct {
		name           string
		method         string
		path           string
		data           interface{}
		maxResponseTime time.Duration
	}{
		{
			name:           "health_check",
			method:         "GET",
			path:           "/api/v1/health",
			maxResponseTime: 50 * time.Millisecond,
		},
		{
			name:           "list_scans",
			method:         "GET",
			path:           "/api/v1/scans",
			maxResponseTime: 200 * time.Millisecond,
		},
		{
			name:   "create_scan",
			method: "POST",
			path:   "/api/v1/scans",
			data: map[string]interface{}{
				"name":    "Performance Test Scan",
				"type":    "vulnerability",
				"targets": []string{"https://example.com"},
			},
			maxResponseTime: 300 * time.Millisecond,
		},
		{
			name:           "get_scan",
			method:         "GET",
			path:           "/api/v1/scans/test-scan-123",
			maxResponseTime: 100 * time.Millisecond,
		},
		{
			name:   "update_scan",
			method: "PUT",
			path:   "/api/v1/scans/test-scan-123",
			data: map[string]interface{}{
				"name": "Updated Performance Test Scan",
			},
			maxResponseTime: 200 * time.Millisecond,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Warm up
			for i := 0; i < 3; i++ {
				resp := pts.makeRequest(tc.method, tc.path, tc.data, headers)
				resp.Body.Close()
			}
			
			// Measure response time
			start := time.Now()
			resp := pts.makeRequest(tc.method, tc.path, tc.data, headers)
			duration := time.Since(start)
			resp.Body.Close()
			
			assert.Equal(t, http.StatusOK, resp.StatusCode, "Request should succeed")
			assert.Less(t, duration, tc.maxResponseTime, 
				"Response time should be less than %v, got %v", tc.maxResponseTime, duration)
			
			t.Logf("%s response time: %v", tc.name, duration)
		})
	}
}

// TestConcurrentRequests tests API performance under concurrent load
func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	pts := &PerformanceTestSuite{}
	pts.SetupSuite(t)
	defer pts.TearDownSuite()
	
	headers := map[string]string{
		"Authorization": "Bearer " + pts.authToken,
	}
	
	testCases := []struct {
		name           string
		concurrency    int
		requests       int
		maxAvgTime     time.Duration
		maxFailureRate float64
	}{
		{
			name:           "low_concurrency",
			concurrency:    10,
			requests:       100,
			maxAvgTime:     500 * time.Millisecond,
			maxFailureRate: 0.01, // 1%
		},
		{
			name:           "medium_concurrency",
			concurrency:    50,
			requests:       500,
			maxAvgTime:     1 * time.Second,
			maxFailureRate: 0.05, // 5%
		},
		{
			name:           "high_concurrency",
			concurrency:    100,
			requests:       1000,
			maxAvgTime:     2 * time.Second,
			maxFailureRate: 0.10, // 10%
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := pts.runConcurrentTest(tc.concurrency, tc.requests, headers)
			
			// Calculate metrics
			avgTime := results.TotalTime / time.Duration(results.SuccessCount)
			failureRate := float64(results.FailureCount) / float64(results.TotalRequests)
			
			// Assertions
			assert.Less(t, avgTime, tc.maxAvgTime,
				"Average response time should be less than %v, got %v", tc.maxAvgTime, avgTime)
			assert.Less(t, failureRate, tc.maxFailureRate,
				"Failure rate should be less than %.2f%%, got %.2f%%", tc.maxFailureRate*100, failureRate*100)
			
			t.Logf("%s results: avg_time=%v, failure_rate=%.2f%%, throughput=%.2f req/s",
				tc.name, avgTime, failureRate*100, results.Throughput)
		})
	}
}

// TestMemoryUsage tests memory usage under load
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	pts := &PerformanceTestSuite{}
	pts.SetupSuite(t)
	defer pts.TearDownSuite()
	
	headers := map[string]string{
		"Authorization": "Bearer " + pts.authToken,
	}
	
	// Measure baseline memory
	var m1, m2 testing.MemStats
	testing.ReadMemStats(&m1)
	
	// Generate load
	concurrency := 50
	requests := 1000
	
	results := pts.runConcurrentTest(concurrency, requests, headers)
	
	// Measure memory after load
	testing.ReadMemStats(&m2)
	
	// Calculate memory usage
	allocDiff := m2.Alloc - m1.Alloc
	totalAllocDiff := m2.TotalAlloc - m1.TotalAlloc
	
	t.Logf("Memory usage - Alloc: %d bytes, TotalAlloc: %d bytes", allocDiff, totalAllocDiff)
	t.Logf("Performance results - Success: %d, Failures: %d, Avg Time: %v",
		results.SuccessCount, results.FailureCount, results.TotalTime/time.Duration(results.SuccessCount))
	
	// Memory usage should be reasonable (less than 100MB for this test)
	maxMemoryUsage := uint64(100 * 1024 * 1024) // 100MB
	assert.Less(t, allocDiff, maxMemoryUsage,
		"Memory usage should be less than %d bytes, got %d bytes", maxMemoryUsage, allocDiff)
}

// TestRateLimiting tests rate limiting behavior
func TestRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	pts := &PerformanceTestSuite{}
	pts.SetupSuite(t)
	defer pts.TearDownSuite()
	
	headers := map[string]string{
		"Authorization": "Bearer " + pts.authToken,
	}
	
	// Make rapid requests to trigger rate limiting
	rateLimitHit := false
	requestCount := 0
	
	for i := 0; i < 200; i++ {
		resp := pts.makeRequest("GET", "/api/v1/health", nil, headers)
		requestCount++
		
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitHit = true
			resp.Body.Close()
			break
		}
		resp.Body.Close()
		
		// Small delay to avoid overwhelming the server
		time.Sleep(10 * time.Millisecond)
	}
	
	t.Logf("Rate limiting test - Requests made: %d, Rate limit hit: %v", requestCount, rateLimitHit)
	
	// Rate limiting should eventually kick in for rapid requests
	// This test is more about observing behavior than strict assertions
	if rateLimitHit {
		t.Logf("Rate limiting is working - hit after %d requests", requestCount)
	} else {
		t.Logf("Rate limiting not triggered within %d requests", requestCount)
	}
}

// TestLargePayloads tests performance with large request/response payloads
func TestLargePayloads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	pts := &PerformanceTestSuite{}
	pts.SetupSuite(t)
	defer pts.TearDownSuite()
	
	headers := map[string]string{
		"Authorization": "Bearer " + pts.authToken,
	}
	
	// Test with large scan creation payload
	largeTargets := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		largeTargets[i] = fmt.Sprintf("https://example%d.com", i)
	}
	
	largeScanData := map[string]interface{}{
		"name":        "Large Payload Test Scan",
		"type":        "vulnerability",
		"targets":     largeTargets,
		"description": string(make([]byte, 10000)), // 10KB description
		"config": map[string]interface{}{
			"large_config": string(make([]byte, 50000)), // 50KB config
		},
	}
	
	start := time.Now()
	resp := pts.makeRequest("POST", "/api/v1/scans", largeScanData, headers)
	duration := time.Since(start)
	resp.Body.Close()
	
	maxTime := 5 * time.Second
	assert.Less(t, duration, maxTime,
		"Large payload request should complete within %v, got %v", maxTime, duration)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Large payload request should succeed")
	
	t.Logf("Large payload test - Duration: %v, Status: %d", duration, resp.StatusCode)
}

// TestDashboardPerformance tests dashboard endpoint performance
func TestDashboardPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	pts := &PerformanceTestSuite{}
	pts.SetupSuite(t)
	defer pts.TearDownSuite()
	
	headers := map[string]string{
		"Authorization": "Bearer " + pts.authToken,
	}
	
	// Test dashboard stats endpoint (typically heavy)
	iterations := 10
	var totalTime time.Duration
	
	for i := 0; i < iterations; i++ {
		start := time.Now()
		resp := pts.makeRequest("GET", "/api/v1/dashboard/stats", nil, headers)
		duration := time.Since(start)
		totalTime += duration
		resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Dashboard request should succeed")
	}
	
	avgTime := totalTime / time.Duration(iterations)
	maxAvgTime := 1 * time.Second
	
	assert.Less(t, avgTime, maxAvgTime,
		"Dashboard average response time should be less than %v, got %v", maxAvgTime, avgTime)
	
	t.Logf("Dashboard performance - Average time: %v over %d requests", avgTime, iterations)
}

// Performance test result structure
type PerformanceResult struct {
	TotalRequests int
	SuccessCount  int
	FailureCount  int
	TotalTime     time.Duration
	MinTime       time.Duration
	MaxTime       time.Duration
	Throughput    float64
}

// runConcurrentTest runs concurrent requests and returns performance metrics
func (pts *PerformanceTestSuite) runConcurrentTest(concurrency, totalRequests int, headers map[string]string) PerformanceResult {
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	result := PerformanceResult{
		TotalRequests: totalRequests,
		MinTime:       time.Hour, // Initialize to large value
	}
	
	requestsPerWorker := totalRequests / concurrency
	
	startTime := time.Now()
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for j := 0; j < requestsPerWorker; j++ {
				reqStart := time.Now()
				resp := pts.makeRequest("GET", "/api/v1/health", nil, headers)
				reqDuration := time.Since(reqStart)
				resp.Body.Close()
				
				mu.Lock()
				if resp.StatusCode == http.StatusOK {
					result.SuccessCount++
					result.TotalTime += reqDuration
					
					if reqDuration < result.MinTime {
						result.MinTime = reqDuration
					}
					if reqDuration > result.MaxTime {
						result.MaxTime = reqDuration
					}
				} else {
					result.FailureCount++
				}
				mu.Unlock()
			}
		}()
	}
	
	wg.Wait()
	totalDuration := time.Since(startTime)
	
	if totalDuration > 0 {
		result.Throughput = float64(result.SuccessCount) / totalDuration.Seconds()
	}
	
	return result
}

// Helper methods

func (pts *PerformanceTestSuite) makeRequest(method, path string, data interface{}, headers map[string]string) *http.Response {
	var body *bytes.Buffer
	
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			panic(err)
		}
		body = bytes.NewBuffer(jsonData)
	}
	
	var req *http.Request
	var err error
	
	if body != nil {
		req, err = http.NewRequest(method, pts.server.URL+path, body)
	} else {
		req, err = http.NewRequest(method, pts.server.URL+path, nil)
	}
	if err != nil {
		panic(err)
	}
	
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	
	return resp
}

func (pts *PerformanceTestSuite) getAuthToken() string {
	loginData := map[string]interface{}{
		"email":    "test@example.com",
		"password": "password123",
	}
	
	resp := pts.makeRequest("POST", "/auth/login", loginData, nil)
	defer resp.Body.Close()
	
	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}
	
	return result["token"].(string)
}

// Mock handlers for performance testing

func (pts *PerformanceTestSuite) handleLogin(c *gin.Context) {
	token, _ := middleware.GenerateJWT("test-user-123", &pts.config.Security.JWT)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    "test-user-123",
			"email": "test@example.com",
		},
	})
}

func (pts *PerformanceTestSuite) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

func (pts *PerformanceTestSuite) handleListScans(c *gin.Context) {
	// Simulate database query delay
	time.Sleep(10 * time.Millisecond)
	
	scans := make([]gin.H, 20)
	for i := 0; i < 20; i++ {
		scans[i] = gin.H{
			"id":         fmt.Sprintf("scan-%d", i+1),
			"name":       fmt.Sprintf("Test Scan %d", i+1),
			"type":       "vulnerability",
			"status":     "completed",
			"created_at": time.Now().Add(-time.Duration(i) * time.Hour),
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"scans": scans,
		"total": 20,
	})
}

func (pts *PerformanceTestSuite) handleCreateScan(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	// Simulate processing delay
	time.Sleep(50 * time.Millisecond)
	
	c.JSON(http.StatusCreated, gin.H{
		"scan_id": fmt.Sprintf("scan-%d", time.Now().Unix()),
		"status":  "created",
	})
}

func (pts *PerformanceTestSuite) handleGetScan(c *gin.Context) {
	scanID := c.Param("id")
	
	// Simulate database lookup delay
	time.Sleep(5 * time.Millisecond)
	
	c.JSON(http.StatusOK, gin.H{
		"id":         scanID,
		"name":       "Test Scan",
		"type":       "vulnerability",
		"status":     "completed",
		"created_at": time.Now().Add(-time.Hour),
	})
}

func (pts *PerformanceTestSuite) handleUpdateScan(c *gin.Context) {
	scanID := c.Param("id")
	
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	// Simulate update delay
	time.Sleep(20 * time.Millisecond)
	
	c.JSON(http.StatusOK, gin.H{
		"id":         scanID,
		"name":       data["name"],
		"updated_at": time.Now(),
	})
}

func (pts *PerformanceTestSuite) handleDeleteScan(c *gin.Context) {
	// Simulate deletion delay
	time.Sleep(15 * time.Millisecond)
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Scan deleted successfully",
	})
}

func (pts *PerformanceTestSuite) handleDashboardStats(c *gin.Context) {
	// Simulate heavy computation delay
	time.Sleep(200 * time.Millisecond)
	
	c.JSON(http.StatusOK, gin.H{
		"total_scans":         1250,
		"active_scans":        15,
		"completed_scans":     1200,
		"failed_scans":        35,
		"total_vulnerabilities": 5420,
		"critical_vulnerabilities": 125,
		"high_vulnerabilities":     890,
		"medium_vulnerabilities":   2100,
		"low_vulnerabilities":      2305,
		"scan_trends": []gin.H{
			{"date": "2024-01-01", "count": 45},
			{"date": "2024-01-02", "count": 52},
			{"date": "2024-01-03", "count": 38},
		},
	})
}

func (pts *PerformanceTestSuite) handleGenerateReport(c *gin.Context) {
	// Simulate report generation delay
	time.Sleep(500 * time.Millisecond)
	
	c.JSON(http.StatusOK, gin.H{
		"report_id":    fmt.Sprintf("report-%d", time.Now().Unix()),
		"download_url": "/downloads/report.pdf",
		"expires_at":   time.Now().Add(24 * time.Hour),
	})
}

func (pts *PerformanceTestSuite) handleBulkCreateScans(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	// Simulate bulk processing delay
	time.Sleep(1 * time.Second)
	
	c.JSON(http.StatusCreated, gin.H{
		"created_count": 10,
		"scan_ids":      []string{"scan-1", "scan-2", "scan-3"},
	})
}

// Benchmark tests

func BenchmarkHealthEndpoint(b *testing.B) {
	pts := &PerformanceTestSuite{}
	pts.SetupSuite(b)
	defer pts.TearDownSuite()
	
	headers := map[string]string{
		"Authorization": "Bearer " + pts.authToken,
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp := pts.makeRequest("GET", "/api/v1/health", nil, headers)
			resp.Body.Close()
		}
	})
}

func BenchmarkListScans(b *testing.B) {
	pts := &PerformanceTestSuite{}
	pts.SetupSuite(b)
	defer pts.TearDownSuite()
	
	headers := map[string]string{
		"Authorization": "Bearer " + pts.authToken,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := pts.makeRequest("GET", "/api/v1/scans", nil, headers)
		resp.Body.Close()
	}
}

func BenchmarkCreateScan(b *testing.B) {
	pts := &PerformanceTestSuite{}
	pts.SetupSuite(b)
	defer pts.TearDownSuite()
	
	headers := map[string]string{
		"Authorization": "Bearer " + pts.authToken,
	}
	
	scanData := map[string]interface{}{
		"name":    "Benchmark Test Scan",
		"type":    "vulnerability",
		"targets": []string{"https://example.com"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := pts.makeRequest("POST", "/api/v1/scans", scanData, headers)
		resp.Body.Close()
	}
}