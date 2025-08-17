package benchmark

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/cache"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/orchestrator"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// Service provides performance benchmarking and load testing capabilities
type Service struct {
	orchestrator *orchestrator.Service
	statsCache   *cache.StatsCache
	config       *Config
}

// Config holds benchmarking configuration
type Config struct {
	MaxConcurrentTests int           `json:"max_concurrent_tests"`
	DefaultTimeout     time.Duration `json:"default_timeout"`
	WarmupDuration     time.Duration `json:"warmup_duration"`
	TestDuration       time.Duration `json:"test_duration"`
	CooldownDuration   time.Duration `json:"cooldown_duration"`
}

// DefaultConfig returns default benchmarking configuration
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrentTests: 100,
		DefaultTimeout:     5 * time.Minute,
		WarmupDuration:     30 * time.Second,
		TestDuration:       2 * time.Minute,
		CooldownDuration:   30 * time.Second,
	}
}

// NewService creates a new benchmarking service
func NewService(orchestrator *orchestrator.Service, statsCache *cache.StatsCache, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		orchestrator: orchestrator,
		statsCache:   statsCache,
		config:       config,
	}
}

// RunLoadTest executes a load test with the specified parameters
func (s *Service) RunLoadTest(ctx context.Context, params *LoadTestParams) (*LoadTestResult, error) {
	if params == nil {
		return nil, errors.NewValidationError("load test parameters are required")
	}

	if err := s.validateParams(params); err != nil {
		return nil, err
	}

	result := &LoadTestResult{
		TestID:    fmt.Sprintf("load_test_%d", time.Now().Unix()),
		StartTime: time.Now(),
		Params:    *params,
		Metrics:   &LoadTestMetrics{},
	}

	// Run the load test
	if err := s.executeLoadTest(ctx, params, result); err != nil {
		result.Error = err.Error()
		result.Status = TestStatusFailed
	} else {
		result.Status = TestStatusCompleted
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Cache the result
	key := cache.CacheKey{Prefix: "load_test", ID: result.TestID}
	s.statsCache.Set(ctx, key, result, 24*time.Hour)

	return result, nil
}

// RunBenchmark executes a performance benchmark
func (s *Service) RunBenchmark(ctx context.Context, params *BenchmarkParams) (*BenchmarkResult, error) {
	if params == nil {
		return nil, errors.NewValidationError("benchmark parameters are required")
	}

	result := &BenchmarkResult{
		TestID:    fmt.Sprintf("benchmark_%d", time.Now().Unix()),
		StartTime: time.Now(),
		Params:    *params,
	}

	// Run the benchmark
	if err := s.executeBenchmark(ctx, params, result); err != nil {
		result.Error = err.Error()
		result.Status = TestStatusFailed
	} else {
		result.Status = TestStatusCompleted
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Cache the result
	key := cache.CacheKey{Prefix: "benchmark", ID: result.TestID}
	s.statsCache.Set(ctx, key, result, 24*time.Hour)

	return result, nil
}

// executeLoadTest runs the actual load test
func (s *Service) executeLoadTest(ctx context.Context, params *LoadTestParams, result *LoadTestResult) error {
	result.Status = TestStatusRunning

	// Warmup phase
	if s.config.WarmupDuration > 0 {
		if err := s.warmupPhase(ctx, params, result); err != nil {
			return err
		}
	}

	// Main test phase
	if err := s.mainTestPhase(ctx, params, result); err != nil {
		return err
	}

	// Cooldown phase
	if s.config.CooldownDuration > 0 {
		if err := s.cooldownPhase(ctx, params, result); err != nil {
			return err
		}
	}

	return nil
}

// warmupPhase executes the warmup phase of the load test
func (s *Service) warmupPhase(ctx context.Context, params *LoadTestParams, result *LoadTestResult) error {
	result.Phase = TestPhaseWarmup
	
	// Run a smaller number of requests to warm up the system
	warmupConcurrency := params.Concurrency / 4
	if warmupConcurrency < 1 {
		warmupConcurrency = 1
	}

	warmupCtx, cancel := context.WithTimeout(ctx, s.config.WarmupDuration)
	defer cancel()

	return s.runConcurrentRequests(warmupCtx, warmupConcurrency, params, result)
}

// mainTestPhase executes the main test phase
func (s *Service) mainTestPhase(ctx context.Context, params *LoadTestParams, result *LoadTestResult) error {
	result.Phase = TestPhaseMain
	
	testCtx, cancel := context.WithTimeout(ctx, s.config.TestDuration)
	defer cancel()

	return s.runConcurrentRequests(testCtx, params.Concurrency, params, result)
}

// cooldownPhase executes the cooldown phase
func (s *Service) cooldownPhase(ctx context.Context, params *LoadTestParams, result *LoadTestResult) error {
	result.Phase = TestPhaseCooldown
	
	// Wait for system to stabilize
	time.Sleep(s.config.CooldownDuration)
	return nil
}

// runConcurrentRequests executes concurrent scan requests
func (s *Service) runConcurrentRequests(ctx context.Context, concurrency int, params *LoadTestParams, result *LoadTestResult) error {
	var wg sync.WaitGroup
	requestCh := make(chan struct{}, concurrency)
	errorCh := make(chan error, concurrency)
	metricsCh := make(chan *RequestMetrics, concurrency*10)

	// Start metrics collector
	go s.collectRequestMetrics(metricsCh, result)

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(ctx, requestCh, errorCh, metricsCh, params)
		}()
	}

	// Send requests
	go func() {
		defer close(requestCh)
		for {
			select {
			case <-ctx.Done():
				return
			case requestCh <- struct{}{}:
				// Request sent
			}
		}
	}()

	// Wait for completion
	wg.Wait()
	close(metricsCh)
	close(errorCh)

	// Collect errors
	for err := range errorCh {
		if err != nil {
			result.Metrics.ErrorCount++
		}
	}

	return nil
}

