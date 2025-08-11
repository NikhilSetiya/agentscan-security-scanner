import React from 'react';
import { clsx } from 'clsx';
import './Button.css';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  children: React.ReactNode;
  loadingText?: string;
  icon?: React.ReactNode;
  iconPosition?: 'left' | 'right';
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ 
    className, 
    variant = 'primary', 
    size = 'md', 
    loading = false, 
    disabled, 
    children, 
    loadingText,
    icon,
    iconPosition = 'left',
    'aria-label': ariaLabel,
    ...props 
  }, ref) => {
    const isDisabled = disabled || loading;
    
    return (
      <button
        className={clsx(
          'btn',
          `btn-${variant}`,
          `btn-${size}`,
          {
            'btn-loading': loading,
            'btn-disabled': isDisabled,
            'btn-icon-only': !children && icon,
          },
          className
        )}
        disabled={isDisabled}
        aria-label={loading ? loadingText || 'Loading...' : ariaLabel}
        aria-busy={loading}
        ref={ref}
        {...props}
      >
        {loading && (
          <svg className="btn-spinner" viewBox="0 0 24 24" aria-hidden="true">
            <circle
              className="btn-spinner-circle"
              cx="12"
              cy="12"
              r="10"
              fill="none"
              strokeWidth="2"
            />
          </svg>
        )}
        
        {!loading && icon && iconPosition === 'left' && (
          <span className="btn-icon btn-icon-left" aria-hidden="true">
            {icon}
          </span>
        )}
        
        {!loading && children && (
          <span className="btn-text">{children}</span>
        )}
        
        {loading && loadingText && (
          <span className="btn-text">{loadingText}</span>
        )}
        
        {!loading && icon && iconPosition === 'right' && (
          <span className="btn-icon btn-icon-right" aria-hidden="true">
            {icon}
          </span>
        )}
      </button>
    );
  }
);

Button.displayName = 'Button';