import { useEffect, useCallback, useRef } from 'react';

export interface KeyboardShortcut {
  key: string;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
  altKey?: boolean;
  action: () => void;
  description: string;
  category?: string;
  preventDefault?: boolean;
}

export interface UseKeyboardShortcutsOptions {
  enabled?: boolean;
  preventDefault?: boolean;
}

export const useKeyboardShortcuts = (
  shortcuts: KeyboardShortcut[],
  options: UseKeyboardShortcutsOptions = {}
) => {
  const { enabled = true, preventDefault = true } = options;
  const shortcutsRef = useRef(shortcuts);
  
  // Update shortcuts ref when shortcuts change
  useEffect(() => {
    shortcutsRef.current = shortcuts;
  }, [shortcuts]);

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (!enabled) return;

    // Don't trigger shortcuts when user is typing in input fields
    const target = event.target as HTMLElement;
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.contentEditable === 'true'
    ) {
      return;
    }

    const matchingShortcut = shortcutsRef.current.find(shortcut => {
      const keyMatches = shortcut.key.toLowerCase() === event.key.toLowerCase();
      const ctrlMatches = !!shortcut.ctrlKey === event.ctrlKey;
      const metaMatches = !!shortcut.metaKey === event.metaKey;
      const shiftMatches = !!shortcut.shiftKey === event.shiftKey;
      const altMatches = !!shortcut.altKey === event.altKey;

      return keyMatches && ctrlMatches && metaMatches && shiftMatches && altMatches;
    });

    if (matchingShortcut) {
      if (preventDefault || matchingShortcut.preventDefault !== false) {
        event.preventDefault();
      }
      matchingShortcut.action();
    }
  }, [enabled, preventDefault]);

  useEffect(() => {
    if (enabled) {
      document.addEventListener('keydown', handleKeyDown);
      return () => document.removeEventListener('keydown', handleKeyDown);
    }
  }, [enabled, handleKeyDown]);

  return {
    shortcuts: shortcutsRef.current,
  };
};

// Hook for managing global application shortcuts
export const useGlobalShortcuts = () => {
  const shortcuts: KeyboardShortcut[] = [
    {
      key: '/',
      action: () => {
        const searchInput = document.querySelector('[data-search-input]') as HTMLInputElement;
        if (searchInput) {
          searchInput.focus();
        }
      },
      description: 'Focus search',
      category: 'Navigation',
    },
    {
      key: 'Escape',
      action: () => {
        // Close any open modals or dropdowns
        const activeElement = document.activeElement as HTMLElement;
        if (activeElement && activeElement.blur) {
          activeElement.blur();
        }
        
        // Dispatch custom event for components to listen to
        document.dispatchEvent(new CustomEvent('global-escape'));
      },
      description: 'Close modals/dropdowns',
      category: 'Navigation',
    },
    {
      key: 'k',
      ctrlKey: true,
      action: () => {
        // Open command palette (if implemented)
        document.dispatchEvent(new CustomEvent('open-command-palette'));
      },
      description: 'Open command palette',
      category: 'Navigation',
    },
    {
      key: 'k',
      metaKey: true, // For Mac users
      action: () => {
        document.dispatchEvent(new CustomEvent('open-command-palette'));
      },
      description: 'Open command palette',
      category: 'Navigation',
    },
    {
      key: 'n',
      ctrlKey: true,
      action: () => {
        document.dispatchEvent(new CustomEvent('new-scan'));
      },
      description: 'Start new scan',
      category: 'Actions',
    },
    {
      key: 'n',
      metaKey: true, // For Mac users
      action: () => {
        document.dispatchEvent(new CustomEvent('new-scan'));
      },
      description: 'Start new scan',
      category: 'Actions',
    },
    {
      key: 'r',
      ctrlKey: true,
      action: () => {
        document.dispatchEvent(new CustomEvent('refresh-data'));
      },
      description: 'Refresh data',
      category: 'Actions',
    },
    {
      key: 'r',
      metaKey: true, // For Mac users
      action: () => {
        document.dispatchEvent(new CustomEvent('refresh-data'));
      },
      description: 'Refresh data',
      category: 'Actions',
    },
    {
      key: '?',
      shiftKey: true,
      action: () => {
        document.dispatchEvent(new CustomEvent('show-shortcuts-help'));
      },
      description: 'Show keyboard shortcuts',
      category: 'Help',
    },
  ];

  return useKeyboardShortcuts(shortcuts);
};

// Utility function to format shortcut display
export const formatShortcut = (shortcut: KeyboardShortcut): string => {
  const parts: string[] = [];
  
  // Detect if user is on Mac
  const isMac = typeof navigator !== 'undefined' && navigator.platform.toUpperCase().indexOf('MAC') >= 0;
  
  if (shortcut.ctrlKey) parts.push(isMac ? '⌘' : 'Ctrl');
  if (shortcut.metaKey) parts.push('⌘');
  if (shortcut.altKey) parts.push(isMac ? '⌥' : 'Alt');
  if (shortcut.shiftKey) parts.push(isMac ? '⇧' : 'Shift');
  
  // Format special keys
  let key = shortcut.key;
  switch (key.toLowerCase()) {
    case 'escape':
      key = 'Esc';
      break;
    case 'arrowup':
      key = '↑';
      break;
    case 'arrowdown':
      key = '↓';
      break;
    case 'arrowleft':
      key = '←';
      break;
    case 'arrowright':
      key = '→';
      break;
    case ' ':
      key = 'Space';
      break;
    default:
      key = key.toUpperCase();
  }
  
  parts.push(key);
  
  return parts.join(isMac ? '' : '+');
};