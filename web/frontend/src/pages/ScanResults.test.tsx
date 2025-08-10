import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { vi } from 'vitest';
import { ScanResults } from './ScanResults';

// Mock useParams
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ id: 'test-scan-id' }),
  };
});

const ScanResultsWithRouter = () => (
  <BrowserRouter>
    <ScanResults />
  </BrowserRouter>
);

describe('ScanResults', () => {
  it('renders scan header with repository info', () => {
    render(<ScanResultsWithRouter />);
    expect(screen.getByText('frontend/web-app')).toBeInTheDocument();
    expect(screen.getByText('main')).toBeInTheDocument();
    expect(screen.getByText('abc123def456')).toBeInTheDocument();
  });

  it('displays scan statistics', () => {
    render(<ScanResultsWithRouter />);
    expect(screen.getByText('Total Findings')).toBeInTheDocument();
    expect(screen.getAllByText('High').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Medium').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Low').length).toBeGreaterThan(0);
  });

  it('renders findings table with headers', () => {
    render(<ScanResultsWithRouter />);
    expect(screen.getByText('Severity')).toBeInTheDocument();
    expect(screen.getByText('Rule & Description')).toBeInTheDocument();
    expect(screen.getByText('File & Line')).toBeInTheDocument();
    expect(screen.getByText('Tools')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
  });

  it('displays findings data', () => {
    render(<ScanResultsWithRouter />);
    expect(screen.getByText('XSS-001')).toBeInTheDocument();
    expect(screen.getByText('Cross-Site Scripting (XSS) vulnerability')).toBeInTheDocument();
    expect(screen.getByText('src/components/UserProfile.tsx')).toBeInTheDocument();
  });

  it('allows filtering by severity', () => {
    render(<ScanResultsWithRouter />);
    const severityFilter = screen.getByDisplayValue('All Severities');
    fireEvent.change(severityFilter, { target: { value: 'high' } });
    expect(severityFilter).toHaveValue('high');
  });

  it('allows searching findings', () => {
    render(<ScanResultsWithRouter />);
    const searchInput = screen.getByPlaceholderText('Search findings...');
    fireEvent.change(searchInput, { target: { value: 'XSS' } });
    expect(searchInput).toHaveValue('XSS');
  });

  it('shows export buttons', () => {
    render(<ScanResultsWithRouter />);
    expect(screen.getByText('JSON')).toBeInTheDocument();
    expect(screen.getByText('PDF')).toBeInTheDocument();
  });

  it('displays connection status', () => {
    render(<ScanResultsWithRouter />);
    // Initially shows "Connecting..."
    expect(screen.getByText('Connecting...')).toBeInTheDocument();
  });

  it('allows expanding finding details', async () => {
    render(<ScanResultsWithRouter />);
    const detailsButton = screen.getAllByText('Details')[0];
    fireEvent.click(detailsButton);
    
    // Should show expanded details
    expect(screen.getByText('Description')).toBeInTheDocument();
    expect(screen.getByText('Code Snippet')).toBeInTheDocument();
    expect(screen.getByText('Suggested Fix')).toBeInTheDocument();
  });
});