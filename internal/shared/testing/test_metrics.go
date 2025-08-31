package testing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TestMetrics tracks test execution metrics
type TestMetrics struct {
	mu                sync.RWMutex
	TestRuns          []TestRun          `json:"test_runs"`
	CoverageHistory   []CoverageReport   `json:"coverage_history"`
	PerformanceData   []PerformancePoint `json:"performance_data"`
	ReliabilityStats  ReliabilityStats   `json:"reliability_stats"`
	metricsFile       string
}

// TestRun represents a single test execution
type TestRun struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	Duration     time.Duration     `json:"duration"`
	TestType     string            `json:"test_type"` // unit, integration, e2e
	Status       string            `json:"status"`    // passed, failed, skipped
	TestCount    int               `json:"test_count"`
	PassedCount  int               `json:"passed_count"`
	FailedCount  int               `json:"failed_count"`
	SkippedCount int               `json:"skipped_count"`
	Coverage     float64           `json:"coverage"`
	Packages     []PackageResult   `json:"packages"`
	Environment  map[string]string `json:"environment"`
	GitCommit    string            `json:"git_commit,omitempty"`
	Branch       string            `json:"branch,omitempty"`
}

// PackageResult represents test results for a single package
type PackageResult struct {
	Package      string        `json:"package"`
	Duration     time.Duration `json:"duration"`
	TestCount    int           `json:"test_count"`
	PassedCount  int           `json:"passed_count"`
	FailedCount  int           `json:"failed_count"`
	SkippedCount int           `json:"skipped_count"`
	Coverage     float64       `json:"coverage"`
	FailedTests  []string      `json:"failed_tests,omitempty"`
}

// CoverageReport represents coverage data over time
type CoverageReport struct {
	Timestamp      time.Time            `json:"timestamp"`
	OverallCoverage float64             `json:"overall_coverage"`
	PackageCoverage map[string]float64  `json:"package_coverage"`
	GitCommit      string               `json:"git_commit,omitempty"`
	Branch         string               `json:"branch,omitempty"`
}

// PerformancePoint represents a performance measurement
type PerformancePoint struct {
	Timestamp   time.Time `json:"timestamp"`
	TestType    string    `json:"test_type"`
	Duration    time.Duration `json:"duration"`
	TestCount   int       `json:"test_count"`
	Throughput  float64   `json:"throughput"` // tests per second
	GitCommit   string    `json:"git_commit,omitempty"`
}

// ReliabilityStats tracks test reliability over time
type ReliabilityStats struct {
	TotalRuns       int     `json:"total_runs"`
	SuccessfulRuns  int     `json:"successful_runs"`
	FailedRuns      int     `json:"failed_runs"`
	FlakyTests      []FlakyTest `json:"flaky_tests"`
	SuccessRate     float64 `json:"success_rate"`
	LastUpdated     time.Time `json:"last_updated"`
}

// FlakyTest represents a test that fails intermittently
type FlakyTest struct {
	TestName     string    `json:"test_name"`
	Package      string    `json:"package"`
	FailureCount int       `json:"failure_count"`
	TotalRuns    int       `json:"total_runs"`
	FailureRate  float64   `json:"failure_rate"`
	LastFailure  time.Time `json:"last_failure"`
	Errors       []string  `json:"errors"`
}

// NewTestMetrics creates a new test metrics tracker
func NewTestMetrics(metricsFile string) *TestMetrics {
	tm := &TestMetrics{
		TestRuns:        make([]TestRun, 0),
		CoverageHistory: make([]CoverageReport, 0),
		PerformanceData: make([]PerformancePoint, 0),
		ReliabilityStats: ReliabilityStats{
			FlakyTests: make([]FlakyTest, 0),
		},
		metricsFile: metricsFile,
	}
	
	// Load existing metrics if file exists
	tm.LoadMetrics()
	
	return tm
}

// RecordTestRun records a test execution
func (tm *TestMetrics) RecordTestRun(run TestRun) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	// Set ID if not provided
	if run.ID == "" {
		run.ID = fmt.Sprintf("run_%d", time.Now().Unix())
	}
	
	// Set timestamp if not provided
	if run.Timestamp.IsZero() {
		run.Timestamp = time.Now()
	}
	
	// Add git information if available
	if run.GitCommit == "" {
		run.GitCommit = tm.getGitCommit()
	}
	if run.Branch == "" {
		run.Branch = tm.getGitBranch()
	}
	
	tm.TestRuns = append(tm.TestRuns, run)
	
	// Update reliability stats
	tm.updateReliabilityStats(run)
	
	// Record performance data
	if run.Duration > 0 && run.TestCount > 0 {
		throughput := float64(run.TestCount) / run.Duration.Seconds()
		tm.PerformanceData = append(tm.PerformanceData, PerformancePoint{
			Timestamp:  run.Timestamp,
			TestType:   run.TestType,
			Duration:   run.Duration,
			TestCount:  run.TestCount,
			Throughput: throughput,
			GitCommit:  run.GitCommit,
		})
	}
	
	// Record coverage if available
	if run.Coverage > 0 {
		packageCoverage := make(map[string]float64)
		for _, pkg := range run.Packages {
			packageCoverage[pkg.Package] = pkg.Coverage
		}
		
		tm.CoverageHistory = append(tm.CoverageHistory, CoverageReport{
			Timestamp:       run.Timestamp,
			OverallCoverage: run.Coverage,
			PackageCoverage: packageCoverage,
			GitCommit:       run.GitCommit,
			Branch:          run.Branch,
		})
	}
	
	// Keep only last 1000 runs to prevent unbounded growth
	if len(tm.TestRuns) > 1000 {
		tm.TestRuns = tm.TestRuns[len(tm.TestRuns)-1000:]
	}
	
	// Save metrics
	tm.saveMetrics()
}

