import React, { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/Card';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/Table';
import { Button } from '../components/ui/Button';
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

// Mock data for demonstration
const mockScanData = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  repository: 'frontend/web-app',
  branch: 'main',
  commit: 'abc123def456',
  commitMessage: 'Add user authentication flow',
  status: 'completed',
  startedAt: '2024-01-07T10:30:00Z',
  completedAt: '2024-01-07T10:32:34Z',
  duration: '2m 34s',
  triggeredBy: 'John Doe',
  scanType: 'full',
};

const mockFindings = [
  {
    id: '1',
    severity: 'high',
    rule: 'XSS-001',
    title: 'Cross-Site Scripting (XSS) vulnerability',
    description: 'User input is not properly sanitized before being rendered in the DOM',
    file: 'src/components/UserProfile.tsx',
    line: 42,
    column: 15,
    tools: ['semgrep', 'eslint', 'bandit'],
    confidence: 0.95,
    status: 'open',
    codeSnippet: `const userBio = props.user.bio;
return <div dangerouslySetInnerHTML={{__html: userBio}} />;`,
    fixSuggestion: 'Use proper HTML sanitization or escape user input before rendering',
  },
  {
    id: '2',
    severity: 'medium',
    rule: 'SQL-002',
    title: 'SQL Injection vulnerability',
    description: 'Database query constructed using string concatenation',
    file: 'src/api/users.ts',
    line: 15,
    column: 8,
    tools: ['semgrep', 'eslint'],
    confidence: 0.87,
    status: 'open',
    codeSnippet: `const query = "SELECT * FROM users WHERE id = " + userId;
const result = await db.query(query);`,
    fixSuggestion: 'Use parameterized queries or prepared statements',
  },
  {
    id: '3',
    severity: 'low',
    rule: 'CRYPTO-003',
    title: 'Weak cryptographic algorithm',
    description: 'MD5 hash function is cryptographically weak',
    file: 'src/utils/hash.ts',
    line: 8,
    column: 20,
    tools: ['semgrep'],
    confidence: 0.72,
    status: 'ignored',
    codeSnippet: `import crypto from 'crypto';
const hash = crypto.createHash('md5').update(data).digest('hex');`,
    fixSuggestion: 'Use SHA-256 or other secure hash functions',
  },
  {
    id: '4',
    severity: 'high',
    rule: 'AUTH-004',
    title: 'Hardcoded API key',
    description: 'API key found in source code',
    file: 'src/config/api.ts',
    line: 3,
    column: 1,
    tools: ['trufflehog', 'git-secrets'],
    confidence: 0.98,
    status: 'open',
    codeSnippet: `const API_KEY = "sk-1234567890abcdef";
const config = { apiKey: API_KEY };`,
    fixSuggestion: 'Move API keys to environment variables',
  },
  {
    id: '5',
    severity: 'medium',
    rule: 'DEPS-005',
    title: 'Vulnerable dependency',
    description: 'Package lodash@4.17.20 has known security vulnerabilities',
    file: 'package.json',
    line: 25,
    column: 5,
    tools: ['npm-audit'],
    confidence: 1.0,
    status: 'open',
    codeSnippet: `"dependencies": {
  "lodash": "4.17.20"
}`,
    fixSuggestion: 'Update to lodash@4.17.21 or later',
  },
];

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

const ToolsList: React.FC<{ tools: string[]; totalTools?: number }> = ({ tools, totalTools = 5 }) => {
  return (
    <div className="tools-list">
      <span className="tools-count">{tools.length}/{totalTools}</span>
      <div className="tools-names">
        {tools.map((tool, index) => (
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
  finding: typeof mockFindings[0]; 
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
            <span className="file-path">{finding.file}</span>
            <span className="line-number">:{finding.line}</span>
          </div>
        </TableCell>
        <TableCell>
          <ToolsList tools={finding.tools} />
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
              
              <div className="code-snippet">
                <h4>Code Snippet</h4>
                <pre><code>{finding.codeSnippet}</code></pre>
              </div>
              
              <div className="fix-suggestion">
                <h4>Suggested Fix</h4>
                <p>{finding.fixSuggestion}</p>
              </div>
              
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
  const [findings] = useState(mockFindings);
  
  // Use the id parameter (in a real app, this would fetch scan data)
  console.log('Scan ID:', id);
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [sortField, setSortField] = useState<SortField>('severity');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedFindings, setExpandedFindings] = useState<Set<string>>(new Set());
  const [isConnected, setIsConnected] = useState(false);

  // Mock WebSocket connection for real-time updates
  useEffect(() => {
    // Simulate WebSocket connection
    const timer = setTimeout(() => {
      setIsConnected(true);
    }, 1000);

    return () => clearTimeout(timer);
  }, []);

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
          !finding.file.toLowerCase().includes(searchQuery.toLowerCase()) &&
          !finding.rule.toLowerCase().includes(searchQuery.toLowerCase())) return false;
      return true;
    })
    .sort((a, b) => {
      let aValue: string | number;
      let bValue: string | number;

      switch (sortField) {
        case 'severity':
          const severityOrder = { high: 3, medium: 2, low: 1 };
          aValue = severityOrder[a.severity as keyof typeof severityOrder];
          bValue = severityOrder[b.severity as keyof typeof severityOrder];
          break;
        case 'file':
          aValue = a.file;
          bValue = b.file;
          break;
        case 'line':
          aValue = a.line;
          bValue = b.line;
          break;
        case 'rule':
          aValue = a.rule;
          bValue = b.rule;
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

  const findingStats = {
    total: findings.length,
    high: findings.filter(f => f.severity === 'high').length,
    medium: findings.filter(f => f.severity === 'medium').length,
    low: findings.filter(f => f.severity === 'low').length,
    open: findings.filter(f => f.status === 'open').length,
  };

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
                <h1 className="scan-title">{mockScanData.repository}</h1>
                <div className="scan-meta">
                  <div className="meta-item">
                    <GitBranch size={16} />
                    <span>{mockScanData.branch}</span>
                  </div>
                  <div className="meta-item">
                    <GitCommit size={16} />
                    <span>{mockScanData.commit}</span>
                  </div>
                  <div className="meta-item">
                    <Clock size={16} />
                    <span>{mockScanData.duration}</span>
                  </div>
                  <div className="meta-item">
                    <CheckCircle size={16} />
                    <span className="status-completed">Completed</span>
                  </div>
                </div>
              </div>
              
              <div className="scan-stats">
                <div className="stat-item">
                  <span className="stat-value">{findingStats.total}</span>
                  <span className="stat-label">Total Findings</span>
                </div>
                <div className="stat-item stat-high">
                  <span className="stat-value">{findingStats.high}</span>
                  <span className="stat-label">High</span>
                </div>
                <div className="stat-item stat-medium">
                  <span className="stat-value">{findingStats.medium}</span>
                  <span className="stat-label">Medium</span>
                </div>
                <div className="stat-item stat-low">
                  <span className="stat-value">{findingStats.low}</span>
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