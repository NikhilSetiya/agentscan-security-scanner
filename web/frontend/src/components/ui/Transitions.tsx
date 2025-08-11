import React, { useState, useEffect, useRef } from 'react';
import { clsx } from 'clsx';
import './Transitions.css';

export interface FadeInProps {
  children: React.ReactNode;
  delay?: number;
  duration?: number;
  className?: string;
  trigger?: boolean;
}

export const FadeIn: React.FC<FadeInProps> = ({
  children,
  delay = 0,
  duration = 300,
  className,
  trigger = true,
}) => {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    if (trigger) {
      const timer = setTimeout(() => {
        setIsVisible(true);
      }, delay);
      return () => clearTimeout(timer);
    } else {
      setIsVisible(false);
    }
  }, [trigger, delay]);

  return (
    <div
      className={clsx(
        'fade-in',
        { 'fade-in-visible': isVisible },
        className
      )}
      style={{
        transitionDuration: `${duration}ms`,
      }}
    >
      {children}
    </div>
  );
};

export interface SlideInProps {
  children: React.ReactNode;
  direction?: 'up' | 'down' | 'left' | 'right';
  delay?: number;
  duration?: number;
  distance?: number;
  className?: string;
  trigger?: boolean;
}

export const SlideIn: React.FC<SlideInProps> = ({
  children,
  direction = 'up',
  delay = 0,
  duration = 300,
  distance = 20,
  className,
  trigger = true,
}) => {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    if (trigger) {
      const timer = setTimeout(() => {
        setIsVisible(true);
      }, delay);
      return () => clearTimeout(timer);
    } else {
      setIsVisible(false);
    }
  }, [trigger, delay]);

  const getTransform = () => {
    if (isVisible) return 'translate3d(0, 0, 0)';
    
    switch (direction) {
      case 'up':
        return `translate3d(0, ${distance}px, 0)`;
      case 'down':
        return `translate3d(0, -${distance}px, 0)`;
      case 'left':
        return `translate3d(${distance}px, 0, 0)`;
      case 'right':
        return `translate3d(-${distance}px, 0, 0)`;
      default:
        return `translate3d(0, ${distance}px, 0)`;
    }
  };

  return (
    <div
      className={clsx('slide-in', className)}
      style={{
        transform: getTransform(),
        opacity: isVisible ? 1 : 0,
        transitionDuration: `${duration}ms`,
        transitionProperty: 'transform, opacity',
        transitionTimingFunction: 'cubic-bezier(0.4, 0, 0.2, 1)',
      }}
    >
      {children}
    </div>
  );
};

export interface StaggeredListProps {
  children: React.ReactNode[];
  staggerDelay?: number;
  itemDelay?: number;
  className?: string;
  trigger?: boolean;
}

export const StaggeredList: React.FC<StaggeredListProps> = ({
  children,
  staggerDelay = 100,
  itemDelay = 0,
  className,
  trigger = true,
}) => {
  return (
    <div className={clsx('staggered-list', className)}>
      {React.Children.map(children, (child, index) => (
        <SlideIn
          key={index}
          delay={itemDelay + index * staggerDelay}
          trigger={trigger}
        >
          {child}
        </SlideIn>
      ))}
    </div>
  );
};

export interface ScaleInProps {
  children: React.ReactNode;
  delay?: number;
  duration?: number;
  className?: string;
  trigger?: boolean;
}

export const ScaleIn: React.FC<ScaleInProps> = ({
  children,
  delay = 0,
  duration = 300,
  className,
  trigger = true,
}) => {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    if (trigger) {
      const timer = setTimeout(() => {
        setIsVisible(true);
      }, delay);
      return () => clearTimeout(timer);
    } else {
      setIsVisible(false);
    }
  }, [trigger, delay]);

  return (
    <div
      className={clsx('scale-in', className)}
      style={{
        transform: isVisible ? 'scale(1)' : 'scale(0.95)',
        opacity: isVisible ? 1 : 0,
        transitionDuration: `${duration}ms`,
        transitionProperty: 'transform, opacity',
        transitionTimingFunction: 'cubic-bezier(0.4, 0, 0.2, 1)',
      }}
    >
      {children}
    </div>
  );
};

// Hook for intersection observer animations
export const useInViewAnimation = (options?: IntersectionObserverInit) => {
  const [isInView, setIsInView] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsInView(true);
          // Optionally unobserve after first intersection
          observer.unobserve(entry.target);
        }
      },
      {
        threshold: 0.1,
        rootMargin: '50px',
        ...options,
      }
    );

    if (ref.current) {
      observer.observe(ref.current);
    }

    return () => observer.disconnect();
  }, []);

  return { ref, isInView };
};

// Animated container for scroll-triggered animations
export interface AnimateOnScrollProps {
  children: React.ReactNode;
  animation?: 'fadeIn' | 'slideUp' | 'slideDown' | 'slideLeft' | 'slideRight' | 'scaleIn';
  delay?: number;
  duration?: number;
  className?: string;
  threshold?: number;
  rootMargin?: string;
}

export const AnimateOnScroll: React.FC<AnimateOnScrollProps> = ({
  children,
  animation = 'fadeIn',
  delay = 0,
  duration = 600,
  className,
  threshold = 0.1,
  rootMargin = '50px',
}) => {
  const { ref, isInView } = useInViewAnimation({ threshold, rootMargin });

  const renderAnimation = () => {
    switch (animation) {
      case 'slideUp':
        return (
          <SlideIn direction="up" trigger={isInView} delay={delay} duration={duration}>
            {children}
          </SlideIn>
        );
      case 'slideDown':
        return (
          <SlideIn direction="down" trigger={isInView} delay={delay} duration={duration}>
            {children}
          </SlideIn>
        );
      case 'slideLeft':
        return (
          <SlideIn direction="left" trigger={isInView} delay={delay} duration={duration}>
            {children}
          </SlideIn>
        );
      case 'slideRight':
        return (
          <SlideIn direction="right" trigger={isInView} delay={delay} duration={duration}>
            {children}
          </SlideIn>
        );
      case 'scaleIn':
        return (
          <ScaleIn trigger={isInView} delay={delay} duration={duration}>
            {children}
          </ScaleIn>
        );
      case 'fadeIn':
      default:
        return (
          <FadeIn trigger={isInView} delay={delay} duration={duration}>
            {children}
          </FadeIn>
        );
    }
  };

  return (
    <div ref={ref} className={className}>
      {renderAnimation()}
    </div>
  );
};

// Page transition wrapper
export interface PageTransitionProps {
  children: React.ReactNode;
  className?: string;
}

export const PageTransition: React.FC<PageTransitionProps> = ({ children, className }) => {
  return (
    <div className={clsx('page-transition', className)}>
      <FadeIn duration={200}>
        <SlideIn direction="up" distance={10} duration={300}>
          {children}
        </SlideIn>
      </FadeIn>
    </div>
  );
};