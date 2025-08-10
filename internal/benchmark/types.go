package benchmark

import (
	"time"
)

// LoadTestParams defines parameters for load testing
type LoadTestParams struct {
	Concurrency        int    `json:"concurrency"`
	TestRepository     string `json:"test_repository"`
	IncrementalScans   bool   `json:"incremental_scans"`
	WaitForCompletion  bool   `json:"wait_for_completion"`
	Duration           time.Duration `json:"duration,omitempty"`
}

// LoadTestResult contains the results of a load test
type LoadTestResult struct {
	TestID    string           `json:"test_id"`
	StartTime time.Time        `json:"start_time"`
	EndTime   time.Time        `json:"end_time"`
	Duration  time.Duration    `json:"duration"`
	Status    TestStatus       `json:"status"`
	Phase     TestPhase        `json:"phase"`
	Params    LoadTestParams   `json:"params"`
	Metrics   *LoadTestMetrics `json:"metrics"`
	Error     string           `json:"error,omitempty"`
}

// LoadTestMetrics contains metrics from a load test
type LoadTestMetrics struct {
	TotalRequests       int64         `json:"total_requests"`
	SuccessCount        int64         `json:"success_count"`
	ErrorCount          int64         `json:"error_count"`
	SuccessRate         float64       `json:"success_rate"`
	CompletionRate      float64       `json:"completion_rate"`
	TimeoutRate         float64       `json:"timeout_rate"`
	RequestsPerSecond   float64       `json:"requests_per_second"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	MinResponseTime     time.Duration `json:"min_response_time"`
	MaxResponseTime     time.Duration `json:"max_response_time"`
}

// BenchmarkParams defines parameters for benchmarking
type BenchmarkParams struct {
	Operation      BenchmarkOperation `json:"operation"`
	Iterations     int                `json:"iterations"`
	TestRepository string             `json:"test_repository,omitempty"`
	QueryType      string             `json:"query_type,omitempty"`
	CacheOperation string             `json:"cache_operation,omitempty"`
}

// BenchmarkResult contains the results of a benchmark
type BenchmarkResult struct {
	TestID     string                `json:"test_id"`
	StartTime  time.Time             `json:"start_time"`
	EndTime    time.Time             `json:"end_time"`
	Duration   time.Duration         `json:"duration"`
	Status     TestStatus            `json:"status"`
	Params     BenchmarkParams       `json:"params"`
	Iterations []*BenchmarkIteration `json:"iterations"`
	Metrics    *BenchmarkMetrics     `json:"metrics"`
	Error      string                `json:"error,omitempty"`
}

// BenchmarkIteration represents a single benchmark iteration
type BenchmarkIteration struct {
	Number    int           `json:"number"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	JobID     string        `json:"job_id,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// BenchmarkMetrics contains aggregate benchmark metrics
type BenchmarkMetrics struct {
	TotalIterations      int           `json:"total_iterations"`
	SuccessfulIterations int           `json:"successful_iterations"`
	FailedIterations     int           `json:"failed_iterations"`
	AverageDuration      time.Duration `json:"average_duration"`
	MinDuration          time.Duration `json:"min_duration"`
	MaxDuration          time.Duration `json:"max_duration"`
	SuccessRate          float64       `json:"success_rate"`
}

// RequestMetrics contains metrics for individual requests
type RequestMetrics struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	JobID     string        `json:"job_id,omitempty"`
	Completed bool          `json:"completed"`
	TimedOut  bool          `json:"timed_out"`
	Error     error         `json:"error,omitempty"`
}

// TestStatus represents the status of a test
type TestStatus string

const (
	TestStatusPending   TestStatus = "pending"
	TestStatusRunning   TestStatus = "running"
	TestStatusCompleted TestStatus = "completed"
	TestStatusFailed    TestStatus = "failed"
	TestStatusCancelled TestStatus = "cancelled"
)

// TestPhase represents the phase of a load test
type TestPhase string

const (
	TestPhaseWarmup   TestPhase = "warmup"
	TestPhaseMain     TestPhase = "main"
	TestPhaseCooldown TestPhase = "cooldown"
)

// BenchmarkOperation represents the type of operation to benchmark
type BenchmarkOperation string

const (
	BenchmarkOperationScan  BenchmarkOperation = "scan"
	BenchmarkOperationQuery BenchmarkOperation = "query"
	BenchmarkOperationCache BenchmarkOperation = "cache"
)