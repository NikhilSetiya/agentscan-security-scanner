import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { Dashboard } from './Dashboard';

const DashboardWithRouter = () => (
  <BrowserRouter>
    <Dashboard />
  </BrowserRouter>
);

describe('Dashboard', () => {
  it('renders dashboard title', () => {
    render(<DashboardWithRouter />);
    expect(screen.getByText('Security Dashboard')).toBeInTheDocument();
  });

  it('renders statistics cards', () => {
    render(<DashboardWithRouter />);
    expect(screen.getByText('Total Scans')).toBeInTheDocument();
    expect(screen.getByText('High Severity')).toBeInTheDocument();
    expect(screen.getByText('Medium Severity')).toBeInTheDocument();
    expect(screen.getByText('Low Severity')).toBeInTheDocument();
  });

  it('renders recent scans table', () => {
    render(<DashboardWithRouter />);
    expect(screen.getByText('Recent Scans')).toBeInTheDocument();
    expect(screen.getByText('Repository')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Findings')).toBeInTheDocument();
  });

  it('renders findings trend chart', () => {
    render(<DashboardWithRouter />);
    expect(screen.getByText('Findings Trend')).toBeInTheDocument();
  });

  it('displays action buttons', () => {
    render(<DashboardWithRouter />);
    expect(screen.getByText('Filter')).toBeInTheDocument();
    expect(screen.getByText('Export')).toBeInTheDocument();
    expect(screen.getByText('New Scan')).toBeInTheDocument();
  });
});