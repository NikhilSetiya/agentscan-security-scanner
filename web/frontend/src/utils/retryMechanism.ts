/**
 * Retry Mechanism for API Calls
 * Implements exponential backoff and intelligent retry logic
 */

import { observeLogger } from '../services/observeLogger'

interface RetryOptions {
  maxAttempts: number
  baseDelay: number
  maxDelay: number
  backoffFactor: number
  retryCondition?: (error: any) => boolean
}

interface RetryResult<T> {
  success: boolean
  data?: T
  error?: any
  attempts: number
  totalTime: number
}

const DEFAULT_RETRY_OPTIONS: RetryOptions = {
  maxAttempts: 3,
  baseDelay: 1000, // 1 second
  maxDelay: 10000, // 10 seconds
  backoffFactor: 2,
  retryCondition: (error) => {
    // Retry on network errors, timeouts, and 5xx server errors
    if (error?.name === 'AbortError') return false // Don't retry timeouts
    if (error?.status >= 400 && error?.status < 500) return false // Don't retry client errors
    return true // Retry network errors and server errors
  }
}

/**
 * Retry a function with exponential backoff
 */
export async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  options: Partial<RetryOptions> = {}
): Promise<RetryResult<T>> {
  const opts = { ...DEFAULT_RETRY_OPTIONS, ...options }
  const startTime = Date.now()
  let lastError: any = null

  for (let attempt = 1; attempt <= opts.maxAttempts; attempt++) {
    try {
      observeLogger.logEvent('debug', `Retry attempt ${attempt}/${opts.maxAttempts}`, {
        attempt,
        maxAttempts: opts.maxAttempts,
        type: 'retry_attempt'
      })

      const result = await fn()
      const totalTime = Date.now() - startTime

      observeLogger.logEvent('info', 'Retry succeeded', {
        attempt,
        totalTime,
        type: 'retry_success'
      })

      return {
        success: true,
        data: result,
        attempts: attempt,
        totalTime
      }

    } catch (error) {
      lastError = error
      const totalTime = Date.now() - startTime

      observeLogger.logEvent('warn', `Retry attempt ${attempt} failed`, {
        attempt,
        maxAttempts: opts.maxAttempts,
        error: error instanceof Error ? error.message : 'Unknown error',
        totalTime,
        type: 'retry_failure'
      })

      // Check if we should retry
      if (attempt === opts.maxAttempts || !opts.retryCondition!(error)) {
        observeLogger.logEvent('error', 'Retry exhausted', {
          attempts: attempt,
          totalTime,
          finalError: error instanceof Error ? error.message : 'Unknown error',
          type: 'retry_exhausted'
        })

        return {
          success: false,
          error: lastError,
          attempts: attempt,
          totalTime
        }
      }

      // Calculate delay with exponential backoff
      const delay = Math.min(
        opts.baseDelay * Math.pow(opts.backoffFactor, attempt - 1),
        opts.maxDelay
      )

      observeLogger.logEvent('debug', `Waiting ${delay}ms before retry`, {
        attempt,
        delay,
        type: 'retry_delay'
      })

      await new Promise(resolve => setTimeout(resolve, delay))
    }
  }

  // This should never be reached, but TypeScript requires it
  return {
    success: false,
    error: lastError,
    attempts: opts.maxAttempts,
    totalTime: Date.now() - startTime
  }
}

/**
 * Create a retry wrapper for API functions
 */
export function createRetryWrapper<T extends any[], R>(
  fn: (...args: T) => Promise<R>,
  options: Partial<RetryOptions> = {}
) {
  return async (...args: T): Promise<R> => {
    const result = await retryWithBackoff(() => fn(...args), options)
    
    if (result.success) {
      return result.data!
    } else {
      throw result.error
    }
  }
}

/**
 * Circuit breaker pattern for API calls
 */
class CircuitBreaker {
  private failures = 0
  private lastFailureTime = 0
  private state: 'closed' | 'open' | 'half-open' = 'closed'

  constructor(
    private failureThreshold = 5,
    private recoveryTimeout = 30000 // 30 seconds
  ) {}

  async execute<T>(fn: () => Promise<T>): Promise<T> {
    if (this.state === 'open') {
      if (Date.now() - this.lastFailureTime > this.recoveryTimeout) {
        this.state = 'half-open'
        observeLogger.logEvent('info', 'Circuit breaker half-open', {
          type: 'circuit_breaker_half_open'
        })
      } else {
        observeLogger.logEvent('warn', 'Circuit breaker open - request blocked', {
          type: 'circuit_breaker_blocked'
        })
        throw new Error('Circuit breaker is open')
      }
    }

    try {
      const result = await fn()
      
      if (this.state === 'half-open') {
        this.reset()
        observeLogger.logEvent('info', 'Circuit breaker closed', {
          type: 'circuit_breaker_closed'
        })
      }
      
      return result
    } catch (error) {
      this.recordFailure()
      throw error
    }
  }

  private recordFailure() {
    this.failures++
    this.lastFailureTime = Date.now()

    if (this.failures >= this.failureThreshold) {
      this.state = 'open'
      observeLogger.logEvent('error', 'Circuit breaker opened', {
        failures: this.failures,
        threshold: this.failureThreshold,
        type: 'circuit_breaker_opened'
      })
    }
  }

  private reset() {
    this.failures = 0
    this.state = 'closed'
  }

  getState() {
    return {
      state: this.state,
      failures: this.failures,
      lastFailureTime: this.lastFailureTime
    }
  }
}

// Create a global circuit breaker for API calls
export const apiCircuitBreaker = new CircuitBreaker()

/**
 * Enhanced API call wrapper with retry and circuit breaker
 */
export async function enhancedApiCall<T>(
  fn: () => Promise<T>,
  options: Partial<RetryOptions> = {}
): Promise<T> {
  return apiCircuitBreaker.execute(async () => {
    const result = await retryWithBackoff(fn, options)
    
    if (result.success) {
      return result.data!
    } else {
      throw result.error
    }
  })
}

// Export types
export type { RetryOptions, RetryResult }