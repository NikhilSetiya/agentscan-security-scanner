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

// Mock the API hooks
vi.mock('../hooks/useApi', () => ({
  useScanResults: vi.fn(() => ({
    data: {
      scan: {
        id: 'test-scan-id',
        repository: { name: 'frontend/web-app' },
        branch: 'main',
        commit: 'abc123def456',
        status: 'completed',
        started_at: '2024-01-01T10:00:00Z',
        completed_at: '2024-01-01T10:05:00Z',
        repository_id: 'repo-1',
      },
      findings: [
        {
          id: 'finding-1',
          rule: 'XSS-001',
          title: 'Cross-Site Scripting (XSS) vulnerability',
          description: 'Potential XSS vulnerability detected',
          severity: 'high',
          file_path: 'src/components/UserProfile.tsx',
          line_number: 42,
          tool: 'eslint-security',
          tools: ['eslint-security', 'semgrep'],
          status: 'open',
          confidence: 95,
          code_snippet: 'const userInput = req.body.input;\nres.send(`<div>${userInput}</div>`);',
          fix_suggestion: 'Use proper HTML escaping or a templating library that escapes by default.',
        },
      ],
      statistics: {
        total: 1,
        by_severity: { high: 1, medium: 0, low: 0 },
        by_status: { open: 1 },
        by_tool: { 'eslint-security': 1 },
      },
    },
    loading: false,
    error: null,
    execute: vi.fn(),
  })),
  useWebSocket: vi.fn(() => ({
    isConnected: false,
    connectionState: 'connecting',
    lastMessage: null,
    sendMessage: vi.fn(),
  })),
}));

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