import React, { useState, useEffect } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/Card';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/Table';
import { Button } from '../components/ui/Button';
import { LoadingSkeleton, StatCardSkeleton, TableSkeleton } from '../components/ui/LoadingSkeleton';
import { NetworkError } from '../components/ui/ErrorState';
import { FadeIn, StaggeredList, AnimateOnScroll } from '../components/ui/Transitions';
import { useDashboardStats } from '../hooks/useApi';
import { 
  Shield, 
  AlertTriangle, 
  CheckCircle, 
  Clock,
  Search,
  Filter,
  Download,
  RefreshCw
} from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import './Dashboard.css';

// Utility functions
const calculateDuration = (startTime: string, endTime: string): string => {
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

const formatTimestamp = (timestamp: string): string => {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  
  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins} minute${diffMins !== 1 ? 's' : ''} ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
  if (diffDays < 7) return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;
  
  return date.toLocaleDateString();
};

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

const FindingsSummary: React.FC<{ scan: any }> = ({ scan }) => {
  if (scan.status === 'running') {
    return <span className="text-gray-400">Running...</span>;
  }
  
  if (scan.status === 'failed') {
    return <span className="text-gray-400">Failed</span>;
  }

  const findingsCount = scan.findings_count || 0;
  
  if (findingsCount === 0) {
    return <span className="finding-count finding-clean">Clean</span>;
  }

  // For now, show total findings count since we don't have severity breakdown in the scan object
  return (
    <div className="findings-summary">
      <span className="finding-count finding-total">
        {findingsCount} Finding{findingsCount !== 1 ? 's' : ''}
      </span>
    </div>
  );
};

export const Dashboard: React.FC = () => {
  const { data: dashboardData, loading: isLoading, error, execute: refetchData } = useDashboardStats();
  const [isRefreshing, setIsRefreshing] = useState(false);

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try {
      await refetchData();
    } finally {
      setIsRefreshing(false);
    }
  };

  const handleNewScan = () => {
    // For demo purposes, show a simple alert
    // In a real app, this would open a modal to select repository and scan options
    alert('New scan functionality would open a repository selection modal');
  };

  // Listen for global refresh event
  useEffect(() => {
    const handleGlobalRefresh = () => handleRefresh();
    document.addEventListener('refresh-data', handleGlobalRefresh);
    return () => document.removeEventListener('refresh-data', handleGlobalRefresh);
  }, []);

  // Extract data with fallbacks for when API data is not available
  const scanStats = dashboardData ? {
    totalScans: dashboardData.total_scans,
    highSeverity: dashboardData.findings_by_severity.high,
    mediumSeverity: dashboardData.findings_by_severity.medium,
    lowSeverity: dashboardData.findings_by_severity.low,
  } : {
    totalScans: 0,
    highSeverity: 0,
    mediumSeverity: 0,
    lowSeverity: 0,
  };

  const recentScans = dashboardData?.recent_scans || [];
  const trendData = dashboardData?.trend_data || [];

  if (error && !isLoading) {
    return (
      <div className="dashboard">
        <NetworkError 
          onRetry={handleRefresh}
        />
      </div>
    );
  }

  return (
    <div className="dashboard">
      <FadeIn>
        <div className="dashboard-header">
          <div>
            <h1 className="dashboard-title">Security Dashboard</h1>
            <p className="dashboard-subtitle">
              Monitor your security posture across all repositories
            </p>
          </div>
          <div className="dashboard-actions">
            <Button 
              variant="secondary" 
              size="md"
              icon={<RefreshCw size={16} />}
              loading={isRefreshing}
              loadingText="Refreshing..."
              onClick={handleRefresh}
              aria-label="Refresh dashboard data"
            >
              Refresh
            </Button>
            <Button 
              variant="secondary" 
              size="md"
              icon={<Filter size={16} />}
              aria-label="Filter results"
            >
              Filter
            </Button>
            <Button 
              variant="secondary" 
              size="md"
              icon={<Download size={16} />}
              aria-label="Export data"
            >
              Export
            </Button>
            <Button 
              variant="primary" 
              size="md"
              icon={<Search size={16} />}
              onClick={handleNewScan}
              aria-label="Start new security scan"
            >
              New Scan
            </Button>
          </div>
        </div>
      </FadeIn>

      {/* Statistics Cards */}
      <div className="stats-grid">
        {isLoading ? (
          <>
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
          </>
        ) : (
          <StaggeredList staggerDelay={100}>
            <Card hover className="hover-lift">
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

            <Card hover className="hover-lift">
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

            <Card hover className="hover-lift">
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

            <Card hover className="hover-lift">
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
          </StaggeredList>
        )}
      </div>

      {/* Recent Scans Table */}
      <AnimateOnScroll animation="slideUp" delay={200}>
        <Card className="hover-lift">
          <CardHeader>
            <CardTitle>Recent Scans</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <TableSkeleton rows={5} columns={5} />
            ) : (
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
                  {recentScans.map((scan, index) => (
                    <TableRow 
                      key={scan.id}
                      className={`animate-fade-in stagger-${Math.min(index + 1, 5)}`}
                    >
                      <TableCell>
                        <div className="repository-name">
                          {scan.repository?.name || `Repository ${scan.repository_id}`}
                        </div>
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={scan.status} />
                      </TableCell>
                      <TableCell>
                        <FindingsSummary scan={scan} />
                      </TableCell>
                      <TableCell>
                        <span className="duration">
                          {scan.duration || (scan.completed_at && scan.started_at ? 
                            calculateDuration(scan.started_at, scan.completed_at) : '-')}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="timestamp">
                          {formatTimestamp(scan.started_at)}
                        </span>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </AnimateOnScroll>

      {/* Findings Trend Chart */}
      <AnimateOnScroll animation="slideUp" delay={400}>
        <Card className="hover-lift">
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
            {isLoading ? (
              <div className="chart-skeleton">
                <LoadingSkeleton height={300} />
              </div>
            ) : (
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
                        boxShadow: 'var(--shadow-lg)',
                      }}
                    />
                    <Line 
                      type="monotone" 
                      dataKey="high" 
                      stroke="var(--color-error)" 
                      strokeWidth={2}
                      dot={{ fill: 'var(--color-error)', strokeWidth: 2, r: 4 }}
                      activeDot={{ r: 6, stroke: 'var(--color-error)', strokeWidth: 2 }}
                    />
                    <Line 
                      type="monotone" 
                      dataKey="medium" 
                      stroke="var(--color-warning)" 
                      strokeWidth={2}
                      dot={{ fill: 'var(--color-warning)', strokeWidth: 2, r: 4 }}
                      activeDot={{ r: 6, stroke: 'var(--color-warning)', strokeWidth: 2 }}
                    />
                    <Line 
                      type="monotone" 
                      dataKey="low" 
                      stroke="var(--color-info)" 
                      strokeWidth={2}
                      dot={{ fill: 'var(--color-info)', strokeWidth: 2, r: 4 }}
                      activeDot={{ r: 6, stroke: 'var(--color-info)', strokeWidth: 2 }}
                    />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            )}
          </CardContent>
        </Card>
      </AnimateOnScroll>
    </div>
  );
};