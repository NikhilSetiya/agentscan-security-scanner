/**
 * Enhanced Error Handler for AgentScan Frontend
 * Handles standardized API errors and provides user-friendly error messages
 */

import { ApiError } from '../services/api';
import { observeLogger } from '../services/observeLogger';

// Error severity levels
export type ErrorSeverity = 'low' | 'medium' | 'high' | 'critical';

// Enhanced error interface
export interface EnhancedError {
  code: string;
  message: string;
  userMessage: string;
  severity: ErrorSeverity;
  details?: Record<string, any>;
  retryable: boolean;
  actionable: boolean;
}

// Error code mappings to user-friendly messages
const ERROR_MESSAGES: Record<string, Partial<EnhancedError>> = {
  // Authentication errors
  'UNAUTHORIZED': {
    userMessage: 'Please log in to access this feature.',
    severity: 'medium',
    retryable: false,
    actionable: true,
  },
  'FORBIDDEN': {
    userMessage: 'You don\'t have permission to perform this action.',
    severity: 'medium',
    retryable: false,
    actionable: false,
  },
  'AUTHENTICATION_ERROR': {
    userMessage: 'Authentication failed. Please check your credentials.',
    severity: 'medium',
    retryable: true,
    actionable: true,
  },

  // Validation errors
  'VALIDATION_ERROR': {
    userMessage: 'Please check your input and try again.',
    severity: 'low',
    retryable: true,
    actionable: true,
  },
  'BAD_REQUEST': {
    userMessage: 'Invalid request. Please check your input.',
    severity: 'low',
    retryable: true,
    actionable: true,
  },

  // Resource errors
  'NOT_FOUND': {
    userMessage: 'The requested resource was not found.',
    severity: 'medium',
    retryable: false,
    actionable: false,
  },
  'CONFLICT': {
    userMessage: 'This action conflicts with existing data.',
    severity: 'medium',
    retryable: false,
    actionable: true,
  },

  // Rate limiting
  'RATE_LIMIT_EXCEEDED': {
    userMessage: 'Too many requests. Please wait a moment and try again.',
    severity: 'low',
    retryable: true,
    actionable: false,
  },

  // Network errors
  'NETWORK_ERROR': {
    userMessage: 'Network connection failed. Please check your internet connection.',
    severity: 'high',
    retryable: true,
    actionable: true,
  },
  'TIMEOUT': {
    userMessage: 'Request timed out. Please try again.',
    severity: 'medium',
    retryable: true,
    actionable: true,
  },

  // Server errors
  'INTERNAL_ERROR': {
    userMessage: 'An internal server error occurred. Please try again later.',
    severity: 'high',
    retryable: true,
    actionable: false,
  },
  'EXTERNAL_SERVICE_ERROR': {
    userMessage: 'An external service is temporarily unavailable.',
    severity: 'high',
    retryable: true,
    actionable: false,
  },

  // Agent-specific errors
  'AGENT_ERROR': {
    userMessage: 'Security scan agent encountered an error.',
    severity: 'medium',
    retryable: true,
    actionable: false,
  },
  'SCAN_ERROR': {
    userMessage: 'Security scan failed. Please try again.',
    severity: 'medium',
    retryable: true,
    actionable: true,
  },
  'CONSENSUS_ERROR': {
    userMessage: 'Error processing scan results.',
    severity: 'medium',
    retryable: true,
    actionable: false,
  },

  // Unknown errors
  'UNKNOWN_ERROR': {
    userMessage: 'An unexpected error occurred. Please try again.',
    severity: 'high',
    retryable: true,
    actionable: true,
  },
};

/**
 * Enhances an API error with user-friendly information
 */
export function enhanceError(apiError: ApiError): EnhancedError {
  const errorConfig = ERROR_MESSAGES[apiError.code] || ERROR_MESSAGES['UNKNOWN_ERROR'];
  
  const enhancedError: EnhancedError = {
    code: apiError.code,
    message: apiError.message,
    userMessage: errorConfig.userMessage || apiError.message,
    severity: errorConfig.severity || 'medium',
    details: apiError.details,
    retryable: errorConfig.retryable ?? true,
    actionable: errorConfig.actionable ?? true,
  };

  // Log error to Observe MCP for debugging
  observeLogger.logError(new Error(`API Error: ${apiError.code}`), {
    code: apiError.code,
    message: apiError.message,
    details: apiError.details,
    severity: enhancedError.severity,
  });

  return enhancedError;
}

