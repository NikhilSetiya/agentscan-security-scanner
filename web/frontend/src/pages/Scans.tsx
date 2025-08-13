import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/Card';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/Table';
import { Button } from '../components/ui/Button';
import { SearchInput } from '../components/ui/SearchInput';
import { TableSkeleton } from '../components/ui/LoadingSkeleton';
import { ErrorState } from '../components/ui/ErrorState';
import { FadeIn, AnimateOnScroll } from '../components/ui/Transitions';
import { 
  RotateCcw, 
  Download, 
  Plus,
  Clock,
  CheckCircle,
  XCircle,
  Calendar,
  GitBranch,
  User,
  ExternalLink
} from 'lucide-react';
import './Scans.css';

// Mock data for scans
const mockScans = [
  {
    id: 'scan-001',
    repository: 'frontend/web-app',
    branch: 'main',
    status: 'completed',
    progress: 100,
    findings: { high: 2, medium: 8, low: 3, total: 13 },
    duration: '2m 34s',
    startedAt: '2024-01-15T10:30:00Z',
    completedAt: '2024-01-15T10:32:34Z',
    triggeredBy: 'john.doe@company.com',
    agents: ['semgrep', 'eslint', 'bandit'],
    commit: 'a1b2c3d',
  },
  {
    id: 'scan-002',
    repository: 'backend/api-service',
    branch: 'develop',
    status: 'running',
    progress: 65,
    findings: null,
    duration: null,
    startedAt: '2024-01-15T10:25:00Z',
    completedAt: null,
    triggeredBy: 'jane.smith@company.com',
    agents: ['semgrep', 'gosec', 'nancy'],
    commit: 'e4f5g6h',
  },
  {
    id: 'scan-003',
    repository: 'mobile/ios-app',
    branch: 'feature/auth',
    status: 'completed',
    progress: 100,
    findings: { high: 0, medium: 4, low: 12, total: 16 },
    duration: '1m 45s',
    startedAt: '2024-01-15T09:15:00Z',
    completedAt: '2024-01-15T09:16:45Z',
    triggeredBy: 'mike.wilson@company.com',
    agents: ['semgrep', 'mobsf'],
    commit: 'i7j8k9l',
  },
  {
    id: 'scan-004',
    repository: 'infrastructure/terraform',
    branch: 'main',
    status: 'failed',
    progress: 0,
    findings: null,
    duration: null,
    startedAt: '2024-01-15T08:45:00Z',
    completedAt: '2024-01-15T08:47:12Z',
    triggeredBy: 'sarah.johnson@company.com',
    agents: ['tfsec', 'checkov'],
    commit: 'm1n2o3p',
  },
  {
    id: 'scan-005',
    repository: 'docs/documentation',
    branch: 'main',
    status: 'completed',
    progress: 100,
    findings: { high: 0, medium: 0, low: 1, total: 1 },
    duration: '45s',
    startedAt: '2024-01-15T08:00:00Z',
    completedAt: '2024-01-15T08:00:45Z',
    triggeredBy: 'alex.brown@company.com',
    agents: ['semgrep'],
    commit: 'q4r5s6t',
  },
];

const StatusBadge: React.FC<{ status: string; progress?: number }> = ({ status, progress }) => {
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'completed':
        return { icon: CheckCircle, color: 'var(--color-success)', bg: '#dcfce7', text: 'Completed' };
      case 'running':
        return { icon: Clock, color: 'var(--color-warning)', bg: '#fef3c7', text: `Running (${progress}%)` };
      case 'failed':
        return { icon: XCircle, color: 'var(--color-error)', bg: '#fee2e2', text: 'Failed' };
      case 'queued':
        return { icon: Clock, color: 'var(--color-info)', bg: '#dbeafe', text: 'Queued' };
      default:
        return { icon: Clock, color: 'var(--color-gray-500)', bg: 'var(--color-gray-100)', text: status };
    }
  };

  const config = getStatusConfig(status);
  const Icon = config.icon;

  return (
    <div className="status-badge-container">
      <span 
        className="status-badge"
        style={{ 
          backgroundColor: config.bg,
          color: config.color,
        }}
      >
        <Icon size={14} />
        {config.text}
      </span>
      {status === 'running' && progress && (
        <div className="progress-bar">
          <div 
            className="progress-fill" 
            style={{ width: `${progress}%` }}
          />
        </div>
      )}
    </div>
  );
};

const FindingsSummary: React.FC<{ findings: any }> = ({ findings }) => {
  if (!findings) return <span className="text-gray-400">-</span>;

  return (
    <div className="findings-summary">
      <div className="findings-total">{findings.total} total</div>
      <div className="findings-breakdown">
        {findings.high > 0 && (
          <span className="finding-count finding-high">
            {findings.high} High
          </span>
        )}
        {findings.medium > 0 && (
          <span className="finding-count finding-medium">
            {findings.medium} Med
          </span>
        )}
        {findings.low > 0 && (
          <span className="finding-count finding-low">
            {findings.low} Low
          </span>
        )}
        {findings.total === 0 && (
          <span className="finding-count finding-clean">Clean</span>
        )}
      </div>
    </div>
  );
};

