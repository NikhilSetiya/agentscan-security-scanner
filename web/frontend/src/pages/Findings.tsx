import React, { useState, useEffect } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/Card';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/Table';
import { Button } from '../components/ui/Button';
import { SearchInput } from '../components/ui/SearchInput';
import { TableSkeleton } from '../components/ui/LoadingSkeleton';
import { ErrorState } from '../components/ui/ErrorState';
import { FadeIn, AnimateOnScroll } from '../components/ui/Transitions';
import { 
  AlertTriangle, 
  Shield, 
  Info, 
  Download, 
  ExternalLink,
  FileText,
  Code,
  Calendar,
  Tag,
  CheckCircle,
  XCircle,
  Clock
} from 'lucide-react';
import './Findings.css';

// Mock data for findings
const mockFindings = [
  {
    id: 'finding-001',
    title: 'SQL Injection vulnerability in user authentication',
    description: 'Potential SQL injection vulnerability detected in login endpoint. User input is not properly sanitized before database query.',
    severity: 'high',
    confidence: 95,
    status: 'open',
    category: 'injection',
    cwe: 'CWE-89',
    file: 'src/auth/login.js',
    line: 42,
    repository: 'backend/api-service',
    scanId: 'scan-001',
    detectedBy: ['semgrep', 'eslint'],
    firstSeen: '2024-01-15T10:32:00Z',
    lastSeen: '2024-01-15T10:32:00Z',
    falsePositive: false,
  },
  {
    id: 'finding-002',
    title: 'Hardcoded API key in configuration file',
    description: 'API key is hardcoded in the configuration file. This poses a security risk if the code is exposed.',
    severity: 'high',
    confidence: 98,
    status: 'open',
    category: 'secrets',
    cwe: 'CWE-798',
    file: 'config/production.js',
    line: 15,
    repository: 'backend/api-service',
    scanId: 'scan-001',
    detectedBy: ['semgrep', 'truffleHog'],
    firstSeen: '2024-01-15T10:32:00Z',
    lastSeen: '2024-01-15T10:32:00Z',
    falsePositive: false,
  },
  {
    id: 'finding-003',
    title: 'Cross-Site Scripting (XSS) in user profile',
    description: 'User input in profile page is not properly escaped, allowing potential XSS attacks.',
    severity: 'medium',
    confidence: 87,
    status: 'resolved',
    category: 'xss',
    cwe: 'CWE-79',
    file: 'src/components/UserProfile.jsx',
    line: 128,
    repository: 'frontend/web-app',
    scanId: 'scan-002',
    detectedBy: ['semgrep'],
    firstSeen: '2024-01-14T15:20:00Z',
    lastSeen: '2024-01-15T10:32:00Z',
    falsePositive: false,
  },
  {
    id: 'finding-004',
    title: 'Insecure random number generation',
    description: 'Using Math.random() for security-sensitive operations. Use cryptographically secure random number generator.',
    severity: 'medium',
    confidence: 92,
    status: 'open',
    category: 'crypto',
    cwe: 'CWE-338',
    file: 'src/utils/token.js',
    line: 67,
    repository: 'frontend/web-app',
    scanId: 'scan-002',
    detectedBy: ['eslint'],
    firstSeen: '2024-01-15T10:32:00Z',
    lastSeen: '2024-01-15T10:32:00Z',
    falsePositive: false,
  },
  {
    id: 'finding-005',
    title: 'Missing input validation in API endpoint',
    description: 'API endpoint does not validate input parameters, which could lead to various security issues.',
    severity: 'low',
    confidence: 78,
    status: 'false_positive',
    category: 'validation',
    cwe: 'CWE-20',
    file: 'src/api/users.js',
    line: 203,
    repository: 'backend/api-service',
    scanId: 'scan-001',
    detectedBy: ['semgrep'],
    firstSeen: '2024-01-15T10:32:00Z',
    lastSeen: '2024-01-15T10:32:00Z',
    falsePositive: true,
  },
];

