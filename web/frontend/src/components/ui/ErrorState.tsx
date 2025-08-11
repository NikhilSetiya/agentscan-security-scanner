import React from 'react';
import { clsx } from 'clsx';
import { AlertTriangle, RefreshCw, Bug, Wifi, Shield } from 'lucide-react';
import { Button } from './Button';
import './ErrorState.css';

export interface ErrorStateProps {
  variant?: 'error' | 'warning' | 'network' | 'not-found' | 'forbidden' | 'maintenance';
  title?: string;
  message?: string;
  action?: {
    label: string;
    onClick: () => void;
    variant?: 'primary' | 'secondary';
  };
  secondaryAction?: {
    label: string;
    onClick: () => void;
  };
  className?: string;
  size?: 'sm' | 'md' | 'lg';
}

const errorConfigs = {
  error: {
    icon: AlertTriangle,
    title: 'Something went wrong',
    message: 'An unexpected error occurred. Please try again or contact support if the problem persists.',
    color: 'var(--color-error)',
    bgColor: 'rgba(220, 38, 38, 0.05)',
  },
  warning: {
    icon: AlertTriangle,
    title: 'Warning',
    message: 'There might be an issue that needs your attention.',
    color: 'var(--color-warning)',
    bgColor: 'rgba(217, 119, 6, 0.05)',
  },
  network: {
    icon: Wifi,
    title: 'Connection problem',
    message: 'Unable to connect to the server. Please check your internet connection and try again.',
    color: 'var(--color-error)',
    bgColor: 'rgba(220, 38, 38, 0.05)',
  },
  'not-found': {
    icon: Bug,
    title: 'Page not found',
    message: 'The page you\'re looking for doesn\'t exist or has been moved.',
    color: 'var(--color-gray-500)',
    bgColor: 'var(--color-gray-50)',
  },
  forbidden: {
    icon: Shield,
    title: 'Access denied',
    message: 'You don\'t have permission to access this resource.',
    color: 'var(--color-error)',
    bgColor: 'rgba(220, 38, 38, 0.05)',
  },
  maintenance: {
    icon: RefreshCw,
    title: 'Under maintenance',
    message: 'The system is currently under maintenance. Please try again later.',
    color: 'var(--color-info)',
    bgColor: 'rgba(37, 99, 235, 0.05)',
  },
};

export const ErrorState: React.FC<ErrorStateProps> = ({
  variant = 'error',
  title,
  message,
  action,
  secondaryAction,
  className,
  size = 'md',
}) => {
  const config = errorConfigs[variant];
  const Icon = config.icon;

  return (
    <div className={clsx('error-state', `error-state-${size}`, className)}>
      <div 
        className="error-state-icon"
        style={{ 
          color: config.color,
          backgroundColor: config.bgColor,
        }}
      >
        <Icon size={size === 'sm' ? 24 : size === 'lg' ? 48 : 32} />
      </div>
      
      <div className="error-state-content">
        <h3 className="error-state-title">
          {title || config.title}
        </h3>
        <p className="error-state-message">
          {message || config.message}
        </p>
      </div>

      {(action || secondaryAction) && (
        <div className="error-state-actions">
          {action && (
            <Button
              variant={action.variant || 'primary'}
              onClick={action.onClick}
              size={size}
            >
              {action.label}
            </Button>
          )}
          {secondaryAction && (
            <Button
              variant="ghost"
              onClick={secondaryAction.onClick}
              size={size}
            >
              {secondaryAction.label}
            </Button>
          )}
        </div>
      )}
    </div>
  );
};

// Specialized error components for common scenarios
export const NetworkError: React.FC<{
  onRetry?: () => void;
  className?: string;
}> = ({ onRetry, className }) => {
  return (
    <ErrorState
      variant="network"
      action={onRetry ? {
        label: 'Try again',
        onClick: onRetry,
      } : undefined}
      className={className}
    />
  );
};

export const NotFoundError: React.FC<{
  onGoHome?: () => void;
  onGoBack?: () => void;
  className?: string;
}> = ({ onGoHome, onGoBack, className }) => {
  return (
    <ErrorState
      variant="not-found"
      action={onGoHome ? {
        label: 'Go home',
        onClick: onGoHome,
      } : undefined}
      secondaryAction={onGoBack ? {
        label: 'Go back',
        onClick: onGoBack,
      } : undefined}
      className={className}
    />
  );
};

export const ForbiddenError: React.FC<{
  onGoHome?: () => void;
  className?: string;
}> = ({ onGoHome, className }) => {
  return (
    <ErrorState
      variant="forbidden"
      action={onGoHome ? {
        label: 'Go home',
        onClick: onGoHome,
      } : undefined}
      className={className}
    />
  );
};

// Inline error component for form fields and smaller spaces
export const InlineError: React.FC<{
  message: string;
  className?: string;
}> = ({ message, className }) => {
  return (
    <div className={clsx('inline-error', className)}>
      <AlertTriangle size={16} />
      <span>{message}</span>
    </div>
  );
};

// Error boundary fallback component
export const ErrorBoundaryFallback: React.FC<{
  error: Error;
  resetError: () => void;
}> = ({ error, resetError }) => {
  return (
    <div className="error-boundary-fallback">
      <ErrorState
        variant="error"
        title="Application Error"
        message="Something went wrong in the application. This error has been logged and will be investigated."
        action={{
          label: 'Try again',
          onClick: resetError,
        }}
        secondaryAction={{
          label: 'Reload page',
          onClick: () => window.location.reload(),
        }}
        size="lg"
      />
      
      {import.meta.env.MODE === 'development' && (
        <details className="error-details">
          <summary>Error details (development only)</summary>
          <pre className="error-stack">
            {error.message}
            {'\n\n'}
            {error.stack}
          </pre>
        </details>
      )}
    </div>
  );
};