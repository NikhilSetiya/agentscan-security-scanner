import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { 
  LayoutDashboard, 
  Search, 
  AlertTriangle, 
  Settings, 
  FileText,
  Shield,
  Activity
} from 'lucide-react';
import { clsx } from 'clsx';
import './Sidebar.css';

export interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
}

const navigationItems = [
  {
    label: 'Dashboard',
    href: '/',
    icon: LayoutDashboard,
  },
  {
    label: 'Scans',
    href: '/scans',
    icon: Search,
  },
  {
    label: 'Findings',
    href: '/findings',
    icon: AlertTriangle,
  },
  {
    label: 'Reports',
    href: '/reports',
    icon: FileText,
  },
  {
    label: 'Security',
    href: '/security',
    icon: Shield,
  },
  {
    label: 'Activity',
    href: '/activity',
    icon: Activity,
  },
  {
    label: 'Settings',
    href: '/settings',
    icon: Settings,
  },
];

export const Sidebar: React.FC<SidebarProps> = ({ isOpen, onClose }) => {
  const location = useLocation();

  return (
    <>
      {/* Mobile overlay */}
      {isOpen && <div className="sidebar-overlay" onClick={onClose} />}
      
      <aside className={clsx('sidebar', { 'sidebar-open': isOpen })}>
        <div className="sidebar-content">
          <nav className="sidebar-nav">
            <div className="sidebar-section">
              <h3 className="sidebar-section-title">Navigation</h3>
              <ul className="sidebar-menu">
                {navigationItems.map((item) => {
                  const Icon = item.icon;
                  const isActive = location.pathname === item.href;
                  
                  return (
                    <li key={item.href}>
                      <Link
                        to={item.href}
                        className={clsx('sidebar-item', { 'sidebar-item-active': isActive })}
                        onClick={() => {
                          // Close sidebar on mobile when item is clicked
                          if (window.innerWidth <= 768) {
                            onClose();
                          }
                        }}
                      >
                        <Icon size={18} />
                        <span>{item.label}</span>
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          </nav>
        </div>
      </aside>
    </>
  );
};