const SeverityBadge: React.FC<{ severity: string }> = ({ severity }) => {
  const getSeverityConfig = (severity: string) => {
    switch (severity) {
      case 'high':
        return { icon: AlertTriangle, color: 'var(--color-error)', bg: '#fee2e2', text: 'High' };
      case 'medium':
        return { icon: Shield, color: 'var(--color-warning)', bg: '#fef3c7', text: 'Medium' };
      case 'low':
        return { icon: Info, color: 'var(--color-info)', bg: '#dbeafe', text: 'Low' };
      default:
        return { icon: Info, color: 'var(--color-gray-500)', bg: 'var(--color-gray-100)', text: severity };
    }
  };

  const config = getSeverityConfig(severity);
  const Icon = config.icon;

  return (
    <span 
      className="severity-badge"
      style={{ 
        backgroundColor: config.bg,
        color: config.color,
      }}
    >
      <Icon size={14} />
      {config.text}
    </span>
  );
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'open':
        return { icon: AlertTriangle, color: 'var(--color-error)', bg: '#fee2e2', text: 'Open' };
      case 'resolved':
        return { icon: CheckCircle, color: 'var(--color-success)', bg: '#dcfce7', text: 'Resolved' };
      case 'false_positive':
        return { icon: XCircle, color: 'var(--color-gray-500)', bg: 'var(--color-gray-100)', text: 'False Positive' };
      case 'in_progress':
        return { icon: Clock, color: 'var(--color-warning)', bg: '#fef3c7', text: 'In Progress' };
      default:
        return { icon: Clock, color: 'var(--color-gray-500)', bg: 'var(--color-gray-100)', text: status };
    }
  };

  const config = getStatusConfig(status);
  const Icon = config.icon;

  return (
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
  );
};

const ConfidenceMeter: React.FC<{ confidence: number }> = ({ confidence }) => {
  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 90) return 'var(--color-success)';
    if (confidence >= 70) return 'var(--color-warning)';
    return 'var(--color-error)';
  };

  return (
    <div className="confidence-meter">
      <div className="confidence-bar">
        <div 
          className="confidence-fill"
          style={{ 
            width: `${confidence}%`,
            backgroundColor: getConfidenceColor(confidence)
          }}
        />
      </div>
      <span className="confidence-text">{confidence}%</span>
    </div>
  );
};

