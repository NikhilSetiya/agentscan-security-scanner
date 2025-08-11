package security

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SecurityTestResult represents the result of a security test
type SecurityTestResult struct {
	TestName    string                 `json:"test_name"`
	Passed      bool                   `json:"passed"`
	Severity    string                 `json:"severity"` // low, medium, high, critical
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
}

// SecurityTestSuite runs various security tests
type SecurityTestSuite struct {
	baseURL     string
	httpClient  *http.Client
	testResults []SecurityTestResult
}

// NewSecurityTestSuite creates a new security test suite
func NewSecurityTestSuite(baseURL string) *SecurityTestSuite {
	return &SecurityTestSuite{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false, // We want to test TLS properly
				},
			},
		},
		testResults: make([]SecurityTestResult, 0),
	}
}

// RunAllTests runs all security tests
func (sts *SecurityTestSuite) RunAllTests(ctx context.Context) []SecurityTestResult {
	sts.testResults = make([]SecurityTestResult, 0)
	
	// TLS/SSL Tests
	sts.testTLSConfiguration(ctx)
	sts.testTLSCertificate(ctx)
	
	// HTTP Security Headers Tests
	sts.testSecurityHeaders(ctx)
	sts.testCSPHeader(ctx)
	sts.testHSTSHeader(ctx)
	
	// CORS Tests
	sts.testCORSConfiguration(ctx)
	
	// Authentication Tests
	sts.testAuthenticationEndpoints(ctx)
	sts.testJWTSecurity(ctx)
	
	// Input Validation Tests
	sts.testSQLInjection(ctx)
	sts.testXSSProtection(ctx)
	sts.testCSRFProtection(ctx)
	
	// Rate Limiting Tests
	sts.testRateLimiting(ctx)
	
	// Information Disclosure Tests
	sts.testInformationDisclosure(ctx)
	sts.testErrorHandling(ctx)
	
	return sts.testResults
}

// testTLSConfiguration tests TLS configuration
func (sts *SecurityTestSuite) testTLSConfiguration(ctx context.Context) {
	parsedURL, err := url.Parse(sts.baseURL)
	if err != nil {
		sts.addResult("TLS Configuration", false, "high", "Invalid base URL", nil, "Fix base URL configuration")
		return
	}
	
	if parsedURL.Scheme != "https" {
		sts.addResult("TLS Configuration", false, "critical", "HTTPS not enforced", 
			map[string]interface{}{"scheme": parsedURL.Scheme}, 
			"Enforce HTTPS for all connections")
		return
	}
	
	// Test TLS connection
	conn, err := tls.Dial("tcp", parsedURL.Host, &tls.Config{})
	if err != nil {
		sts.addResult("TLS Configuration", false, "critical", "TLS connection failed", 
			map[string]interface{}{"error": err.Error()}, 
			"Fix TLS configuration")
		return
	}
	defer conn.Close()
	
	state := conn.ConnectionState()
	
	// Check TLS version
	if state.Version < tls.VersionTLS12 {
		sts.addResult("TLS Version", false, "high", "Weak TLS version", 
			map[string]interface{}{"version": state.Version}, 
			"Use TLS 1.2 or higher")
	} else {
		sts.addResult("TLS Version", true, "info", "Strong TLS version", 
			map[string]interface{}{"version": state.Version}, "")
	}
	
	// Check cipher suite
	weakCiphers := []uint16{
		tls.TLS_RSA_WITH_RC4_128_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	}
	
	for _, weak := range weakCiphers {
		if state.CipherSuite == weak {
			sts.addResult("TLS Cipher Suite", false, "medium", "Weak cipher suite", 
				map[string]interface{}{"cipher": state.CipherSuite}, 
				"Use strong cipher suites")
			return
		}
	}
	
	sts.addResult("TLS Cipher Suite", true, "info", "Strong cipher suite", 
		map[string]interface{}{"cipher": state.CipherSuite}, "")
}

