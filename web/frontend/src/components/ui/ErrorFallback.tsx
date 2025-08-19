import React from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';
import { Button } from './Button';

interface ErrorFallbackProps {
  message?: string;
  onRetry?: () => void;
  showRetry?: boolean;
}

export const ErrorFallback: React.FC<ErrorFallbackProps> = ({ 
  message = "Something went wrong", 
  onRetry,
  showRetry = true 
}) => {
  return (
    <div className="flex flex-col items-center justify-center p-8 text-center">
      <AlertTriangle className="w-12 h-12 text-orange-500 mb-4" />
      <h3 className="text-lg font-semibold text-gray-900 mb-2">
        Oops! {message}
      </h3>
      <p className="text-gray-600 mb-4">
        We're having trouble loading this data. Please try again.
      </p>
      {showRetry && onRetry && (
        <Button 
          onClick={onRetry}
          variant="secondary"
          size="sm"
          icon={<RefreshCw size={16} />}
        >
          Try Again
        </Button>
      )}
    </div>
  );
};