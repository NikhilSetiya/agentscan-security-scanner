import React from 'react';
import { Link } from 'react-router-dom';
import { Menu, Bell, User, Settings } from 'lucide-react';
import { Button } from '../ui/Button';
import './TopNavigation.css';

export interface TopNavigationProps {
  onMenuToggle: () => void;
  isSidebarOpen: boolean;
}

export const TopNavigation: React.FC<TopNavigationProps> = ({ onMenuToggle, isSidebarOpen }) => {
  return (
    <nav className="top-nav">
      <div className="nav-left">
        <Button
          variant="ghost"
          size="sm"
          onClick={onMenuToggle}
          className="nav-menu-toggle"
          aria-label={isSidebarOpen ? 'Close sidebar' : 'Open sidebar'}
        >
          <Menu size={20} />
        </Button>
        <Link to="/" className="nav-logo">
          AgentScan
        </Link>
      </div>

      <div className="nav-right">
        <Button variant="ghost" size="sm" aria-label="Notifications">
          <Bell size={20} />
        </Button>
        <Button variant="ghost" size="sm" aria-label="Settings">
          <Settings size={20} />
        </Button>
        <div className="nav-user">
          <Button variant="ghost" size="sm" className="nav-user-button">
            <User size={20} />
            <span className="nav-user-name">John Doe</span>
          </Button>
        </div>
      </div>
    </nav>
  );
};