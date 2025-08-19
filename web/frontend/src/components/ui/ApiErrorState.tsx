/**
 * API Error State Component
 * Displays API errors with enhanced error handling and user-friendly messages
 */

import React from 'react';
import { ErrorState, ErrorStateProps } from './ErrorState';
import { ApiError } from '../../services/api';
import { enhanceError, getErrorVariant, createErrorMessage, shouldRetry } from '../../utils/errorHandler';

interface ApiErrorStateProps extends Omit<ErrorStateProps, 'variant' | 'title' | 'message'> {
  error: ApiError;
  context?: string;
  onRetry?: () => void;
  onDismiss?: () => void;
  showDetails?: boolean;
  attemptCount?: number;
}

export const ApiErrorState: React.FC<ApiErrorStateProps> = ({
  error,
  context,
  onRetry,
  onDismiss,
  showDetails = false,
  attemptCount = 0,
  className,
  size = 'md',
  ...props
}) => {
  const enhancedError = enhanceError(error);
  const variant = getErrorVariant(enhancedError);
  const userMessage = createErrorMessage(enhancedError, context);
  const canRetry = shouldRetry(enhancedError, attemptCount) && onRetry;

  // Determine primary action
  const primaryAction = canRetry ? {
    label: attemptCount > 0 ? 'Try again' : 'Retry',
    onClick: onRetry,
    variant: 'primary' as const,
  } : undefined;

  // Determine secondary action
  const secondaryAction = onDismiss ? {
    label: 'Dismiss',
    onClick: onDismiss,
  } : undefined;

  return (
    <div className="api-error-state">
      <ErrorState
        variant={variant}
        title={getErrorTitle(enhancedError)}
        message={userMessage}
        action={primaryAction}
        secondaryAction={secondaryAction}
        className={className}
        size={size}
        {...props}
      />
      
      {showDetails && enhancedError.details && (
        <details className="error-details">
          <summary>Technical details</summary>
          <div className="error-details-content">
            <p><strong>Error Code:</strong> {enhancedError.code}</p>
            <p><strong>Message:</strong> {enhancedError.message}</p>
            {enhancedError.details && (
              <div>
                <strong>Details:</strong>
                <pre>{JSON.stringify(enhancedError.details, null, 2)}</pre>
              </div>
            )}
          </div>
        </details>
      )}
    </div>
  );
};

// Helper function to get appropriate error title
function getErrorTitle(error: ReturnType<typeof enhanceError>): string {
  switch (error.code) {
    case 'UNAUTHORIZED':
      return 'Authentication Required';
    case 'FORBIDDEN':
      return 'Access Denied';
    case 'NOT_FOUND':
      return 'Not Found';
    case 'VALIDATION_ERROR':
    case 'BAD_REQUEST':
      return 'Invalid Input';
    case 'RATE_LIMIT_EXCEEDED':
      return 'Rate Limited';
    case 'NETWORK_ERROR':
      return 'Connection Error';
    case 'TIMEOUT':
      return 'Request Timeout';
    case 'INTERNAL_ERROR':
      return 'Server Error';
    case 'AGENT_ERROR':
      return 'Scan Agent Error';
    case 'SCAN_ERROR':
      return 'Scan Failed';
    default:
      return 'Error';
  }
}

// Specialized components for common API error scenarios

export const AuthenticationError: React.FC<{
  onLogin?: () => void;
  onDismiss?: () => void;
  className?: string;
}> = ({ onLogin, onDismiss, className }) => {
  const error: ApiError = {
    code: 'UNAUTHORIZED',
    message: 'Authentication required',
  };

  return (
    <ApiErrorState
      error={error}
      onRetry={onLogin}
      onDismiss={onDismiss}
      className={className}
    />
  );
};

export const NetworkError: React.FC<{
  onRetry?: () => void;
  onDismiss?: () => void;
  className?: string;
}> = ({ onRetry, onDismiss, className }) => {
  const error: ApiError = {
    code: 'NETWORK_ERROR',
    message: 'Network connection failed',
  };

  return (
    <ApiErrorState
      error={error}
      onRetry={onRetry}
      onDismiss={onDismiss}
      className={className}
    />
  );
};

export const ValidationError: React.FC<{
  error: ApiError;
  onRetry?: () => void;
  onDismiss?: () => void;
  className?: string;
}> = ({ error, onRetry, onDismiss, className }) => {
  return (
    <ApiErrorState
      error={error}
      context="Validation failed"
      onRetry={onRetry}
      onDismiss={onDismiss}
      showDetails={true}
      className={className}
    />
  );
};

// Inline API error component for smaller spaces
export const InlineApiError: React.FC<{
  error: ApiError;
  context?: string;
  onRetry?: () => void;
  className?: string;
}> = ({ error, context, onRetry, className }) => {
  const enhancedError = enhanceError(error);
  const userMessage = createErrorMessage(enhancedError, context);

  return (
    <div className={`inline-api-error ${className || ''}`}>
      <span className="error-message">{userMessage}</span>
      {onRetry && shouldRetry(enhancedError) && (
        <button 
          className="retry-button"
          onClick={onRetry}
          type="button"
        >
          Retry
        </button>
      )}
    </div>
  );
};