// testTLSCertificate tests TLS certificate
func (sts *SecurityTestSuite) testTLSCertificate(ctx context.Context) {
	resp, err := sts.httpClient.Get(sts.baseURL)
	if err != nil {
		sts.addResult("TLS Certificate", false, "high", "Certificate validation failed", 
			map[string]interface{}{"error": err.Error()}, 
			"Fix TLS certificate issues")
		return
	}
	defer resp.Body.Close()
	
	if resp.TLS == nil {
		sts.addResult("TLS Certificate", false, "critical", "No TLS connection", nil, 
			"Enable HTTPS")
		return
	}
	
	cert := resp.TLS.PeerCertificates[0]
	
	// Check certificate expiration
	now := time.Now()
	if cert.NotAfter.Before(now) {
		sts.addResult("Certificate Expiration", false, "critical", "Certificate expired", 
			map[string]interface{}{"expires": cert.NotAfter}, 
			"Renew TLS certificate")
	} else if cert.NotAfter.Before(now.Add(30 * 24 * time.Hour)) {
		sts.addResult("Certificate Expiration", false, "medium", "Certificate expires soon", 
			map[string]interface{}{"expires": cert.NotAfter}, 
			"Renew TLS certificate soon")
	} else {
		sts.addResult("Certificate Expiration", true, "info", "Certificate valid", 
			map[string]interface{}{"expires": cert.NotAfter}, "")
	}
}

// testSecurityHeaders tests HTTP security headers
func (sts *SecurityTestSuite) testSecurityHeaders(ctx context.Context) {
	resp, err := sts.httpClient.Get(sts.baseURL)
	if err != nil {
		sts.addResult("Security Headers", false, "medium", "Failed to fetch headers", 
			map[string]interface{}{"error": err.Error()}, 
			"Fix connectivity issues")
		return
	}
	defer resp.Body.Close()
	
	requiredHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "", // Any value is acceptable
		"X-XSS-Protection":       "", // Any value is acceptable
		"Referrer-Policy":        "", // Any value is acceptable
	}
	
	for header, expectedValue := range requiredHeaders {
		value := resp.Header.Get(header)
		if value == "" {
			sts.addResult(fmt.Sprintf("Header: %s", header), false, "medium", 
				"Security header missing", 
				map[string]interface{}{"header": header}, 
				fmt.Sprintf("Add %s header", header))
		} else if expectedValue != "" && value != expectedValue {
			sts.addResult(fmt.Sprintf("Header: %s", header), false, "low", 
				"Security header has unexpected value", 
				map[string]interface{}{"header": header, "value": value, "expected": expectedValue}, 
				fmt.Sprintf("Set %s header to %s", header, expectedValue))
		} else {
			sts.addResult(fmt.Sprintf("Header: %s", header), true, "info", 
				"Security header present", 
				map[string]interface{}{"header": header, "value": value}, "")
		}
	}
}

// testCSPHeader tests Content Security Policy header
func (sts *SecurityTestSuite) testCSPHeader(ctx context.Context) {
	resp, err := sts.httpClient.Get(sts.baseURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		sts.addResult("Content Security Policy", false, "high", 
			"CSP header missing", nil, 
			"Add Content-Security-Policy header")
		return
	}
	
	// Check for unsafe directives
	unsafeDirectives := []string{"'unsafe-inline'", "'unsafe-eval'", "*"}
	for _, unsafe := range unsafeDirectives {
		if strings.Contains(csp, unsafe) {
			sts.addResult("CSP Unsafe Directives", false, "medium", 
				"CSP contains unsafe directive", 
				map[string]interface{}{"directive": unsafe, "csp": csp}, 
				"Remove unsafe CSP directives")
		}
	}
	
	sts.addResult("Content Security Policy", true, "info", 
		"CSP header present", 
		map[string]interface{}{"csp": csp}, "")
}