// worker executes individual scan requests
func (s *Service) worker(ctx context.Context, requestCh <-chan struct{}, errorCh chan<- error, metricsCh chan<- *RequestMetrics, params *LoadTestParams) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-requestCh:
			if !ok {
				return
			}

			metrics := &RequestMetrics{
				StartTime: time.Now(),
			}

			// Execute scan request
			scanReq := &types.ScanRequest{
				RepoURL:     params.TestRepository,
				Branch:      "main",
				Incremental: params.IncrementalScans,
				Priority:    types.PriorityMedium,
			}

			job, err := s.orchestrator.SubmitScan(ctx, scanReq)
			if err != nil {
				metrics.Error = err
				errorCh <- err
			} else {
				metrics.JobID = job.ID
				
				// Wait for completion if specified
				if params.WaitForCompletion {
					s.waitForScanCompletion(ctx, job.ID, metrics)
				}
			}

			metrics.EndTime = time.Now()
			metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)

			select {
			case metricsCh <- metrics:
			case <-ctx.Done():
				return
			}
		}
	}
}

// waitForScanCompletion waits for a scan to complete
func (s *Service) waitForScanCompletion(ctx context.Context, jobID string, metrics *RequestMetrics) {
	timeout := time.NewTimer(s.config.DefaultTimeout)
	defer timeout.Stop()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout.C:
			metrics.TimedOut = true
			return
		case <-ticker.C:
			status, err := s.orchestrator.GetScanStatus(ctx, jobID)
			if err != nil {
				metrics.Error = err
				return
			}

			if status.Status == types.ScanStatusCompleted || status.Status == types.ScanStatusFailed {
				metrics.Completed = true
				return
			}
		}
	}
}

// collectRequestMetrics collects and aggregates request metrics
func (s *Service) collectRequestMetrics(metricsCh <-chan *RequestMetrics, result *LoadTestResult) {
	var totalDuration time.Duration
	var minDuration, maxDuration time.Duration
	var completedCount, timedOutCount int64

	for metrics := range metricsCh {
		result.Metrics.TotalRequests++

		if metrics.Error != nil {
			result.Metrics.ErrorCount++
			continue
		}

		if metrics.Completed {
			completedCount++
		}

		if metrics.TimedOut {
			timedOutCount++
		}

		totalDuration += metrics.Duration

		if minDuration == 0 || metrics.Duration < minDuration {
			minDuration = metrics.Duration
		}

		if metrics.Duration > maxDuration {
			maxDuration = metrics.Duration
		}
	}

	if result.Metrics.TotalRequests > 0 {
		result.Metrics.SuccessCount = result.Metrics.TotalRequests - result.Metrics.ErrorCount
		result.Metrics.SuccessRate = float64(result.Metrics.SuccessCount) / float64(result.Metrics.TotalRequests) * 100
	}

	if completedCount > 0 {
		result.Metrics.AverageResponseTime = totalDuration / time.Duration(completedCount)
		result.Metrics.MinResponseTime = minDuration
		result.Metrics.MaxResponseTime = maxDuration
	}

	result.Metrics.CompletionRate = float64(completedCount) / float64(result.Metrics.TotalRequests) * 100
	result.Metrics.TimeoutRate = float64(timedOutCount) / float64(result.Metrics.TotalRequests) * 100

	if result.Duration > 0 {
		result.Metrics.RequestsPerSecond = float64(result.Metrics.TotalRequests) / result.Duration.Seconds()
	}
}

// executeBenchmark runs a performance benchmark
func (s *Service) executeBenchmark(ctx context.Context, params *BenchmarkParams, result *BenchmarkResult) error {
	result.Status = TestStatusRunning

	// Run multiple iterations
	for i := 0; i < params.Iterations; i++ {
		iteration := &BenchmarkIteration{
			Number:    i + 1,
			StartTime: time.Now(),
		}

		// Execute the benchmark operation
		if err := s.executeBenchmarkIteration(ctx, params, iteration); err != nil {
			iteration.Error = err.Error()
		}

		iteration.EndTime = time.Now()
		iteration.Duration = iteration.EndTime.Sub(iteration.StartTime)

		result.Iterations = append(result.Iterations, iteration)
	}

	// Calculate aggregate metrics
	s.calculateBenchmarkMetrics(result)

	return nil
}