export const Findings: React.FC = () => {
  const [findings, setFindings] = useState(mockFindings);
  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState<string | null>(null);
  const [searchValue, setSearchValue] = useState('');
  const [severityFilter, setSeverityFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');

  // Simulate loading
  useEffect(() => {
    const timer = setTimeout(() => {
      setIsLoading(false);
    }, 1000);
    return () => clearTimeout(timer);
  }, []);

  const filteredFindings = findings.filter(finding => {
    const matchesSearch = finding.title.toLowerCase().includes(searchValue.toLowerCase()) ||
                         finding.description.toLowerCase().includes(searchValue.toLowerCase()) ||
                         finding.file.toLowerCase().includes(searchValue.toLowerCase()) ||
                         finding.repository.toLowerCase().includes(searchValue.toLowerCase());
    const matchesSeverity = severityFilter === 'all' || finding.severity === severityFilter;
    const matchesStatus = statusFilter === 'all' || finding.status === statusFilter;
    return matchesSearch && matchesSeverity && matchesStatus;
  });

  const handleMarkResolved = (findingId: string) => {
    setFindings(prev => prev.map(finding => 
      finding.id === findingId 
        ? { ...finding, status: 'resolved' }
        : finding
    ));
  };

  const handleMarkFalsePositive = (findingId: string) => {
    setFindings(prev => prev.map(finding => 
      finding.id === findingId 
        ? { ...finding, status: 'false_positive', falsePositive: true }
        : finding
    ));
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  const getStats = () => {
    const total = findings.length;
    const open = findings.filter(f => f.status === 'open').length;
    const resolved = findings.filter(f => f.status === 'resolved').length;
    const falsePositives = findings.filter(f => f.status === 'false_positive').length;
    const high = findings.filter(f => f.severity === 'high' && f.status === 'open').length;
    const medium = findings.filter(f => f.severity === 'medium' && f.status === 'open').length;
    const low = findings.filter(f => f.severity === 'low' && f.status === 'open').length;

    return { total, open, resolved, falsePositives, high, medium, low };
  };

  const stats = getStats();

  if (error) {
    return (
      <div className="findings-page">
        <ErrorState
          variant="error"
          title="Failed to load findings"
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
    <div className="findings-page">
      <FadeIn>
        <div className="findings-header">
          <div>
            <h1 className="findings-title">Security Findings</h1>
            <p className="findings-subtitle">
              Review and manage security vulnerabilities across all repositories
            </p>
          </div>
          <div className="findings-actions">
            <SearchInput
              value={searchValue}
              onChange={setSearchValue}
              placeholder="Search findings, files, repositories..."
              className="findings-search"
            />
            <select 
              value={severityFilter} 
              onChange={(e) => setSeverityFilter(e.target.value)}
              className="severity-filter"
            >
              <option value="all">All Severities</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>
            <select 
              value={statusFilter} 
              onChange={(e) => setStatusFilter(e.target.value)}
              className="status-filter"
            >
              <option value="all">All Status</option>
              <option value="open">Open</option>
              <option value="resolved">Resolved</option>
              <option value="false_positive">False Positive</option>
            </select>
            <Button 
              variant="secondary" 
              size="md"
              icon={<Download size={16} />}
              aria-label="Export findings data"
            >
              Export
            </Button>
          </div>
        </div>
      </FadeIn>

      {/* Stats Cards */}
      <div className="findings-stats">
        <Card className="stat-card">
          <CardContent>
            <div className="stat-content">
              <div className="stat-value">{stats.total}</div>
              <div className="stat-label">Total Findings</div>
            </div>
          </CardContent>
        </Card>
        <Card className="stat-card stat-card-error">
          <CardContent>
            <div className="stat-content">
              <div className="stat-value">{stats.open}</div>
              <div className="stat-label">Open Issues</div>
            </div>
          </CardContent>
        </Card>
        <Card className="stat-card stat-card-success">
          <CardContent>
            <div className="stat-content">
              <div className="stat-value">{stats.resolved}</div>
              <div className="stat-label">Resolved</div>
            </div>
          </CardContent>
        </Card>
        <Card className="stat-card">
          <CardContent>
            <div className="stat-content">
              <div className="stat-breakdown">
                <span className="breakdown-item breakdown-high">{stats.high} High</span>
                <span className="breakdown-item breakdown-medium">{stats.medium} Med</span>
                <span className="breakdown-item breakdown-low">{stats.low} Low</span>
              </div>
              <div className="stat-label">Open by Severity</div>
            </div>
          </CardContent>
        </Card>
      </div>

      <AnimateOnScroll animation="slideUp" delay={200}>
        <Card className="findings-table-card">
          <CardHeader>
            <CardTitle>Security Findings ({filteredFindings.length})</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <TableSkeleton rows={5} columns={7} />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Finding</TableHead>
                    <TableHead>Severity</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Confidence</TableHead>
                    <TableHead>Location</TableHead>
                    <TableHead>First Seen</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredFindings.map((finding) => (
                    <TableRow key={finding.id} className="finding-row">
                      <TableCell>
                        <div className="finding-info">
                          <div className="finding-title">{finding.title}</div>
                          <div className="finding-description">{finding.description}</div>
                          <div className="finding-meta">
                            <Tag size={12} />
                            <span>{finding.cwe}</span>
                            <span className="category-tag">{finding.category}</span>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <SeverityBadge severity={finding.severity} />
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={finding.status} />
                      </TableCell>
                      <TableCell>
                        <ConfidenceMeter confidence={finding.confidence} />
                      </TableCell>
                      <TableCell>
                        <div className="location-info">
                          <div className="file-info">
                            <FileText size={12} />
                            <span>{finding.file}</span>
                          </div>
                          <div className="line-info">
                            <Code size={12} />
                            <span>Line {finding.line}</span>
                          </div>
                          <div className="repo-info">{finding.repository}</div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="date-info">
                          <Calendar size={12} />
                          <span>{formatDate(finding.firstSeen)}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="finding-actions">
                          {finding.status === 'open' && (
                            <>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleMarkResolved(finding.id)}
                                aria-label="Mark as resolved"
                              >
                                Resolve
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleMarkFalsePositive(finding.id)}
                                aria-label="Mark as false positive"
                              >
                                False Positive
                              </Button>
                            </>
                          )}
                          <Button
                            variant="ghost"
                            size="sm"
                            icon={<ExternalLink size={14} />}
                            aria-label="View finding details"
                          >
                            View
                          </Button>
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