// testHSTSHeader tests HTTP Strict Transport Security header
func (sts *SecurityTestSuite) testHSTSHeader(ctx context.Context) {
	resp, err := sts.httpClient.Get(sts.baseURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	hsts := resp.Header.Get("Strict-Transport-Security")
	if hsts == "" {
		sts.addResult("HSTS Header", false, "medium", 
			"HSTS header missing", nil, 
			"Add Strict-Transport-Security header")
		return
	}
	
	// Check for minimum max-age
	if !strings.Contains(hsts, "max-age=") {
		sts.addResult("HSTS Max-Age", false, "medium", 
			"HSTS max-age directive missing", 
			map[string]interface{}{"hsts": hsts}, 
			"Add max-age directive to HSTS header")
	} else {
		sts.addResult("HSTS Header", true, "info", 
			"HSTS header present", 
			map[string]interface{}{"hsts": hsts}, "")
	}
}

// testCORSConfiguration tests CORS configuration
func (sts *SecurityTestSuite) testCORSConfiguration(ctx context.Context) {
	// Test preflight request
	req, err := http.NewRequestWithContext(ctx, "OPTIONS", sts.baseURL+"/api/scans", nil)
	if err != nil {
		return
	}
	
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	
	resp, err := sts.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin == "*" {
		sts.addResult("CORS Configuration", false, "medium", 
			"CORS allows all origins", 
			map[string]interface{}{"allow_origin": allowOrigin}, 
			"Restrict CORS to specific origins")
	} else if allowOrigin == "https://evil.com" {
		sts.addResult("CORS Configuration", false, "high", 
			"CORS allows untrusted origin", 
			map[string]interface{}{"allow_origin": allowOrigin}, 
			"Restrict CORS to trusted origins only")
	} else {
		sts.addResult("CORS Configuration", true, "info", 
			"CORS properly configured", 
			map[string]interface{}{"allow_origin": allowOrigin}, "")
	}
}

// testAuthenticationEndpoints tests authentication security
func (sts *SecurityTestSuite) testAuthenticationEndpoints(ctx context.Context) {
	// Test login endpoint without credentials
	resp, err := http.Post(sts.baseURL+"/api/auth/login", "application/json", 
		strings.NewReader(`{}`))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		sts.addResult("Authentication Required", false, "critical", 
			"Login succeeds without credentials", nil, 
			"Require proper authentication")
	} else {
		sts.addResult("Authentication Required", true, "info", 
			"Login properly requires credentials", nil, "")
	}
}

// testJWTSecurity tests JWT token security
func (sts *SecurityTestSuite) testJWTSecurity(ctx context.Context) {
	// Test with invalid JWT
	req, err := http.NewRequestWithContext(ctx, "GET", sts.baseURL+"/api/scans", nil)
	if err != nil {
		return
	}
	
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	
	resp, err := sts.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		sts.addResult("JWT Validation", false, "critical", 
			"Invalid JWT accepted", nil, 
			"Implement proper JWT validation")
	} else {
		sts.addResult("JWT Validation", true, "info", 
			"Invalid JWT properly rejected", nil, "")
	}
}

// testSQLInjection tests for SQL injection vulnerabilities
func (sts *SecurityTestSuite) testSQLInjection(ctx context.Context) {
	sqlPayloads := []string{
		"' OR '1'='1",
		"'; DROP TABLE users; --",
		"' UNION SELECT * FROM users --",
	}
	
	for _, payload := range sqlPayloads {
		// Test in query parameter
		testURL := fmt.Sprintf("%s/api/scans?search=%s", sts.baseURL, url.QueryEscape(payload))
		resp, err := sts.httpClient.Get(testURL)
		if err != nil {
			continue
		}
		resp.Body.Close()
		
		// Check for SQL error messages in response
		if resp.StatusCode == http.StatusInternalServerError {
			sts.addResult("SQL Injection Protection", false, "critical", 
				"Potential SQL injection vulnerability", 
				map[string]interface{}{"payload": payload}, 
				"Implement proper input validation and parameterized queries")
			return
		}
	}
	
	sts.addResult("SQL Injection Protection", true, "info", 
		"No SQL injection vulnerabilities detected", nil, "")
}

// testXSSProtection tests for XSS vulnerabilities
func (sts *SecurityTestSuite) testXSSProtection(ctx context.Context) {
	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"<img src=x onerror=alert('xss')>",
	}
	
	for _, payload := range xssPayloads {
		// Test in query parameter
		testURL := fmt.Sprintf("%s/api/scans?name=%s", sts.baseURL, url.QueryEscape(payload))
		resp, err := sts.httpClient.Get(testURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		// This is a basic test - in practice, you'd check response content
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "text/html") && resp.StatusCode == http.StatusOK {
			sts.addResult("XSS Protection", false, "high", 
				"Potential XSS vulnerability", 
				map[string]interface{}{"payload": payload}, 
				"Implement proper output encoding and CSP")
			return
		}
	}
	
	sts.addResult("XSS Protection", true, "info", 
		"No XSS vulnerabilities detected", nil, "")
}

