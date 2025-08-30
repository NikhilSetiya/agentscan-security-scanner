import React, { useState, useCallback, useEffect } from 'react';
import { RefreshCw, AlertCircle, Wifi, WifiOff } from 'lucide-react';
import { Button } from './ui/button';
import { Alert, AlertDescription } from './ui/alert';
import { Card, CardContent } from './ui/card';
import { EnhancedError, shouldRetry, getRetryDelay } from '../utils/errorHandler';

interface RetryWrapperProps {
  children: React.ReactNode;
  onRetry: () => Promise<void>;
  error?: EnhancedError | null;
  loading?: boolean;
  maxRetries?: number;
  showRetryButton?: boolean;
  autoRetry?: boolean;
}

export function RetryWrapper({
  children,
  onRetry,
  error,
  loading = false,
  maxRetries = 3,
  showRetryButton = true,
  autoRetry = false,
}: RetryWrapperProps) {
  const [retryCount, setRetryCount] = useState(0);
  const [isRetrying, setIsRetrying] = useState(false);
  const [nextRetryIn, setNextRetryIn] = useState<number | null>(null);

  // Auto-retry logic
  useEffect(() => {
    if (error && autoRetry && shouldRetry(error, retryCount) && retryCount < maxRetries) {
      const delay = getRetryDelay(error, retryCount);
      setNextRetryIn(delay);

      const timer = setTimeout(() => {
        handleRetry();
      }, delay);

      // Countdown timer
      const countdownInterval = setInterval(() => {
        setNextRetryIn(prev => {
          if (prev === null || prev <= 1000) {
            clearInterval(countdownInterval);
            return null;
          }
          return prev - 1000;
        });
      }, 1000);

      return () => {
        clearTimeout(timer);
        clearInterval(countdownInterval);
      };
    }
  }, [error, autoRetry, retryCount, maxRetries]);

  const handleRetry = useCallback(async () => {
    if (isRetrying || !error) return;

    setIsRetrying(true);
    setNextRetryIn(null);

    try {
      await onRetry();
      setRetryCount(0); // Reset on successful retry
    } catch (retryError) {
      setRetryCount(prev => prev + 1);
    } finally {
      setIsRetrying(false);
    }
  }, [onRetry, error, isRetrying]);

  const canRetry = error && shouldRetry(error, retryCount) && retryCount < maxRetries;
  const hasExceededRetries = retryCount >= maxRetries;

  // Show children if no error
  if (!error) {
    return <>{children}</>;
  }

  // Determine error type for appropriate icon and styling
  const getErrorIcon = () => {
    switch (error.code) {
      case 'NETWORK_ERROR':
      case 'TIMEOUT':
        return <WifiOff className="h-5 w-5" />;
      case 'SERVICE_UNAVAILABLE':
      case 'EXTERNAL_SERVICE_ERROR':
        return <Wifi className="h-5 w-5" />;
      default:
        return <AlertCircle className="h-5 w-5" />;
    }
  };

  const getErrorVariant = () => {
    if (error.severity === 'critical' || error.severity === 'high') {
      return 'destructive';
    }
    return 'default';
  };

  return (
    <Card className="w-full">
      <CardContent className="p-6">
        <Alert variant={getErrorVariant()}>
          {getErrorIcon()}
          <AlertDescription className="ml-2">
            <div className="space-y-3">
              {/* Error message */}
              <div>
                <div className="font-medium">{error.userMessage}</div>
                {error.details && (
                  <div className="text-sm text-muted-foreground mt-1">
                    {JSON.stringify(error.details, null, 2)}
                  </div>
                )}
              </div>

              {/* Retry information */}
              {retryCount > 0 && (
                <div className="text-sm text-muted-foreground">
                  Retry attempt {retryCount} of {maxRetries}
                </div>
              )}

              {/* Auto-retry countdown */}
              {nextRetryIn && (
                <div className="text-sm text-muted-foreground">
                  Retrying automatically in {Math.ceil(nextRetryIn / 1000)} seconds...
                </div>
              )}

              {/* Exceeded retries message */}
              {hasExceededRetries && (
                <div className="text-sm text-destructive">
                  Maximum retry attempts exceeded. Please try again later or contact support.
                </div>
              )}

              {/* Action buttons */}
              <div className="flex gap-2 pt-2">
                {showRetryButton && canRetry && (
                  <Button
                    size="sm"
                    onClick={handleRetry}
                    disabled={isRetrying || loading}
                    className="flex items-center gap-2"
                  >
                    <RefreshCw className={`h-4 w-4 ${isRetrying ? 'animate-spin' : ''}`} />
                    {isRetrying ? 'Retrying...' : 'Retry'}
                  </Button>
                )}

                {hasExceededRetries && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => window.location.reload()}
                    className="flex items-center gap-2"
                  >
                    <RefreshCw className="h-4 w-4" />
                    Reload Page
                  </Button>
                )}
              </div>

              {/* Correlation ID for support */}
              {error.correlationId && (
                <div className="text-xs text-muted-foreground font-mono">
                  Correlation ID: {error.correlationId}
                </div>
              )}
            </div>
          </AlertDescription>
        </Alert>
      </CardContent>
    </Card>
  );
}

// Hook for using retry logic in components
export function useRetry<T>(
  asyncFunction: () => Promise<T>,
  dependencies: React.DependencyList = [],
  options: {
    maxRetries?: number;
    autoRetry?: boolean;
    onError?: (error: EnhancedError) => void;
  } = {}
) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<EnhancedError | null>(null);
  const [loading, setLoading] = useState(false);
  const [retryCount, setRetryCount] = useState(0);

  const { maxRetries = 3, autoRetry = false, onError } = options;

  const execute = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const result = await asyncFunction();
      setData(result);
      setRetryCount(0); // Reset on success
      return result;
    } catch (err) {
      const enhancedError = err as EnhancedError;
      setError(enhancedError);
      
      if (onError) {
        onError(enhancedError);
      }

      // Auto-retry logic
      if (autoRetry && shouldRetry(enhancedError, retryCount) && retryCount < maxRetries) {
        const delay = getRetryDelay(enhancedError, retryCount);
        setRetryCount(prev => prev + 1);
        
        setTimeout(() => {
          execute();
        }, delay);
      }

      throw enhancedError;
    } finally {
      setLoading(false);
    }
  }, [asyncFunction, retryCount, maxRetries, autoRetry, onError, ...dependencies]);

  const retry = useCallback(() => {
    setRetryCount(prev => prev + 1);
    return execute();
  }, [execute]);

  // Initial execution
  useEffect(() => {
    execute();
  }, dependencies);

  return {
    data,
    error,
    loading,
    retry,
    retryCount,
    canRetry: error ? shouldRetry(error, retryCount) && retryCount < maxRetries : false,
  };
}