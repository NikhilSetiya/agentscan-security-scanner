import { render, screen, waitFor } from '@testing-library/react';
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

  it.skip('renders statistics cards', async () => {
    render(<DashboardWithRouter />);
    await waitFor(() => {
      expect(screen.getByText('Total Scans')).toBeInTheDocument();
    }, { timeout: 2000 });
    expect(screen.getByText('High Severity')).toBeInTheDocument();
    expect(screen.getByText('Medium Severity')).toBeInTheDocument();
    expect(screen.getByText('Low Severity')).toBeInTheDocument();
  });

  it.skip('renders recent scans table', async () => {
    render(<DashboardWithRouter />);
    expect(screen.getByText('Recent Scans')).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText('Repository')).toBeInTheDocument();
    }, { timeout: 2000 });
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