// updateReliabilityStats updates reliability statistics
func (tm *TestMetrics) updateReliabilityStats(run TestRun) {
	tm.ReliabilityStats.TotalRuns++
	
	if run.Status == "passed" {
		tm.ReliabilityStats.SuccessfulRuns++
	} else {
		tm.ReliabilityStats.FailedRuns++
		
		// Track flaky tests
		for _, pkg := range run.Packages {
			for _, failedTest := range pkg.FailedTests {
				tm.updateFlakyTest(failedTest, pkg.Package)
			}
		}
	}
	
	// Calculate success rate
	if tm.ReliabilityStats.TotalRuns > 0 {
		tm.ReliabilityStats.SuccessRate = float64(tm.ReliabilityStats.SuccessfulRuns) / float64(tm.ReliabilityStats.TotalRuns)
	}
	
	tm.ReliabilityStats.LastUpdated = time.Now()
}

// updateFlakyTest updates flaky test statistics
func (tm *TestMetrics) updateFlakyTest(testName, packageName string) {
	for i, flaky := range tm.ReliabilityStats.FlakyTests {
		if flaky.TestName == testName && flaky.Package == packageName {
			tm.ReliabilityStats.FlakyTests[i].FailureCount++
			tm.ReliabilityStats.FlakyTests[i].TotalRuns++
			tm.ReliabilityStats.FlakyTests[i].FailureRate = float64(flaky.FailureCount) / float64(flaky.TotalRuns)
			tm.ReliabilityStats.FlakyTests[i].LastFailure = time.Now()
			return
		}
	}
	
	// New flaky test
	tm.ReliabilityStats.FlakyTests = append(tm.ReliabilityStats.FlakyTests, FlakyTest{
		TestName:     testName,
		Package:      packageName,
		FailureCount: 1,
		TotalRuns:    1,
		FailureRate:  1.0,
		LastFailure:  time.Now(),
		Errors:       make([]string, 0),
	})
}

// GetCoverageTrend returns coverage trend over time
func (tm *TestMetrics) GetCoverageTrend(days int) []CoverageReport {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	cutoff := time.Now().AddDate(0, 0, -days)
	var trend []CoverageReport
	
	for _, report := range tm.CoverageHistory {
		if report.Timestamp.After(cutoff) {
			trend = append(trend, report)
		}
	}
	
	return trend
}

// GetPerformanceTrend returns performance trend over time
func (tm *TestMetrics) GetPerformanceTrend(testType string, days int) []PerformancePoint {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	cutoff := time.Now().AddDate(0, 0, -days)
	var trend []PerformancePoint
	
	for _, point := range tm.PerformanceData {
		if point.Timestamp.After(cutoff) && (testType == "" || point.TestType == testType) {
			trend = append(trend, point)
		}
	}
	
	return trend
}

// GetFlakyTests returns tests that fail intermittently
func (tm *TestMetrics) GetFlakyTests(minFailureRate float64) []FlakyTest {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	var flaky []FlakyTest
	for _, test := range tm.ReliabilityStats.FlakyTests {
		if test.FailureRate >= minFailureRate && test.TotalRuns >= 5 {
			flaky = append(flaky, test)
		}
	}
	
	return flaky
}

// GetTestSummary returns a summary of recent test runs
func (tm *TestMetrics) GetTestSummary(days int) TestSummary {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	cutoff := time.Now().AddDate(0, 0, -days)
	
	summary := TestSummary{
		Period:    fmt.Sprintf("Last %d days", days),
		StartDate: cutoff,
		EndDate:   time.Now(),
	}
	
	var totalDuration time.Duration
	var totalTests int
	
	for _, run := range tm.TestRuns {
		if run.Timestamp.After(cutoff) {
			summary.TotalRuns++
			totalDuration += run.Duration
			totalTests += run.TestCount
			
			if run.Status == "passed" {
				summary.SuccessfulRuns++
			} else {
				summary.FailedRuns++
			}
			
			if run.Coverage > summary.MaxCoverage {
				summary.MaxCoverage = run.Coverage
			}
			if summary.MinCoverage == 0 || run.Coverage < summary.MinCoverage {
				summary.MinCoverage = run.Coverage
			}
			summary.TotalCoverage += run.Coverage
		}
	}
	
	if summary.TotalRuns > 0 {
		summary.SuccessRate = float64(summary.SuccessfulRuns) / float64(summary.TotalRuns)
		summary.AverageDuration = totalDuration / time.Duration(summary.TotalRuns)
		summary.AverageCoverage = summary.TotalCoverage / float64(summary.TotalRuns)
		summary.AverageTestsPerRun = float64(totalTests) / float64(summary.TotalRuns)
	}
	
	return summary
}