// testCSRFProtection tests CSRF protection
func (sts *SecurityTestSuite) testCSRFProtection(ctx context.Context) {
	// Test POST request without CSRF token
	resp, err := http.Post(sts.baseURL+"/api/scans", "application/json", 
		strings.NewReader(`{"repository_url": "https://github.com/test/repo"}`))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	// Check if request is accepted without CSRF protection
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		sts.addResult("CSRF Protection", false, "medium", 
			"No CSRF protection detected", nil, 
			"Implement CSRF protection for state-changing operations")
	} else {
		sts.addResult("CSRF Protection", true, "info", 
			"CSRF protection appears to be in place", nil, "")
	}
}

// testRateLimiting tests rate limiting
func (sts *SecurityTestSuite) testRateLimiting(ctx context.Context) {
	// Make multiple rapid requests
	var lastStatusCode int
	for i := 0; i < 10; i++ {
		resp, err := sts.httpClient.Get(sts.baseURL + "/api/health")
		if err != nil {
			continue
		}
		lastStatusCode = resp.StatusCode
		resp.Body.Close()
		
		if resp.StatusCode == http.StatusTooManyRequests {
			sts.addResult("Rate Limiting", true, "info", 
				"Rate limiting is active", nil, "")
			return
		}
	}
	
	if lastStatusCode == http.StatusOK {
		sts.addResult("Rate Limiting", false, "medium", 
			"No rate limiting detected", nil, 
			"Implement rate limiting to prevent abuse")
	}
}

// testInformationDisclosure tests for information disclosure
func (sts *SecurityTestSuite) testInformationDisclosure(ctx context.Context) {
	// Test server header
	resp, err := sts.httpClient.Get(sts.baseURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	server := resp.Header.Get("Server")
	if strings.Contains(strings.ToLower(server), "apache") || 
	   strings.Contains(strings.ToLower(server), "nginx") ||
	   strings.Contains(strings.ToLower(server), "iis") {
		sts.addResult("Server Information Disclosure", false, "low", 
			"Server header reveals server software", 
			map[string]interface{}{"server": server}, 
			"Remove or obfuscate server header")
	} else {
		sts.addResult("Server Information Disclosure", true, "info", 
			"Server header properly configured", 
			map[string]interface{}{"server": server}, "")
	}
}

// testErrorHandling tests error handling security
func (sts *SecurityTestSuite) testErrorHandling(ctx context.Context) {
	// Test 404 error
	resp, err := sts.httpClient.Get(sts.baseURL + "/nonexistent-endpoint")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		sts.addResult("Error Handling", true, "info", 
			"Proper 404 error handling", nil, "")
	} else {
		sts.addResult("Error Handling", false, "low", 
			"Unexpected error response", 
			map[string]interface{}{"status_code": resp.StatusCode}, 
			"Implement proper error handling")
	}
}

// addResult adds a test result to the suite
func (sts *SecurityTestSuite) addResult(testName string, passed bool, severity, description string, details map[string]interface{}, remediation string) {
	result := SecurityTestResult{
		TestName:    testName,
		Passed:      passed,
		Severity:    severity,
		Description: description,
		Details:     details,
		Remediation: remediation,
	}
	sts.testResults = append(sts.testResults, result)
}

// GetResults returns all test results
func (sts *SecurityTestSuite) GetResults() []SecurityTestResult {
	return sts.testResults
}

// GetFailedTests returns only failed test results
func (sts *SecurityTestSuite) GetFailedTests() []SecurityTestResult {
	var failed []SecurityTestResult
	for _, result := range sts.testResults {
		if !result.Passed {
			failed = append(failed, result)
		}
	}
	return failed
}

// GetTestsBySeveity returns test results filtered by severity
func (sts *SecurityTestSuite) GetTestsBySeverity(severity string) []SecurityTestResult {
	var filtered []SecurityTestResult
	for _, result := range sts.testResults {
		if result.Severity == severity {
			filtered = append(filtered, result)
		}
	}
	return filtered
}