/**
 * Gets appropriate error variant for ErrorState component
 */
export function getErrorVariant(error: EnhancedError): 'error' | 'warning' | 'network' | 'forbidden' {
  switch (error.code) {
    case 'NETWORK_ERROR':
    case 'TIMEOUT':
      return 'network';
    case 'FORBIDDEN':
    case 'UNAUTHORIZED':
      return 'forbidden';
    case 'VALIDATION_ERROR':
    case 'BAD_REQUEST':
      return 'warning';
    default:
      return 'error';
  }
}

/**
 * Determines if an error should trigger a retry mechanism
 */
export function shouldRetry(error: EnhancedError, attemptCount: number = 0): boolean {
  if (!error.retryable || attemptCount >= 3) {
    return false;
  }

  // Don't retry client errors (4xx) except for rate limiting
  if (error.code.startsWith('4') && error.code !== 'RATE_LIMIT_EXCEEDED') {
    return false;
  }

  return true;
}

/**
 * Gets retry delay based on error type and attempt count
 */
export function getRetryDelay(error: EnhancedError, attemptCount: number): number {
  const baseDelay = 1000; // 1 second
  const maxDelay = 30000; // 30 seconds

  let delay = baseDelay * Math.pow(2, attemptCount); // Exponential backoff

  // Special cases
  switch (error.code) {
    case 'RATE_LIMIT_EXCEEDED':
      delay = Math.min(delay * 2, maxDelay); // Longer delay for rate limiting
      break;
    case 'NETWORK_ERROR':
    case 'TIMEOUT':
      delay = Math.min(delay, 10000); // Cap network errors at 10 seconds
      break;
  }

  return Math.min(delay, maxDelay);
}

/**
 * Formats error details for display
 */
export function formatErrorDetails(details?: Record<string, any>): string | undefined {
  if (!details || Object.keys(details).length === 0) {
    return undefined;
  }

  // Handle validation errors specially
  if (details.validation) {
    const validationErrors = details.validation;
    if (Array.isArray(validationErrors)) {
      return validationErrors.join(', ');
    }
    if (typeof validationErrors === 'object') {
      return Object.entries(validationErrors)
        .map(([field, message]) => `${field}: ${message}`)
        .join(', ');
    }
  }

  // Handle other details
  return Object.entries(details)
    .map(([key, value]) => `${key}: ${value}`)
    .join(', ');
}

/**
 * Creates a user-friendly error message with context
 */
export function createErrorMessage(error: EnhancedError, context?: string): string {
  let message = error.userMessage;

  if (context) {
    message = `${context}: ${message}`;
  }

  const formattedDetails = formatErrorDetails(error.details);
  if (formattedDetails) {
    message += ` (${formattedDetails})`;
  }

  return message;
}

/**
 * Error handler for React components
 */
export class ErrorHandler {
  private static instance: ErrorHandler;

  public static getInstance(): ErrorHandler {
    if (!ErrorHandler.instance) {
      ErrorHandler.instance = new ErrorHandler();
    }
    return ErrorHandler.instance;
  }

  /**
   * Handles API errors and returns enhanced error information
   */
  public handleApiError(apiError: ApiError, context?: string): EnhancedError {
    const enhancedError = enhanceError(apiError);
    
    // Log to console in development
    if (import.meta.env.MODE === 'development') {
      console.error('API Error:', {
        code: enhancedError.code,
        message: enhancedError.message,
        userMessage: enhancedError.userMessage,
        context,
        details: enhancedError.details,
      });
    }

    return enhancedError;
  }

  /**
   * Handles unexpected errors
   */
  public handleUnexpectedError(error: Error, context?: string): EnhancedError {
    const apiError: ApiError = {
      code: 'UNKNOWN_ERROR',
      message: error.message,
    };

    return this.handleApiError(apiError, context);
  }

  /**
   * Shows a toast notification for errors
   */
  public showErrorToast(error: EnhancedError, context?: string): void {
    const message = createErrorMessage(error, context);
    
    // Dispatch custom event for toast notifications
    window.dispatchEvent(new CustomEvent('show-toast', {
      detail: {
        type: error.severity === 'critical' || error.severity === 'high' ? 'error' : 'warning',
        message,
        duration: error.severity === 'low' ? 3000 : 5000,
      }
    }));
  }
}

// Export singleton instance
export const errorHandler = ErrorHandler.getInstance();

// Utility functions are already exported above