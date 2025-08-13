import React, { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/Card';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/Table';
import { Button } from '../components/ui/Button';
import { LoadingSkeleton } from '../components/ui/LoadingSkeleton';
import { NetworkError } from '../components/ui/ErrorState';
import { useScanResults, useWebSocket } from '../hooks/useApi';
import { 
  ArrowLeft,
  GitBranch,
  GitCommit,
  Clock,
  CheckCircle,
  AlertTriangle,
  Download,
  Search,
  ChevronDown,
  ExternalLink,
  Eye,
  EyeOff
} from 'lucide-react';
import './ScanResults.css';

// Utility functions
const calculateDuration = (startTime: string, endTime?: string): string => {
  if (!endTime) return '-';
  const start = new Date(startTime);
  const end = new Date(endTime);
  const diffMs = end.getTime() - start.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  
  if (diffMins > 0) {
    const remainingSecs = diffSecs % 60;
    return `${diffMins}m ${remainingSecs}s`;
  }
  return `${diffSecs}s`;
};

type SeverityFilter = 'all' | 'high' | 'medium' | 'low';
type StatusFilter = 'all' | 'open' | 'ignored' | 'fixed';
type SortField = 'severity' | 'file' | 'line' | 'rule';
type SortOrder = 'asc' | 'desc';

const SeverityBadge: React.FC<{ severity: string }> = ({ severity }) => {
  const getSeverityConfig = (severity: string) => {
    switch (severity) {
      case 'high':
        return { color: 'var(--color-error)', bg: '#fee2e2', icon: '🔴' };
      case 'medium':
        return { color: 'var(--color-warning)', bg: '#fef3c7', icon: '🟡' };
      case 'low':
        return { color: 'var(--color-info)', bg: '#dbeafe', icon: '🔵' };
      default:
        return { color: 'var(--color-gray-500)', bg: 'var(--color-gray-100)', icon: '⚪' };
    }
  };

  const config = getSeverityConfig(severity);

  return (
    <span 
      className="severity-badge"
      style={{ 
        backgroundColor: config.bg,
        color: config.color,
      }}
    >
      <span className="severity-icon">{config.icon}</span>
      {severity.charAt(0).toUpperCase() + severity.slice(1)}
    </span>
  );
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'open':
        return { color: 'var(--color-error)', bg: '#fee2e2' };
      case 'ignored':
        return { color: 'var(--color-gray-600)', bg: 'var(--color-gray-100)' };
      case 'fixed':
        return { color: 'var(--color-success)', bg: '#dcfce7' };
      default:
        return { color: 'var(--color-gray-500)', bg: 'var(--color-gray-100)' };
    }
  };

  const config = getStatusConfig(status);

  return (
    <span 
      className="status-badge"
      style={{ 
        backgroundColor: config.bg,
        color: config.color,
      }}
    >
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
};

const ToolsList: React.FC<{ finding: any; totalTools?: number }> = ({ finding, totalTools = 5 }) => {
  const tools = finding.tools || [finding.tool].filter(Boolean);
  
  return (
    <div className="tools-list">
      <span className="tools-count">{tools.length}/{totalTools}</span>
      <div className="tools-names">
        {tools.map((tool: string, index: number) => (
          <span key={tool} className="tool-name">
            {tool}
            {index < tools.length - 1 && ', '}
          </span>
        ))}
      </div>
    </div>
  );
};

