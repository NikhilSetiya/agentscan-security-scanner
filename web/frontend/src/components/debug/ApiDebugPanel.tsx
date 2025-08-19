/**
 * API Debug Panel Component
 * Provides debugging tools for API connectivity issues
 */

import React, { useState, useEffect } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '../ui/Card'
import { Button } from '../ui/Button'
import { apiHealthChecker, HealthCheckResult } from '../../utils/apiHealthCheck'
import { environmentValidator } from '../../utils/environmentValidator'
import { observeLogger } from '../../services/observeLogger'
import { apiClient } from '../../services/api'
import { 
  Activity, 
  AlertCircle, 
  CheckCircle, 
  Clock, 
  RefreshCw, 
  Wifi, 
  WifiOff,
  Settings,
  Bug
} from 'lucide-react'

interface ApiDebugPanelProps {
  isOpen: boolean
  onClose: () => void
}

export const ApiDebugPanel: React.FC<ApiDebugPanelProps> = ({ isOpen, onClose }) => {
  const [healthResults, setHealthResults] = useState<HealthCheckResult[]>([])
  const [corsResult, setCorsResult] = useState<HealthCheckResult | null>(null)
  const [isRunning, setIsRunning] = useState(false)
  const [lastCheck, setLastCheck] = useState<string>('')
  const [apiInfo, setApiInfo] = useState<any>(null)
  const [envValidation, setEnvValidation] = useState<any>(null)

  useEffect(() => {
    if (isOpen) {
      runHealthCheck()
      fetchApiInfo()
      validateEnvironment()
    }
  }, [isOpen])

  const validateEnvironment = () => {
    const validation = environmentValidator.validate()
    setEnvValidation(validation)
    
    observeLogger.logEvent(
      validation.isValid ? 'info' : 'warn',
      'Environment validation in debug panel',
      {
        isValid: validation.isValid,
        errorCount: validation.errors.length,
        warningCount: validation.warnings.length
      }
    )
  }

  const runHealthCheck = async () => {
    setIsRunning(true)
    observeLogger.logUserAction('debug_health_check_started', {
      component: 'ApiDebugPanel'
    })

    try {
      const [healthResults, corsResult] = await Promise.all([
        apiHealthChecker.performHealthCheck(),
        apiHealthChecker.testCORS()
      ])

      setHealthResults(healthResults)
      setCorsResult(corsResult)
      setLastCheck(new Date().toLocaleString())

      // Log summary to Observe
      const successCount = healthResults.filter(r => r.success).length + (corsResult.success ? 1 : 0)
      const totalCount = healthResults.length + 1
      
      observeLogger.logEvent('info', 'Health check completed', {
        success_rate: successCount / totalCount,
        total_tests: totalCount,
        successful_tests: successCount,
        type: 'health_check_summary'
      })

    } catch (error) {
      observeLogger.logError(error as Error, {
        component: 'ApiDebugPanel',
        action: 'health_check'
      })
    } finally {
      setIsRunning(false)
    }
  }

  const fetchApiInfo = async () => {
    try {
      const response = await apiClient.healthCheck()
      if (response.data) {
        setApiInfo(response.data)
      }
    } catch (error) {
      console.warn('Failed to fetch API info:', error)
    }
  }

  const copyReport = () => {
    if (!corsResult) return

    const healthReport = apiHealthChecker.generateReport(healthResults, corsResult)
    const envReport = environmentValidator.generateReport()
    const fullReport = `${healthReport}\n\n${envReport}`
    
    navigator.clipboard.writeText(fullReport).then(() => {
      observeLogger.logUserAction('debug_report_copied', {
        component: 'ApiDebugPanel'
      })
      alert('Complete report copied to clipboard!')
    })
  }

  const getStatusIcon = (success: boolean, isRunning: boolean = false) => {
    if (isRunning) return <Clock className="w-4 h-4 animate-spin text-blue-500" />
    return success 
      ? <CheckCircle className="w-4 h-4 text-green-500" />
      : <AlertCircle className="w-4 h-4 text-red-500" />
  }

  const getStatusColor = (success: boolean) => {
    return success ? 'text-green-600' : 'text-red-600'
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
        <div className="flex items-center justify-between p-6 border-b">
          <div className="flex items-center gap-2">
            <Bug className="w-5 h-5 text-blue-600" />
            <h2 className="text-xl font-semibold">API Debug Panel</h2>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={runHealthCheck}
              loading={isRunning}
              icon={<RefreshCw size={16} />}
            >
              Run Check
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={onClose}
            >
              ✕
            </Button>
          </div>
        </div>

        <div className="p-6 overflow-y-auto max-h-[calc(90vh-120px)]">
          {/* Environment Validation */}
          {envValidation && (
            <Card className="mb-6">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  {envValidation.isValid 
                    ? <CheckCircle className="w-4 h-4 text-green-500" />
                    : <AlertCircle className="w-4 h-4 text-red-500" />
                  }
                  Environment Configuration
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {envValidation.errors.length > 0 && (
                    <div className="p-3 bg-red-50 border border-red-200 rounded">
                      <div className="font-medium text-red-800 mb-2">Errors ({envValidation.errors.length})</div>
                      {envValidation.errors.map((error: string, index: number) => (
                        <div key={index} className="text-sm text-red-600">• {error}</div>
                      ))}
                    </div>
                  )}
                  
                  {envValidation.warnings.length > 0 && (
                    <div className="p-3 bg-yellow-50 border border-yellow-200 rounded">
                      <div className="font-medium text-yellow-800 mb-2">Warnings ({envValidation.warnings.length})</div>
                      {envValidation.warnings.map((warning: string, index: number) => (
                        <div key={index} className="text-sm text-yellow-600">• {warning}</div>
                      ))}
                    </div>
                  )}

                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <strong>API Base URL:</strong>
                      <div className="font-mono text-xs bg-gray-100 p-2 rounded mt-1">
                        {envValidation.config.VITE_API_BASE_URL || 'Not configured'}
                      </div>
                    </div>
                    <div>
                      <strong>Environment:</strong>
                      <div className="font-mono text-xs bg-gray-100 p-2 rounded mt-1">
                        {envValidation.config.VITE_NODE_ENV || 'development'}
                      </div>
                    </div>
                    <div>
                      <strong>Origin:</strong>
                      <div className="font-mono text-xs bg-gray-100 p-2 rounded mt-1">
                        {window.location.origin}
                      </div>
                    </div>
                    <div>
                      <strong>Last Check:</strong>
                      <div className="font-mono text-xs bg-gray-100 p-2 rounded mt-1">
                        {lastCheck || 'Never'}
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Health Check Results */}
          <Card className="mb-6">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Activity className="w-4 h-4" />
                Connectivity Tests
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {healthResults.map((result, index) => (
                  <div key={index} className="flex items-center justify-between p-3 border rounded">
                    <div className="flex items-center gap-3">
                      {getStatusIcon(result.success, isRunning)}
                      <div>
                        <div className="font-medium">{result.details?.url}</div>
                        <div className={`text-sm ${getStatusColor(result.success)}`}>
                          {result.message}
                        </div>
                      </div>
                    </div>
                    <div className="text-right text-sm text-gray-500">
                      <div>{result.responseTime}ms</div>
                      <div>Status: {result.status}</div>
                    </div>
                  </div>
                ))}

                {corsResult && (
                  <div className="flex items-center justify-between p-3 border rounded">
                    <div className="flex items-center gap-3">
                      {corsResult.success ? <Wifi className="w-4 h-4 text-green-500" /> : <WifiOff className="w-4 h-4 text-red-500" />}
                      <div>
                        <div className="font-medium">CORS Configuration</div>
                        <div className={`text-sm ${getStatusColor(corsResult.success)}`}>
                          {corsResult.message}
                        </div>
                      </div>
                    </div>
                    <div className="text-right text-sm text-gray-500">
                      <div>{corsResult.responseTime}ms</div>
                      <div>Status: {corsResult.status}</div>
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* API Information */}
          {apiInfo && (
            <Card className="mb-6">
              <CardHeader>
                <CardTitle>Backend Information</CardTitle>
              </CardHeader>
              <CardContent>
                <pre className="text-xs bg-gray-100 p-3 rounded overflow-x-auto">
                  {JSON.stringify(apiInfo, null, 2)}
                </pre>
              </CardContent>
            </Card>
          )}

          {/* Error Details */}
          {(healthResults.some(r => !r.success) || (corsResult && !corsResult.success)) && (
            <Card className="mb-6">
              <CardHeader>
                <CardTitle className="text-red-600">Issues Detected</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {healthResults.filter(r => !r.success).map((result, index) => (
                    <div key={index} className="p-3 bg-red-50 border border-red-200 rounded">
                      <div className="font-medium text-red-800">{result.details?.url}</div>
                      <div className="text-sm text-red-600 mt-1">{result.message}</div>
                      {result.details?.error && (
                        <div className="text-xs text-red-500 mt-2 font-mono">
                          {result.details.error}
                        </div>
                      )}
                    </div>
                  ))}
                  
                  {corsResult && !corsResult.success && (
                    <div className="p-3 bg-red-50 border border-red-200 rounded">
                      <div className="font-medium text-red-800">CORS Configuration</div>
                      <div className="text-sm text-red-600 mt-1">{corsResult.message}</div>
                      <div className="text-xs text-red-500 mt-2">
                        Check that your domain is allowed in the backend CORS configuration
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Actions */}
          <div className="flex gap-3">
            <Button
              variant="primary"
              onClick={runHealthCheck}
              loading={isRunning}
              icon={<RefreshCw size={16} />}
            >
              Run Health Check
            </Button>
            <Button
              variant="secondary"
              onClick={copyReport}
              disabled={!corsResult}
            >
              Copy Report
            </Button>
            <Button
              variant="secondary"
              onClick={() => {
                observeLogger.logUserAction('debug_observe_test', {
                  component: 'ApiDebugPanel'
                })
                alert('Test event sent to Observe!')
              }}
            >
              Test Observe
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default ApiDebugPanel