export const Scans: React.FC = () => {
  const [scans] = useState(mockScans);
  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState<string | null>(null);
  const [searchValue, setSearchValue] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');

  // Simulate loading
  useEffect(() => {
    const timer = setTimeout(() => {
      setIsLoading(false);
    }, 1000);
    return () => clearTimeout(timer);
  }, []);

  const filteredScans = scans.filter(scan => {
    const matchesSearch = scan.repository.toLowerCase().includes(searchValue.toLowerCase()) ||
                         scan.branch.toLowerCase().includes(searchValue.toLowerCase()) ||
                         scan.triggeredBy.toLowerCase().includes(searchValue.toLowerCase());
    const matchesStatus = statusFilter === 'all' || scan.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

  const handleNewScan = () => {
    // TODO: Implement new scan modal
    console.log('Opening new scan modal...');
  };

  const handleRetryFailed = (scanId: string) => {
    console.log('Retrying scan:', scanId);
    // TODO: Implement retry logic
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const formatDuration = (startTime: string, endTime?: string | null) => {
    if (!endTime) return 'Running...';
    const start = new Date(startTime);
    const end = new Date(endTime);
    const diff = Math.floor((end.getTime() - start.getTime()) / 1000);
    const minutes = Math.floor(diff / 60);
    const seconds = diff % 60;
    return `${minutes}m ${seconds}s`;
  };

  if (error) {
    return (
      <div className="scans-page">
        <ErrorState
          variant="error"
          title="Failed to load scans"
          message={error}
          action={{
            label: 'Try again',
            onClick: () => window.location.reload(),
          }}
        />
      </div>
    );
  }

  return (
    <div className="scans-page">
      <FadeIn>
        <div className="scans-header">
          <div>
            <h1 className="scans-title">Security Scans</h1>
            <p className="scans-subtitle">
              Manage and monitor security scans across all repositories
            </p>
          </div>
          <div className="scans-actions">
            <SearchInput
              value={searchValue}
              onChange={setSearchValue}
              placeholder="Search scans, repositories, branches..."
              className="scans-search"
            />
            <select 
              value={statusFilter} 
              onChange={(e) => setStatusFilter(e.target.value)}
              className="status-filter"
            >
              <option value="all">All Status</option>
              <option value="running">Running</option>
              <option value="completed">Completed</option>
              <option value="failed">Failed</option>
              <option value="queued">Queued</option>
            </select>
            <Button 
              variant="secondary" 
              size="md"
              icon={<Download size={16} />}
              aria-label="Export scan data"
            >
              Export
            </Button>
            <Button 
              variant="primary" 
              size="md"
              icon={<Plus size={16} />}
              onClick={handleNewScan}
              aria-label="Start new security scan"
            >
              New Scan
            </Button>
          </div>
        </div>
      </FadeIn>

      <AnimateOnScroll animation="slideUp" delay={200}>
        <Card className="scans-table-card">
          <CardHeader>
            <CardTitle>Recent Scans ({filteredScans.length})</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <TableSkeleton rows={5} columns={7} />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Repository</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Findings</TableHead>
                    <TableHead>Duration</TableHead>
                    <TableHead>Started</TableHead>
                    <TableHead>Triggered By</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredScans.map((scan) => (
                    <TableRow key={scan.id} className="scan-row">
                      <TableCell>
                        <div className="repository-info">
                          <div className="repository-name">{scan.repository}</div>
                          <div className="repository-details">
                            <GitBranch size={12} />
                            <span>{scan.branch}</span>
                            <span className="commit-hash">#{scan.commit}</span>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={scan.status} progress={scan.progress} />
                      </TableCell>
                      <TableCell>
                        <FindingsSummary findings={scan.findings} />
                      </TableCell>
                      <TableCell>
                        <span className="duration">
                          {scan.duration || formatDuration(scan.startedAt, scan.completedAt)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="timestamp-info">
                          <Calendar size={12} />
                          <span>{formatDate(scan.startedAt)}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="user-info">
                          <User size={12} />
                          <span>{scan.triggeredBy.split('@')[0]}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="scan-actions">
                          <Link to={`/scans/${scan.id}`}>
                            <Button
                              variant="ghost"
                              size="sm"
                              icon={<ExternalLink size={14} />}
                              aria-label="View scan details"
                            >
                              View
                            </Button>
                          </Link>
                          {scan.status === 'failed' && (
                            <Button
                              variant="ghost"
                              size="sm"
                              icon={<RotateCcw size={14} />}
                              onClick={() => handleRetryFailed(scan.id)}
                              aria-label="Retry failed scan"
                            >
                              Retry
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </AnimateOnScroll>
    </div>
  );
};