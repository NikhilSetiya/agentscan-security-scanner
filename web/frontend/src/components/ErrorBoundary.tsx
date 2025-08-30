import React, { Component, ErrorInfo, ReactNode } from 'react';
import { AlertTriangle, RefreshCw, Home, Bug } from 'lucide-react';
import { Button } from './ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Alert, AlertDescription } from './ui/alert';
import { errorHandler, EnhancedError } from '../utils/errorHandler';
import { observeLogger } from '../services/observeLogger';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  enhancedError: EnhancedError | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      enhancedError: null,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // Enhance the error for better user experience
    const enhancedError = errorHandler.handleUnexpectedError(error, 'React Error Boundary');
    
    // Log error to Observe MCP
    observeLogger.logError(error, {
      component: 'ErrorBoundary',
      errorInfo: errorInfo.componentStack,
      enhancedError,
    });

    this.setState({
      error,
      errorInfo,
      enhancedError,
    });

    // Call custom error handler if provided
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // Log to console in development
    if (import.meta.env.MODE === 'development') {
      console.error('Error Boundary caught an error:', error);
      console.error('Component stack:', errorInfo.componentStack);
    }
  }

  handleRetry = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      enhancedError: null,
    });
  };

  handleGoHome = () => {
    window.location.href = '/';
  };

  handleReportBug = () => {
    const { error, errorInfo, enhancedError } = this.state;
    
    // Create bug report data
    const bugReport = {
      error: error?.message,
      stack: error?.stack,
      componentStack: errorInfo?.componentStack,
      enhancedError,
      userAgent: navigator.userAgent,
      url: window.location.href,
      timestamp: new Date().toISOString(),
    };

    // In a real app, you would send this to your bug tracking system
    console.log('Bug report:', bugReport);
    
    // For now, copy to clipboard
    navigator.clipboard.writeText(JSON.stringify(bugReport, null, 2));
    alert('Bug report copied to clipboard. Please paste it in your bug report.');
  };

  render() {
    if (this.state.hasError) {
      // Custom fallback UI
      if (this.props.fallback) {
        return this.props.fallback;
      }

      const { enhancedError, error } = this.state;

      return (
        <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
          <Card className="w-full max-w-2xl">
            <CardHeader className="text-center">
              <div className="mx-auto w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mb-4">
                <AlertTriangle className="w-8 h-8 text-red-600" />
              </div>
              <CardTitle className="text-2xl font-bold text-gray-900">
                Something went wrong
              </CardTitle>
              <CardDescription className="text-lg">
                {enhancedError?.userMessage || 'An unexpected error occurred in the application.'}
              </CardDescription>
            </CardHeader>
            
            <CardContent className="space-y-6">
              {/* Error details for development */}
              {import.meta.env.MODE === 'development' && error && (
                <Alert>
                  <Bug className="h-4 w-4" />
                  <AlertDescription className="font-mono text-sm">
                    <div className="font-semibold mb-2">Development Error Details:</div>
                    <div className="whitespace-pre-wrap break-all">
                      {error.message}
                    </div>
                    {enhancedError?.correlationId && (
                      <div className="mt-2 text-xs text-gray-600">
                        Correlation ID: {enhancedError.correlationId}
                      </div>
                    )}
                  </AlertDescription>
                </Alert>
              )}

              {/* Error severity indicator */}
              {enhancedError && (
                <div className="flex items-center gap-2 text-sm">
                  <span className="font-medium">Severity:</span>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                    enhancedError.severity === 'critical' ? 'bg-red-100 text-red-800' :
                    enhancedError.severity === 'high' ? 'bg-orange-100 text-orange-800' :
                    enhancedError.severity === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-blue-100 text-blue-800'
                  }`}>
                    {enhancedError.severity.toUpperCase()}
                  </span>
                </div>
              )}

              {/* Action buttons */}
              <div className="flex flex-col sm:flex-row gap-3">
                {enhancedError?.retryable !== false && (
                  <Button 
                    onClick={this.handleRetry}
                    className="flex items-center gap-2"
                  >
                    <RefreshCw className="w-4 h-4" />
                    Try Again
                  </Button>
                )}
                
                <Button 
                  variant="outline" 
                  onClick={this.handleGoHome}
                  className="flex items-center gap-2"
                >
                  <Home className="w-4 h-4" />
                  Go Home
                </Button>
                
                <Button 
                  variant="outline" 
                  onClick={this.handleReportBug}
                  className="flex items-center gap-2"
                >
                  <Bug className="w-4 h-4" />
                  Report Bug
                </Button>
              </div>

              {/* Help text */}
              <div className="text-sm text-gray-600 text-center">
                If this problem persists, please contact support with the correlation ID above.
              </div>
            </CardContent>
          </Card>
        </div>
      );
    }

    return this.props.children;
  }
}

// Higher-order component for wrapping components with error boundary
export function withErrorBoundary<P extends object>(
  Component: React.ComponentType<P>,
  fallback?: ReactNode
) {
  return function WrappedComponent(props: P) {
    return (
      <ErrorBoundary fallback={fallback}>
        <Component {...props} />
      </ErrorBoundary>
    );
  };
}

// Hook for error boundary in functional components
export function useErrorHandler() {
  return {
    handleError: (error: Error, context?: string) => {
      const enhancedError = errorHandler.handleUnexpectedError(error, context);
      
      // In a real app, you might want to show a toast or modal
      console.error('Handled error:', enhancedError);
      
      return enhancedError;
    },
  };
}