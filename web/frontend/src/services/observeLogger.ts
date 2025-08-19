/**
 * Observe MCP Integration for Frontend Debugging
 * Provides comprehensive logging and monitoring through Observe MCP
 */

interface ObserveConfig {
  endpoint: string
  apiKey: string
  projectId: string
  environment: 'development' | 'staging' | 'production'
  enabled: boolean
}

interface LogEvent {
  timestamp: string
  level: 'debug' | 'info' | 'warn' | 'error'
  message: string
  context: Record<string, any>
  userId?: string
  sessionId?: string
  traceId?: string
}

interface ApiCallEvent {
  method: string
  url: string
  status: number
  duration: number
  requestId?: string
  userId?: string
  error?: string
  requestBody?: any
  responseBody?: any
}

interface ErrorEvent {
  error: Error
  context: Record<string, any>
  userId?: string
  sessionId?: string
  stack?: string
  componentStack?: string
}

interface UserActionEvent {
  action: string
  userId: string
  metadata: Record<string, any>
  timestamp: string
}

class ObserveLogger {
  private config: ObserveConfig
  private sessionId: string
  private userId?: string
  private traceId?: string

  constructor(config: ObserveConfig) {
    this.config = config
    this.sessionId = this.generateSessionId()
    
    // Initialize user ID from auth context if available
    this.initializeUser()
  }

  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  private generateTraceId(): string {
    return `trace_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  private initializeUser(): void {
    // Try to get user from localStorage or auth context
    try {
      const authToken = localStorage.getItem('auth_token')
      if (authToken) {
        // Extract user ID from token (simplified)
        const payload = JSON.parse(atob(authToken.split('.')[1]))
        this.userId = payload.sub || payload.user_id
      }
    } catch (error) {
      console.warn('[OBSERVE] Failed to initialize user:', error)
    }
  }

  /**
   * Set current user ID for logging context
   */
  setUserId(userId: string): void {
    this.userId = userId
  }

  /**
   * Create a new trace for operation tracking
   */
  createTrace(operationName: string): string {
    const traceId = this.generateTraceId()
    this.traceId = traceId

    if (!this.config.enabled) return traceId

    this.logEvent('info', `Started trace: ${operationName}`, {
      traceId,
      operationName,
      type: 'trace_start'
    })

    return traceId
  }

  /**
   * End current trace
   */
  endTrace(traceId: string, success: boolean = true, metadata?: Record<string, any>): void {
    if (!this.config.enabled) return

    this.logEvent('info', `Ended trace: ${traceId}`, {
      traceId,
      success,
      type: 'trace_end',
      ...metadata
    })

    if (this.traceId === traceId) {
      this.traceId = undefined
    }
  }

  /**
   * Log API call with timing and response details
   */
  logApiCall(request: {
    method: string
    url: string
    headers?: Record<string, string>
    body?: any
  }, response: {
    status: number
    headers?: Record<string, string>
    body?: any
    error?: string
  }, duration: number): void {
    if (!this.config.enabled) return

    const event: ApiCallEvent = {
      method: request.method,
      url: request.url,
      status: response.status,
      duration,
      userId: this.userId,
      requestBody: this.sanitizeData(request.body),
      responseBody: this.sanitizeData(response.body),
      error: response.error
    }

    this.sendToObserve('api_call', event)
  }

  /**
   * Log error with context and stack trace
   */
  logError(error: Error, context: Record<string, any> = {}): void {
    if (!this.config.enabled) return

    const errorEvent: ErrorEvent = {
      error: {
        name: error.name,
        message: error.message,
        stack: error.stack
      } as Error,
      context: this.sanitizeData(context),
      userId: this.userId,
      sessionId: this.sessionId,
      stack: error.stack
    }

    this.sendToObserve('error', errorEvent)
    
    // Also log as regular event for easier querying
    this.logEvent('error', error.message, {
      errorName: error.name,
      stack: error.stack,
      ...context
    })
  }

  /**
   * Log user action for behavior tracking
   */
  logUserAction(action: string, metadata: Record<string, any> = {}): void {
    if (!this.config.enabled || !this.userId) return

    const event: UserActionEvent = {
      action,
      userId: this.userId,
      metadata: this.sanitizeData(metadata),
      timestamp: new Date().toISOString()
    }

    this.sendToObserve('user_action', event)
  }

  /**
   * Log general event with level and context
   */
  logEvent(level: LogEvent['level'], message: string, context: Record<string, any> = {}): void {
    if (!this.config.enabled) return

    const event: LogEvent = {
      timestamp: new Date().toISOString(),
      level,
      message,
      context: this.sanitizeData(context),
      userId: this.userId,
      sessionId: this.sessionId,
      traceId: this.traceId
    }

    this.sendToObserve('log', event)

    // Also log to console in development
    if (this.config.environment === 'development') {
      console.log(`[OBSERVE:${level.toUpperCase()}]`, message, context)
    }
  }

  /**
   * Log scan progress for monitoring
   */
  logScanProgress(scanId: string, progress: number, stage: string, metadata?: Record<string, any>): void {
    if (!this.config.enabled) return

    this.logEvent('info', `Scan progress: ${stage}`, {
      scanId,
      progress,
      stage,
      type: 'scan_progress',
      ...metadata
    })
  }

  /**
   * Log performance metrics
   */
  logPerformance(metric: string, value: number, unit: string = 'ms', metadata?: Record<string, any>): void {
    if (!this.config.enabled) return

    this.logEvent('info', `Performance: ${metric}`, {
      metric,
      value,
      unit,
      type: 'performance',
      ...metadata
    })
  }

  /**
   * Send data to Observe MCP
   */
  private async sendToObserve(eventType: string, data: any): Promise<void> {
    try {
      // In a real implementation, this would use the MCP protocol
      // For now, we'll use a REST API call to Observe
      const payload = {
        timestamp: new Date().toISOString(),
        environment: this.config.environment,
        projectId: this.config.projectId,
        eventType,
        data,
        sessionId: this.sessionId,
        userId: this.userId,
        traceId: this.traceId
      }

      // Use fetch to send to Observe endpoint
      await fetch(`${this.config.endpoint}/events`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.config.apiKey}`,
          'X-Observe-Project': this.config.projectId
        },
        body: JSON.stringify(payload)
      }).catch(error => {
        // Don't let logging errors break the application
        console.warn('[OBSERVE] Failed to send event:', error)
      })
    } catch (error) {
      console.warn('[OBSERVE] Failed to send to Observe:', error)
    }
  }

  /**
   * Sanitize data to remove sensitive information
   */
  private sanitizeData(data: any): any {
    if (!data) return data

    const sensitiveKeys = [
      'password', 'token', 'secret', 'key', 'authorization',
      'cookie', 'session', 'auth', 'credential', 'private'
    ]

    const sanitize = (obj: any): any => {
      if (typeof obj !== 'object' || obj === null) return obj
      if (Array.isArray(obj)) return obj.map(sanitize)

      const sanitized: any = {}
      for (const [key, value] of Object.entries(obj)) {
        const lowerKey = key.toLowerCase()
        if (sensitiveKeys.some(sensitive => lowerKey.includes(sensitive))) {
          sanitized[key] = '[REDACTED]'
        } else if (typeof value === 'object') {
          sanitized[key] = sanitize(value)
        } else {
          sanitized[key] = value
        }
      }
      return sanitized
    }

    return sanitize(data)
  }

  /**
   * Create a dashboard for monitoring
   */
  async createDashboard(name: string, queries: Array<{
    name: string
    query: string
    visualization: 'line' | 'bar' | 'table' | 'number'
  }>): Promise<void> {
    if (!this.config.enabled) return

    try {
      await fetch(`${this.config.endpoint}/dashboards`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.config.apiKey}`,
          'X-Observe-Project': this.config.projectId
        },
        body: JSON.stringify({
          name,
          queries,
          projectId: this.config.projectId,
          environment: this.config.environment
        })
      })
    } catch (error) {
      console.warn('[OBSERVE] Failed to create dashboard:', error)
    }
  }
}

// Create and export singleton instance
const observeConfig: ObserveConfig = {
  endpoint: import.meta.env.VITE_OBSERVE_ENDPOINT || 'https://agentscan.observeinc.com/v1',
  apiKey: import.meta.env.VITE_OBSERVE_API_KEY || '',
  projectId: import.meta.env.VITE_OBSERVE_PROJECT_ID || 'agentscan-frontend',
  environment: (import.meta.env.VITE_NODE_ENV as any) || 'development',
  enabled: import.meta.env.VITE_OBSERVE_ENABLED === 'true'
}

export const observeLogger = new ObserveLogger(observeConfig)
export default observeLogger

// Export types for use in other files
export type { LogEvent, ApiCallEvent, ErrorEvent, UserActionEvent, ObserveConfig }