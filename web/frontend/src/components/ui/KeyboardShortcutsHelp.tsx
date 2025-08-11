import React, { useState, useEffect } from 'react';
import { Keyboard, X } from 'lucide-react';
import { Modal, ModalHeader, ModalTitle, ModalContent } from './Modal';
import { Button } from './Button';
import { formatShortcut, KeyboardShortcut } from '../../hooks/useKeyboardShortcuts';
import './KeyboardShortcutsHelp.css';

export interface KeyboardShortcutsHelpProps {
  shortcuts: KeyboardShortcut[];
  isOpen: boolean;
  onClose: () => void;
}

export const KeyboardShortcutsHelp: React.FC<KeyboardShortcutsHelpProps> = ({
  shortcuts,
  isOpen,
  onClose,
}) => {
  // Group shortcuts by category
  const groupedShortcuts = shortcuts.reduce((groups, shortcut) => {
    const category = shortcut.category || 'General';
    if (!groups[category]) {
      groups[category] = [];
    }
    groups[category].push(shortcut);
    return groups;
  }, {} as Record<string, KeyboardShortcut[]>);

  return (
    <Modal isOpen={isOpen} onClose={onClose} className="shortcuts-help-modal">
      <ModalHeader>
        <div className="shortcuts-help-header">
          <div className="shortcuts-help-title-container">
            <Keyboard size={24} />
            <ModalTitle>Keyboard Shortcuts</ModalTitle>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            aria-label="Close shortcuts help"
          >
            <X size={16} />
          </Button>
        </div>
      </ModalHeader>
      
      <ModalContent>
        <div className="shortcuts-help-content">
          {Object.entries(groupedShortcuts).map(([category, categoryShortcuts]) => (
            <div key={category} className="shortcuts-category">
              <h3 className="shortcuts-category-title">{category}</h3>
              <div className="shortcuts-list">
                {categoryShortcuts.map((shortcut, index) => (
                  <div key={`${category}-${index}`} className="shortcut-item">
                    <span className="shortcut-description">{shortcut.description}</span>
                    <kbd className="shortcut-keys">{formatShortcut(shortcut)}</kbd>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
        
        <div className="shortcuts-help-footer">
          <p className="shortcuts-help-note">
            Press <kbd>?</kbd> anytime to show this help
          </p>
        </div>
      </ModalContent>
    </Modal>
  );
};

// Global shortcuts help component that listens for the show-shortcuts-help event
export const GlobalShortcutsHelp: React.FC<{ shortcuts: KeyboardShortcut[] }> = ({ shortcuts }) => {
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    const handleShowShortcuts = () => setIsOpen(true);
    
    document.addEventListener('show-shortcuts-help', handleShowShortcuts);
    
    return () => {
      document.removeEventListener('show-shortcuts-help', handleShowShortcuts);
    };
  }, []);

  return (
    <KeyboardShortcutsHelp
      shortcuts={shortcuts}
      isOpen={isOpen}
      onClose={() => setIsOpen(false)}
    />
  );
};