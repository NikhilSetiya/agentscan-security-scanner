import React from 'react';
import { clsx } from 'clsx';
import './LoadingSkeleton.css';

export interface LoadingSkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  width?: string | number;
  height?: string | number;
  variant?: 'text' | 'rectangular' | 'circular';
  animation?: 'pulse' | 'wave';
}

export const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({
  className,
  width,
  height,
  variant = 'text',
  animation = 'wave',
  style,
  ...props
}) => {
  const skeletonStyle = {
    width,
    height,
    ...style,
  };

  return (
    <div
      className={clsx(
        'loading-skeleton',
        `loading-skeleton-${variant}`,
        `loading-skeleton-${animation}`,
        className
      )}
      style={skeletonStyle}
      aria-label="Loading..."
      {...props}
    />
  );
};

// Specialized skeleton components for common use cases
export const TextSkeleton: React.FC<{ lines?: number; className?: string }> = ({ 
  lines = 1, 
  className 
}) => {
  return (
    <div className={clsx('text-skeleton-container', className)}>
      {Array.from({ length: lines }).map((_, index) => (
        <LoadingSkeleton
          key={index}
          variant="text"
          height="1em"
          width={index === lines - 1 ? '60%' : '100%'}
          className="text-skeleton-line"
        />
      ))}
    </div>
  );
};

export const CardSkeleton: React.FC<{ className?: string }> = ({ className }) => {
  return (
    <div className={clsx('card-skeleton', className)}>
      <div className="card-skeleton-header">
        <LoadingSkeleton variant="text" width="40%" height="1.5em" />
        <LoadingSkeleton variant="text" width="60%" height="1em" />
      </div>
      <div className="card-skeleton-content">
        <TextSkeleton lines={3} />
      </div>
    </div>
  );
};

export const TableSkeleton: React.FC<{ 
  rows?: number; 
  columns?: number; 
  className?: string 
}> = ({ 
  rows = 5, 
  columns = 4, 
  className 
}) => {
  return (
    <div className={clsx('table-skeleton', className)}>
      {/* Header */}
      <div className="table-skeleton-header">
        {Array.from({ length: columns }).map((_, index) => (
          <LoadingSkeleton
            key={`header-${index}`}
            variant="text"
            width="80%"
            height="1.2em"
          />
        ))}
      </div>
      
      {/* Rows */}
      {Array.from({ length: rows }).map((_, rowIndex) => (
        <div key={`row-${rowIndex}`} className="table-skeleton-row">
          {Array.from({ length: columns }).map((_, colIndex) => (
            <LoadingSkeleton
              key={`cell-${rowIndex}-${colIndex}`}
              variant="text"
              width={colIndex === 0 ? '90%' : '70%'}
              height="1em"
            />
          ))}
        </div>
      ))}
    </div>
  );
};

export const StatCardSkeleton: React.FC<{ className?: string }> = ({ className }) => {
  return (
    <div className={clsx('stat-card-skeleton', className)}>
      <LoadingSkeleton variant="circular" width={48} height={48} />
      <div className="stat-card-skeleton-content">
        <LoadingSkeleton variant="text" width="60%" height="1.5em" />
        <LoadingSkeleton variant="text" width="80%" height="1em" />
      </div>
    </div>
  );
};