// executeBenchmarkIteration executes a single benchmark iteration
func (s *Service) executeBenchmarkIteration(ctx context.Context, params *BenchmarkParams, iteration *BenchmarkIteration) error {
	switch params.Operation {
	case BenchmarkOperationScan:
		return s.benchmarkScanOperation(ctx, params, iteration)
	case BenchmarkOperationQuery:
		return s.benchmarkQueryOperation(ctx, params, iteration)
	case BenchmarkOperationCache:
		return s.benchmarkCacheOperation(ctx, params, iteration)
	default:
		return errors.NewValidationError("unsupported benchmark operation")
	}
}

// benchmarkScanOperation benchmarks scan operations
func (s *Service) benchmarkScanOperation(ctx context.Context, params *BenchmarkParams, iteration *BenchmarkIteration) error {
	scanReq := &types.ScanRequest{
		RepoURL:     params.TestRepository,
		Branch:      "main",
		Incremental: false,
		Priority:    types.PriorityHigh,
	}

	job, err := s.orchestrator.SubmitScan(ctx, scanReq)
	if err != nil {
		return err
	}

	iteration.JobID = job.ID

	// Wait for completion
	return s.waitForBenchmarkCompletion(ctx, job.ID, iteration)
}

// benchmarkQueryOperation benchmarks database query operations
func (s *Service) benchmarkQueryOperation(ctx context.Context, params *BenchmarkParams, iteration *BenchmarkIteration) error {
	// This would benchmark database queries
	// Implementation depends on specific queries to benchmark
	return nil
}

// benchmarkCacheOperation benchmarks cache operations
func (s *Service) benchmarkCacheOperation(ctx context.Context, params *BenchmarkParams, iteration *BenchmarkIteration) error {
	// This would benchmark cache operations
	// Implementation depends on specific cache operations to benchmark
	return nil
}

// waitForBenchmarkCompletion waits for a benchmark operation to complete
func (s *Service) waitForBenchmarkCompletion(ctx context.Context, jobID string, iteration *BenchmarkIteration) error {
	timeout := time.NewTimer(s.config.DefaultTimeout)
	defer timeout.Stop()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return errors.NewTimeoutError("benchmark operation")
		case <-ticker.C:
			status, err := s.orchestrator.GetScanStatus(ctx, jobID)
			if err != nil {
				return err
			}

			if status.Status == types.ScanStatusCompleted {
				return nil
			}

			if status.Status == types.ScanStatusFailed {
				return errors.NewInternalError("scan failed")
			}
		}
	}
}

// calculateBenchmarkMetrics calculates aggregate benchmark metrics
func (s *Service) calculateBenchmarkMetrics(result *BenchmarkResult) {
	if len(result.Iterations) == 0 {
		return
	}

	var totalDuration time.Duration
	var minDuration, maxDuration time.Duration
	var successCount int

	for _, iteration := range result.Iterations {
		if iteration.Error == "" {
			successCount++
		}

		totalDuration += iteration.Duration

		if minDuration == 0 || iteration.Duration < minDuration {
			minDuration = iteration.Duration
		}

		if iteration.Duration > maxDuration {
			maxDuration = iteration.Duration
		}
	}

	result.Metrics = &BenchmarkMetrics{
		TotalIterations:     len(result.Iterations),
		SuccessfulIterations: successCount,
		FailedIterations:    len(result.Iterations) - successCount,
		AverageDuration:     totalDuration / time.Duration(len(result.Iterations)),
		MinDuration:         minDuration,
		MaxDuration:         maxDuration,
		SuccessRate:         float64(successCount) / float64(len(result.Iterations)) * 100,
	}
}

// validateParams validates load test parameters
func (s *Service) validateParams(params *LoadTestParams) error {
	if params.Concurrency <= 0 {
		return errors.NewValidationError("concurrency must be greater than 0")
	}

	if params.Concurrency > s.config.MaxConcurrentTests {
		return errors.NewValidationError("concurrency exceeds maximum allowed")
	}

	if params.TestRepository == "" {
		return errors.NewValidationError("test repository is required")
	}

	return nil
}

// GetLoadTestResult retrieves a cached load test result
func (s *Service) GetLoadTestResult(ctx context.Context, testID string) (*LoadTestResult, error) {
	key := cache.CacheKey{Prefix: "load_test", ID: testID}
	var result LoadTestResult
	if err := s.statsCache.Get(ctx, key, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetBenchmarkResult retrieves a cached benchmark result
func (s *Service) GetBenchmarkResult(ctx context.Context, testID string) (*BenchmarkResult, error) {
	key := cache.CacheKey{Prefix: "benchmark", ID: testID}
	var result BenchmarkResult
	if err := s.statsCache.Get(ctx, key, &result); err != nil {
		return nil, err
	}
	return &result, nil
}