const FindingRow: React.FC<{ 
  finding: any; 
  onToggleDetails: (id: string) => void; 
  showDetails: boolean;
}> = ({ finding, onToggleDetails, showDetails }) => {
  return (
    <>
      <TableRow className="finding-row">
        <TableCell>
          <SeverityBadge severity={finding.severity} />
        </TableCell>
        <TableCell>
          <div className="rule-info">
            <span className="rule-id">{finding.rule}</span>
            <span className="rule-title">{finding.title}</span>
          </div>
        </TableCell>
        <TableCell>
          <div className="file-location">
            <span className="file-path">{finding.file_path}</span>
            <span className="line-number">:{finding.line_number}</span>
          </div>
        </TableCell>
        <TableCell>
          <ToolsList finding={finding} />
        </TableCell>
        <TableCell>
          <StatusBadge status={finding.status} />
        </TableCell>
        <TableCell>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onToggleDetails(finding.id)}
            className="details-toggle"
          >
            {showDetails ? <EyeOff size={16} /> : <Eye size={16} />}
            {showDetails ? 'Hide' : 'Details'}
          </Button>
        </TableCell>
      </TableRow>
      {showDetails && (
        <TableRow className="finding-details-row">
          <TableCell colSpan={6}>
            <div className="finding-details">
              <div className="finding-description">
                <h4>Description</h4>
                <p>{finding.description}</p>
              </div>
              
              {finding.code_snippet && (
                <div className="code-snippet">
                  <h4>Code Snippet</h4>
                  <pre><code>{finding.code_snippet}</code></pre>
                </div>
              )}
              
              {finding.fix_suggestion && (
                <div className="fix-suggestion">
                  <h4>Suggested Fix</h4>
                  <p>{finding.fix_suggestion}</p>
                </div>
              )}
              
              <div className="finding-actions">
                <Button variant="primary" size="sm">
                  Mark as Fixed
                </Button>
                <Button variant="secondary" size="sm">
                  Ignore Finding
                </Button>
                <Button variant="ghost" size="sm">
                  <ExternalLink size={16} />
                  View in Editor
                </Button>
              </div>
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
};

export const ScanResults: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { data: scanResults, loading, error, execute: refetchResults } = useScanResults(id);
  const { isConnected } = useWebSocket(`ws://localhost:8080/ws/scans/${id}`, !!id);
  
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [sortField, setSortField] = useState<SortField>('severity');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedFindings, setExpandedFindings] = useState<Set<string>>(new Set());

  // Extract data with fallbacks
  const scanData = scanResults?.scan;
  const findings = scanResults?.findings || [];

  const handleToggleDetails = (findingId: string) => {
    const newExpanded = new Set(expandedFindings);
    if (newExpanded.has(findingId)) {
      newExpanded.delete(findingId);
    } else {
      newExpanded.add(findingId);
    }
    setExpandedFindings(newExpanded);
  };

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('desc');
    }
  };

  const handleExport = (format: 'pdf' | 'json') => {
    // Mock export functionality
    console.log(`Exporting scan results as ${format}`);
    // In a real implementation, this would trigger a download
  };

  // Filter and sort findings
  const filteredFindings = findings
    .filter(finding => {
      if (severityFilter !== 'all' && finding.severity !== severityFilter) return false;
      if (statusFilter !== 'all' && finding.status !== statusFilter) return false;
      if (searchQuery && !finding.title.toLowerCase().includes(searchQuery.toLowerCase()) &&
          !finding.file_path.toLowerCase().includes(searchQuery.toLowerCase()) &&
          !finding.rule_id.toLowerCase().includes(searchQuery.toLowerCase())) return false;
      return true;
    })
    .sort((a, b) => {
      let aValue: string | number;
      let bValue: string | number;

      switch (sortField) {
        case 'severity':
          const severityOrder = { critical: 5, high: 4, medium: 3, low: 2, info: 1 };
          aValue = severityOrder[a.severity as keyof typeof severityOrder] || 0;
          bValue = severityOrder[b.severity as keyof typeof severityOrder] || 0;
          break;
        case 'file':
          aValue = a.file_path;
          bValue = b.file_path;
          break;
        case 'line':
          aValue = a.line_number;
          bValue = b.line_number;
          break;
        case 'rule':
          aValue = a.rule_id;
          bValue = b.rule_id;
          break;
        default:
          return 0;
      }

      if (sortOrder === 'asc') {
        return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
      } else {
        return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
      }
    });

  const findingStats = scanResults?.statistics || {
    total: findings.length,
    by_severity: {
      critical: findings.filter(f => f.severity === 'critical').length,
      high: findings.filter(f => f.severity === 'high').length,
      medium: findings.filter(f => f.severity === 'medium').length,
      low: findings.filter(f => f.severity === 'low').length,
      info: findings.filter(f => f.severity === 'info').length,
    },
    by_status: {
      open: findings.filter(f => f.status === 'open').length,
    },
    by_tool: {},
  };

  if (loading) {
    return (
      <div className="scan-results">
        <div className="scan-header">
          <div className="header-navigation">
            <Link to="/scans" className="back-link">
              <ArrowLeft size={16} />
              Back to Scans
            </Link>
          </div>
          <Card>
            <CardContent>
              <LoadingSkeleton height={120} />
            </CardContent>
          </Card>
        </div>
        <Card>
          <CardContent>
            <LoadingSkeleton height={60} />
          </CardContent>
        </Card>
        <Card>
          <CardContent>
            <LoadingSkeleton height={400} />
          </CardContent>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className="scan-results">
        <div className="scan-header">
          <div className="header-navigation">
            <Link to="/scans" className="back-link">
              <ArrowLeft size={16} />
              Back to Scans
            </Link>
          </div>
        </div>
        <NetworkError 
          onRetry={refetchResults}
        />
      </div>
    );
  }

  if (!scanData) {
    return (
      <div className="scan-results">
        <div className="scan-header">
          <div className="header-navigation">
            <Link to="/scans" className="back-link">
              <ArrowLeft size={16} />
              Back to Scans
            </Link>
          </div>
        </div>
        <Card>
          <CardContent>
            <div className="empty-state">
              <AlertTriangle size={48} />
              <h3>Scan not found</h3>
              <p>The requested scan could not be found.</p>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="scan-results">
      {/* Header */}
      <div className="scan-header">
        <div className="header-navigation">
          <Link to="/scans" className="back-link">
            <ArrowLeft size={16} />
            Back to Scans
          </Link>
        </div>
        
        <Card>
          <CardContent>
            <div className="scan-info">
              <div className="scan-basic-info">
                <h1 className="scan-title">
                  {scanData.repository?.name || `Repository ${scanData.repository_id}`}
                </h1>
                <div className="scan-meta">
                  <div className="meta-item">
                    <GitBranch size={16} />
                    <span>{scanData.branch}</span>
                  </div>
                  <div className="meta-item">
                    <GitCommit size={16} />
                    <span>{scanData.commit}</span>
                  </div>
                  <div className="meta-item">
                    <Clock size={16} />
                    <span>{calculateDuration(scanData.started_at, scanData.completed_at)}</span>
                  </div>
                  <div className="meta-item">
                    <CheckCircle size={16} />
                    <span className={`status-${scanData.status}`}>
                      {scanData.status.charAt(0).toUpperCase() + scanData.status.slice(1)}
                    </span>
                  </div>
                </div>
              </div>
              
              <div className="scan-stats">
                <div className="stat-item">
                  <span className="stat-value">{findingStats.total}</span>
                  <span className="stat-label">Total Findings</span>
                </div>
                <div className="stat-item stat-high">
                  <span className="stat-value">{findingStats.by_severity?.high || 0}</span>
                  <span className="stat-label">High</span>
                </div>
                <div className="stat-item stat-medium">
                  <span className="stat-value">{findingStats.by_severity?.medium || 0}</span>
                  <span className="stat-label">Medium</span>
                </div>
                <div className="stat-item stat-low">
                  <span className="stat-value">{findingStats.by_severity?.low || 0}</span>
                  <span className="stat-label">Low</span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters and Actions */}
      <Card>
        <CardContent>
          <div className="filters-section">
            <div className="filters-left">
              <div className="search-box">
                <Search size={16} />
                <input
                  type="text"
                  placeholder="Search findings..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="search-input"
                />
              </div>
              
              <div className="filter-group">
                <select
                  value={severityFilter}
                  onChange={(e) => setSeverityFilter(e.target.value as SeverityFilter)}
                  className="filter-select"
                >
                  <option value="all">All Severities</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
                
                <select
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
                  className="filter-select"
                >
                  <option value="all">All Status</option>
                  <option value="open">Open</option>
                  <option value="ignored">Ignored</option>
                  <option value="fixed">Fixed</option>
                </select>
              </div>
            </div>
            
            <div className="filters-right">
              <div className="connection-status">
                <div className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`}></div>
                <span>{isConnected ? 'Live Updates' : 'Connecting...'}</span>
              </div>
              
              <Button variant="secondary" size="sm" onClick={() => handleExport('json')}>
                <Download size={16} />
                JSON
              </Button>
              <Button variant="secondary" size="sm" onClick={() => handleExport('pdf')}>
                <Download size={16} />
                PDF
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Findings Table */}
      <Card>
        <CardHeader>
          <CardTitle>
            Findings ({filteredFindings.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>
                  <button 
                    className="sort-button"
                    onClick={() => handleSort('severity')}
                  >
                    Severity
                    {sortField === 'severity' && (
                      <ChevronDown 
                        size={14} 
                        className={sortOrder === 'asc' ? 'rotate-180' : ''} 
                      />
                    )}
                  </button>
                </TableHead>
                <TableHead>
                  <button 
                    className="sort-button"
                    onClick={() => handleSort('rule')}
                  >
                    Rule & Description
                    {sortField === 'rule' && (
                      <ChevronDown 
                        size={14} 
                        className={sortOrder === 'asc' ? 'rotate-180' : ''} 
                      />
                    )}
                  </button>
                </TableHead>
                <TableHead>
                  <button 
                    className="sort-button"
                    onClick={() => handleSort('file')}
                  >
                    File & Line
                    {sortField === 'file' && (
                      <ChevronDown 
                        size={14} 
                        className={sortOrder === 'asc' ? 'rotate-180' : ''} 
                      />
                    )}
                  </button>
                </TableHead>
                <TableHead>Tools</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredFindings.map((finding) => (
                <FindingRow
                  key={finding.id}
                  finding={finding}
                  onToggleDetails={handleToggleDetails}
                  showDetails={expandedFindings.has(finding.id)}
                />
              ))}
            </TableBody>
          </Table>
          
          {filteredFindings.length === 0 && (
            <div className="empty-state">
              <AlertTriangle size={48} />
              <h3>No findings match your filters</h3>
              <p>Try adjusting your search criteria or filters</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};