import React from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/Card';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/Table';
import { Button } from '../components/ui/Button';
import { 
  Shield, 
  AlertTriangle, 
  CheckCircle, 
  Clock,

  Search,
  Filter,
  Download
} from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import './Dashboard.css';

// Mock data for demonstration
const scanStats = {
  totalScans: 1247,
  highSeverity: 23,
  mediumSeverity: 156,
  lowSeverity: 89,
};

const recentScans = [
  {
    id: '1',
    repository: 'frontend/web-app',
    status: 'completed',
    findings: { high: 2, medium: 8, low: 3 },
    duration: '2m 34s',
    timestamp: '2 minutes ago',
  },
  {
    id: '2',
    repository: 'backend/api-service',
    status: 'running',
    findings: null,
    duration: null,
    timestamp: 'Started 5 minutes ago',
  },
  {
    id: '3',
    repository: 'mobile/ios-app',
    status: 'completed',
    findings: { high: 0, medium: 4, low: 12 },
    duration: '1m 45s',
    timestamp: '1 hour ago',
  },
  {
    id: '4',
    repository: 'infrastructure/terraform',
    status: 'failed',
    findings: null,
    duration: null,
    timestamp: '2 hours ago',
  },
  {
    id: '5',
    repository: 'docs/documentation',
    status: 'completed',
    findings: { high: 0, medium: 0, low: 1 },
    duration: '45s',
    timestamp: '3 hours ago',
  },
];

const trendData = [
  { date: '2024-01-01', high: 15, medium: 45, low: 23 },
  { date: '2024-01-02', high: 12, medium: 38, low: 28 },
  { date: '2024-01-03', high: 18, medium: 52, low: 31 },
  { date: '2024-01-04', high: 8, medium: 29, low: 19 },
  { date: '2024-01-05', high: 23, medium: 67, low: 42 },
  { date: '2024-01-06', high: 16, medium: 43, low: 35 },
  { date: '2024-01-07', high: 11, medium: 31, low: 24 },
];

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'completed':
        return { icon: CheckCircle, color: 'var(--color-success)', bg: '#dcfce7' };
      case 'running':
        return { icon: Clock, color: 'var(--color-warning)', bg: '#fef3c7' };
      case 'failed':
        return { icon: AlertTriangle, color: 'var(--color-error)', bg: '#fee2e2' };
      default:
        return { icon: Clock, color: 'var(--color-gray-500)', bg: 'var(--color-gray-100)' };
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
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
};

const FindingsSummary: React.FC<{ findings: { high: number; medium: number; low: number } | null }> = ({ findings }) => {
  if (!findings) return <span className="text-gray-400">-</span>;

  return (
    <div className="findings-summary">
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
      {findings.high === 0 && findings.medium === 0 && findings.low === 0 && (
        <span className="finding-count finding-clean">Clean</span>
      )}
    </div>
  );
};

export const Dashboard: React.FC = () => {
  return (
    <div className="dashboard">
      <div className="dashboard-header">
        <div>
          <h1 className="dashboard-title">Security Dashboard</h1>
          <p className="dashboard-subtitle">
            Monitor your security posture across all repositories
          </p>
        </div>
        <div className="dashboard-actions">
          <Button variant="secondary" size="md">
            <Filter size={16} />
            Filter
          </Button>
          <Button variant="secondary" size="md">
            <Download size={16} />
            Export
          </Button>
          <Button variant="primary" size="md">
            <Search size={16} />
            New Scan
          </Button>
        </div>
      </div>

      {/* Statistics Cards */}
      <div className="stats-grid">
        <Card hover>
          <CardContent>
            <div className="stat-card">
              <div className="stat-icon stat-icon-primary">
                <Shield size={24} />
              </div>
              <div className="stat-content">
                <div className="stat-value">{scanStats.totalScans.toLocaleString()}</div>
                <div className="stat-label">Total Scans</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover>
          <CardContent>
            <div className="stat-card">
              <div className="stat-icon stat-icon-error">
                <AlertTriangle size={24} />
              </div>
              <div className="stat-content">
                <div className="stat-value">{scanStats.highSeverity}</div>
                <div className="stat-label">High Severity</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover>
          <CardContent>
            <div className="stat-card">
              <div className="stat-icon stat-icon-warning">
                <AlertTriangle size={24} />
              </div>
              <div className="stat-content">
                <div className="stat-value">{scanStats.mediumSeverity}</div>
                <div className="stat-label">Medium Severity</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card hover>
          <CardContent>
            <div className="stat-card">
              <div className="stat-icon stat-icon-info">
                <AlertTriangle size={24} />
              </div>
              <div className="stat-content">
                <div className="stat-value">{scanStats.lowSeverity}</div>
                <div className="stat-label">Low Severity</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Scans Table */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Scans</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Repository</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Findings</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Time</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {recentScans.map((scan) => (
                <TableRow key={scan.id}>
                  <TableCell>
                    <div className="repository-name">{scan.repository}</div>
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={scan.status} />
                  </TableCell>
                  <TableCell>
                    <FindingsSummary findings={scan.findings} />
                  </TableCell>
                  <TableCell>
                    <span className="duration">
                      {scan.duration || '-'}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="timestamp">{scan.timestamp}</span>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Findings Trend Chart */}
      <Card>
        <CardHeader>
          <div className="chart-header">
            <CardTitle>Findings Trend</CardTitle>
            <div className="chart-legend">
              <div className="legend-item">
                <div className="legend-color" style={{ backgroundColor: 'var(--color-error)' }}></div>
                <span>High</span>
              </div>
              <div className="legend-item">
                <div className="legend-color" style={{ backgroundColor: 'var(--color-warning)' }}></div>
                <span>Medium</span>
              </div>
              <div className="legend-item">
                <div className="legend-color" style={{ backgroundColor: 'var(--color-info)' }}></div>
                <span>Low</span>
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="chart-container">
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={trendData}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-gray-200)" />
                <XAxis 
                  dataKey="date" 
                  stroke="var(--color-gray-500)"
                  fontSize={12}
                  tickFormatter={(value) => new Date(value).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
                />
                <YAxis 
                  stroke="var(--color-gray-500)"
                  fontSize={12}
                />
                <Tooltip 
                  contentStyle={{
                    backgroundColor: 'var(--color-bg-primary)',
                    border: '1px solid var(--color-gray-200)',
                    borderRadius: 'var(--radius-md)',
                    fontSize: '14px',
                  }}
                />
                <Line 
                  type="monotone" 
                  dataKey="high" 
                  stroke="var(--color-error)" 
                  strokeWidth={2}
                  dot={{ fill: 'var(--color-error)', strokeWidth: 2, r: 4 }}
                />
                <Line 
                  type="monotone" 
                  dataKey="medium" 
                  stroke="var(--color-warning)" 
                  strokeWidth={2}
                  dot={{ fill: 'var(--color-warning)', strokeWidth: 2, r: 4 }}
                />
                <Line 
                  type="monotone" 
                  dataKey="low" 
                  stroke="var(--color-info)" 
                  strokeWidth={2}
                  dot={{ fill: 'var(--color-info)', strokeWidth: 2, r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};