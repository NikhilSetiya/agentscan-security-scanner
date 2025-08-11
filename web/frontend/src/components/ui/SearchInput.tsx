import React, { useState, useRef, useEffect } from 'react';
import { Search, X } from 'lucide-react';
import { clsx } from 'clsx';
import './SearchInput.css';

export interface SearchInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange' | 'size'> {
  value?: string;
  onChange?: (value: string) => void;
  onClear?: () => void;
  placeholder?: string;
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  showClearButton?: boolean;
  className?: string;
}

export const SearchInput: React.FC<SearchInputProps> = ({
  value = '',
  onChange,
  onClear,
  placeholder = 'Search...',
  size = 'md',
  loading = false,
  showClearButton = true,
  className,
  disabled,
  ...props
}) => {
  const [internalValue, setInternalValue] = useState(value);
  const inputRef = useRef<HTMLInputElement>(null);

  // Update internal value when prop changes
  useEffect(() => {
    setInternalValue(value);
  }, [value]);

  // Focus input when '/' key is pressed globally
  useEffect(() => {
    const handleGlobalKeyDown = (event: KeyboardEvent) => {
      if (event.key === '/' && !event.ctrlKey && !event.metaKey) {
        const target = event.target as HTMLElement;
        // Don't focus if user is already typing in an input
        if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA' && target.contentEditable !== 'true') {
          event.preventDefault();
          inputRef.current?.focus();
        }
      }
    };

    document.addEventListener('keydown', handleGlobalKeyDown);
    return () => document.removeEventListener('keydown', handleGlobalKeyDown);
  }, []);

  const handleInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = event.target.value;
    setInternalValue(newValue);
    onChange?.(newValue);
  };

  const handleClear = () => {
    setInternalValue('');
    onChange?.('');
    onClear?.();
    inputRef.current?.focus();
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Escape') {
      if (internalValue) {
        handleClear();
      } else {
        inputRef.current?.blur();
      }
    }
  };

  return (
    <div className={clsx('search-input', `search-input-${size}`, className)}>
      <div className="search-input-icon">
        <Search size={size === 'sm' ? 14 : size === 'lg' ? 20 : 16} />
      </div>
      
      <input
        ref={inputRef}
        type="text"
        value={internalValue}
        onChange={handleInputChange}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        disabled={disabled || loading}
        className="search-input-field"
        data-search-input
        aria-label="Search"
        {...props}
      />
      
      {loading && (
        <div className="search-input-loading">
          <div className="search-loading-spinner" />
        </div>
      )}
      
      {showClearButton && internalValue && !loading && (
        <button
          type="button"
          onClick={handleClear}
          className="search-input-clear"
          aria-label="Clear search"
          tabIndex={-1}
        >
          <X size={size === 'sm' ? 14 : size === 'lg' ? 20 : 16} />
        </button>
      )}
      
      <div className="search-input-shortcut">
        <kbd>/</kbd>
      </div>
    </div>
  );
};