// TestSummary represents a summary of test metrics
type TestSummary struct {
	Period             string        `json:"period"`
	StartDate          time.Time     `json:"start_date"`
	EndDate            time.Time     `json:"end_date"`
	TotalRuns          int           `json:"total_runs"`
	SuccessfulRuns     int           `json:"successful_runs"`
	FailedRuns         int           `json:"failed_runs"`
	SuccessRate        float64       `json:"success_rate"`
	AverageDuration    time.Duration `json:"average_duration"`
	AverageCoverage    float64       `json:"average_coverage"`
	MinCoverage        float64       `json:"min_coverage"`
	MaxCoverage        float64       `json:"max_coverage"`
	TotalCoverage      float64       `json:"total_coverage"`
	AverageTestsPerRun float64       `json:"average_tests_per_run"`
}

// LoadMetrics loads metrics from file
func (tm *TestMetrics) LoadMetrics() error {
	if tm.metricsFile == "" {
		return nil
	}
	
	if _, err := os.Stat(tm.metricsFile); os.IsNotExist(err) {
		return nil // File doesn't exist, start fresh
	}
	
	data, err := os.ReadFile(tm.metricsFile)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, tm)
}

// saveMetrics saves metrics to file
func (tm *TestMetrics) saveMetrics() error {
	if tm.metricsFile == "" {
		return nil
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(tm.metricsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(tm, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(tm.metricsFile, data, 0644)
}

// ExportMetrics exports metrics to a file
func (tm *TestMetrics) ExportMetrics(filename string) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	data, err := json.MarshalIndent(tm, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}

// GenerateReport generates a human-readable test report
func (tm *TestMetrics) GenerateReport(days int) string {
	summary := tm.GetTestSummary(days)
	coverageTrend := tm.GetCoverageTrend(days)
	flakyTests := tm.GetFlakyTests(0.1) // 10% failure rate threshold
	
	report := fmt.Sprintf(`
# Test Metrics Report

## Summary (%s)
- **Total Runs**: %d
- **Success Rate**: %.2f%%
- **Average Duration**: %v
- **Average Coverage**: %.2f%%
- **Coverage Range**: %.2f%% - %.2f%%

## Reliability
- **Successful Runs**: %d
- **Failed Runs**: %d
- **Flaky Tests**: %d

## Recent Coverage Trend
`,
		summary.Period,
		summary.TotalRuns,
		summary.SuccessRate*100,
		summary.AverageDuration,
		summary.AverageCoverage,
		summary.MinCoverage,
		summary.MaxCoverage,
		summary.SuccessfulRuns,
		summary.FailedRuns,
		len(flakyTests),
	)
	
	// Add coverage trend
	if len(coverageTrend) > 0 {
		report += "| Date | Coverage |\n|------|----------|\n"
		for _, point := range coverageTrend[max(0, len(coverageTrend)-10):] {
			report += fmt.Sprintf("| %s | %.2f%% |\n", 
				point.Timestamp.Format("2006-01-02"), 
				point.OverallCoverage)
		}
	}
	
	// Add flaky tests
	if len(flakyTests) > 0 {
		report += "\n## Flaky Tests\n"
		report += "| Test | Package | Failure Rate | Last Failure |\n"
		report += "|------|---------|--------------|-------------|\n"
		for _, test := range flakyTests {
			report += fmt.Sprintf("| %s | %s | %.2f%% | %s |\n",
				test.TestName,
				test.Package,
				test.FailureRate*100,
				test.LastFailure.Format("2006-01-02"))
		}
	}
	
	return report
}

// Helper functions

func (tm *TestMetrics) getGitCommit() string {
	// Try to get git commit hash
	// This is a simplified implementation
	return os.Getenv("GIT_COMMIT")
}

func (tm *TestMetrics) getGitBranch() string {
	// Try to get git branch
	// This is a simplified implementation
	branch := os.Getenv("GIT_BRANCH")
	if branch == "" {
		branch = "main"
	}
	return branch
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Global test metrics instance
var globalTestMetrics *TestMetrics

// InitGlobalTestMetrics initializes the global test metrics
func InitGlobalTestMetrics(metricsFile string) {
	globalTestMetrics = NewTestMetrics(metricsFile)
}

// RecordGlobalTestRun records a test run to the global metrics
func RecordGlobalTestRun(run TestRun) {
	if globalTestMetrics != nil {
		globalTestMetrics.RecordTestRun(run)
	}
}

// GetGlobalTestMetrics returns the global test metrics instance
func GetGlobalTestMetrics() *TestMetrics {
	return globalTestMetrics
}