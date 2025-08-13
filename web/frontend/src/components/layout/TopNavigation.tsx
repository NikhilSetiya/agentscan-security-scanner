import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { Menu, Bell, User, Settings, LogOut } from 'lucide-react';
import { Button } from '../ui/Button';
import { SearchInput } from '../ui/SearchInput';
import { useAuth } from '../../contexts/AuthContext';
import './TopNavigation.css';

export interface TopNavigationProps {
  onMenuToggle: () => void;
  isSidebarOpen: boolean;
}

export const TopNavigation: React.FC<TopNavigationProps> = ({ onMenuToggle, isSidebarOpen }) => {
  const [searchValue, setSearchValue] = useState('');
  const [showUserMenu, setShowUserMenu] = useState(false);
  const { state, logout } = useAuth();

  const handleSearch = (value: string) => {
    setSearchValue(value);
    // TODO: Implement search functionality
    console.log('Searching for:', value);
  };

  const handleLogout = async () => {
    await logout();
    setShowUserMenu(false);
  };

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

      <div className="nav-center">
        <SearchInput
          value={searchValue}
          onChange={handleSearch}
          placeholder="Search repositories, scans, findings..."
          size="md"
          className="nav-search"
        />
      </div>

      <div className="nav-right">
        <Button 
          variant="ghost" 
          size="sm" 
          aria-label="Notifications"
          className="nav-action-button"
        >
          <Bell size={20} />
        </Button>
        <Button 
          variant="ghost" 
          size="sm" 
          aria-label="Settings"
          className="nav-action-button"
        >
          <Settings size={20} />
        </Button>
        <div className="nav-user">
          <div className="user-menu-container">
            <Button 
              variant="ghost" 
              size="sm" 
              className="nav-user-button"
              onClick={() => setShowUserMenu(!showUserMenu)}
            >
              <User size={20} />
              <span className="nav-user-name">
                {state.user?.username || 'User'}
              </span>
            </Button>
            
            {showUserMenu && (
              <div className="user-menu">
                <div className="user-menu-header">
                  <div className="user-info">
                    <div className="user-name">{state.user?.username}</div>
                    <div className="user-email">{state.user?.email}</div>
                  </div>
                </div>
                <div className="user-menu-divider"></div>
                <div className="user-menu-items">
                  <Link to="/settings" className="user-menu-item" onClick={() => setShowUserMenu(false)}>
                    <Settings size={16} />
                    Settings
                  </Link>
                  <button className="user-menu-item" onClick={handleLogout}>
                    <LogOut size={16} />
                    Sign Out
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
};