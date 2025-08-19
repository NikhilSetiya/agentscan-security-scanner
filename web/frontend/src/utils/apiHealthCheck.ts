/**
 * API Health Check Utility
 * Tests connectivity between frontend and backend
 */

import { observeLogger } from '../services/observeLogger'

interface HealthCheckResult {
  success: boolean
  status: number
  message: string
  responseTime: number
  timestamp: string
  details?: any
}

interface ConnectivityTest {
  endpoint: string
  method: 'GET' | 'POST'
  expectedStatus: number
  timeout: number
}

class ApiHealthChecker {
  private baseUrl: string

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl.replace(/\/+$/, '') // Remove trailing slashes
  }

  /**
   * Perform comprehensive API health check
   */
  async performHealthCheck(): Promise<HealthCheckResult[]> {
    const tests: ConnectivityTest[] = [
      {
        endpoint: '/health',
        method: 'GET',
        expectedStatus: 200,
        timeout: 5000
      },
      {
        endpoint: '/api/v1',
        method: 'GET',
        expectedStatus: 200,
        timeout: 5000
      },
      {
        endpoint: '/api/v1/auth/login',
        method: 'POST',
        expectedStatus: 400, // Should return 400 for missing credentials
        timeout: 5000
      }
    ]

    const results: HealthCheckResult[] = []

    for (const test of tests) {
      const result = await this.runSingleTest(test)
      results.push(result)
      
      // Log result to Observe
      observeLogger.logEvent(
        result.success ? 'info' : 'error',
        `Health check: ${test.endpoint}`,
        {
          endpoint: test.endpoint,
          method: test.method,
          success: result.success,
          status: result.status,
          responseTime: result.responseTime,
          type: 'health_check'
        }
      )
    }

    return results
  }

  /**
   * Run a single connectivity test
   */
  private async runSingleTest(test: ConnectivityTest): Promise<HealthCheckResult> {
    const startTime = Date.now()
    const url = `${this.baseUrl}${test.endpoint}`

    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), test.timeout)

      const requestOptions: RequestInit = {
        method: test.method,
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json'
        },
        signal: controller.signal
      }

      // Add body for POST requests
      if (test.method === 'POST' && test.endpoint.includes('login')) {
        requestOptions.body = JSON.stringify({
          username: 'test',
          password: 'test'
        })
      }

      const response = await fetch(url, requestOptions)
      clearTimeout(timeoutId)

      const responseTime = Date.now() - startTime
      const success = response.status === test.expectedStatus

      let responseData: any = null
      try {
        const contentType = response.headers.get('content-type')
        if (contentType && contentType.includes('application/json')) {
          responseData = await response.json()
        } else {
          responseData = await response.text()
        }
      } catch (parseError) {
        responseData = { error: 'Failed to parse response' }
      }

      return {
        success,
        status: response.status,
        message: success 
          ? `✅ ${test.endpoint} - OK (${responseTime}ms)`
          : `❌ ${test.endpoint} - Expected ${test.expectedStatus}, got ${response.status}`,
        responseTime,
        timestamp: new Date().toISOString(),
        details: {
          url,
          method: test.method,
          expectedStatus: test.expectedStatus,
          actualStatus: response.status,
          headers: Object.fromEntries(response.headers.entries()),
          body: responseData
        }
      }

    } catch (error) {
      const responseTime = Date.now() - startTime
      
      let errorMessage = 'Unknown error'
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          errorMessage = `Timeout after ${test.timeout}ms`
        } else {
          errorMessage = error.message
        }
      }

      return {
        success: false,
        status: 0,
        message: `❌ ${test.endpoint} - ${errorMessage}`,
        responseTime,
        timestamp: new Date().toISOString(),
        details: {
          url,
          method: test.method,
          error: errorMessage,
          timeout: test.timeout
        }
      }
    }
  }

  /**
   * Test CORS configuration
   */
  async testCORS(): Promise<HealthCheckResult> {
    const startTime = Date.now()
    const url = `${this.baseUrl}/api/v1`

    try {
      // Make a preflight request
      const response = await fetch(url, {
        method: 'OPTIONS',
        headers: {
          'Origin': window.location.origin,
          'Access-Control-Request-Method': 'GET',
          'Access-Control-Request-Headers': 'Content-Type, Authorization'
        }
      })

      const responseTime = Date.now() - startTime
      const corsHeaders = {
        'Access-Control-Allow-Origin': response.headers.get('Access-Control-Allow-Origin'),
        'Access-Control-Allow-Methods': response.headers.get('Access-Control-Allow-Methods'),
        'Access-Control-Allow-Headers': response.headers.get('Access-Control-Allow-Headers'),
        'Access-Control-Allow-Credentials': response.headers.get('Access-Control-Allow-Credentials')
      }

      const success = response.status === 204 || response.status === 200
      
      return {
        success,
        status: response.status,
        message: success 
          ? `✅ CORS - Configured correctly (${responseTime}ms)`
          : `❌ CORS - Configuration issue (status: ${response.status})`,
        responseTime,
        timestamp: new Date().toISOString(),
        details: {
          url,
          method: 'OPTIONS',
          corsHeaders,
          origin: window.location.origin
        }
      }

    } catch (error) {
      const responseTime = Date.now() - startTime
      
      return {
        success: false,
        status: 0,
        message: `❌ CORS - ${error instanceof Error ? error.message : 'Unknown error'}`,
        responseTime,
        timestamp: new Date().toISOString(),
        details: {
          url,
          method: 'OPTIONS',
          error: error instanceof Error ? error.message : 'Unknown error',
          origin: window.location.origin
        }
      }
    }
  }

  /**
   * Generate health check report
   */
  generateReport(results: HealthCheckResult[], corsResult: HealthCheckResult): string {
    const allResults = [...results, corsResult]
    const successCount = allResults.filter(r => r.success).length
    const totalCount = allResults.length
    const avgResponseTime = allResults.reduce((sum, r) => sum + r.responseTime, 0) / totalCount

    let report = `🔍 API Health Check Report\n`
    report += `================================\n`
    report += `Timestamp: ${new Date().toISOString()}\n`
    report += `Success Rate: ${successCount}/${totalCount} (${Math.round(successCount/totalCount*100)}%)\n`
    report += `Average Response Time: ${Math.round(avgResponseTime)}ms\n`
    report += `Base URL: ${this.baseUrl}\n\n`

    report += `Test Results:\n`
    report += `-------------\n`
    allResults.forEach(result => {
      report += `${result.message}\n`
    })

    if (successCount < totalCount) {
      report += `\n⚠️  Issues Detected:\n`
      report += `-------------------\n`
      allResults.filter(r => !r.success).forEach(result => {
        report += `• ${result.details?.url}: ${result.details?.error || 'Status ' + result.status}\n`
      })

      report += `\n🔧 Troubleshooting:\n`
      report += `------------------\n`
      report += `1. Check if backend is running: ${this.baseUrl}/health\n`
      report += `2. Verify CORS configuration allows your domain\n`
      report += `3. Check network connectivity and firewall settings\n`
      report += `4. Verify API endpoint URLs are correct\n`
    }

    return report
  }
}

// Create and export singleton instance
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'
const baseUrl = API_BASE_URL.replace('/api/v1', '') // Remove /api/v1 to get base URL

export const apiHealthChecker = new ApiHealthChecker(baseUrl)
export default apiHealthChecker

// Export types
export type { HealthCheckResult, ConnectivityTest }