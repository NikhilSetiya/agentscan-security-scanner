import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { vi } from 'vitest';
import { Dashboard } from './Dashboard';

// Mock the API hooks
vi.mock('../hooks/useApi', () => ({
  useDashboardStats: vi.fn(() => ({
    data: {
      total_scans: 150,
      total_repositories: 25,
      findings_by_severity: {
        critical: 5,
        high: 15,
        medium: 30,
        low: 20,
        info: 10,
      },
      recent_scans: [
        {
          id: 'scan-1',
          repository: { name: 'frontend/web-app' },
          status: 'completed',
          findings_count: 5,
          started_at: '2024-01-01T10:00:00Z',
          completed_at: '2024-01-01T10:05:00Z',
          repository_id: 'repo-1',
        },
        {
          id: 'scan-2',
          repository: { name: 'backend/api' },
          status: 'running',
          findings_count: 0,
          started_at: '2024-01-01T11:00:00Z',
          repository_id: 'repo-2',
        },
      ],
      trend_data: [
        { date: '2024-01-01', high: 5, medium: 10, low: 15 },
        { date: '2024-01-02', high: 3, medium: 8, low: 12 },
      ],
    },
    loading: false,
    error: null,
    execute: vi.fn(),
  })),
}));

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