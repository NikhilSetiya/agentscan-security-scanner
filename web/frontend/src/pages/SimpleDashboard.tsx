import React, { useState } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { ErrorFallback } from '../components/ui/ErrorFallback';
import { useDashboardStats } from '../hooks/useApi';
import { Shield, AlertTriangle, CheckCircle, RefreshCw, Search } from 'lucide-react';

export const SimpleDashboard: React.FC = () => {
  const { data, loading, error, execute: refetch } = useDashboardStats();
  const [refreshing, setRefreshing] = useState(false);

  const handleRefresh = async () => {
    setRefreshing(true);
    await refetch();
    setRefreshing(false);
  };

  // Show error state
  if (error && !loading) {
    return (
      <div className="p-6">
        <ErrorFallback 
          message="Failed to load dashboard data" 
          onRetry={handleRefresh}
        />
      </div>
    );
  }

  // Use fallback data if no data available
  const stats = data || {
    total_scans: 0,
    total_repositories: 0,
    findings_by_severity: { critical: 0, high: 0, medium: 0, low: 0, info: 0 },
    recent_scans: []
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Security Dashboard</h1>
          <p className="text-gray-600">Monitor your security posture across all repositories</p>
        </div>
        <div className="flex gap-2">
          <Button 
            variant="secondary" 
            onClick={handleRefresh}
            loading={refreshing || loading}
            icon={<RefreshCw size={16} />}
          >
            Refresh
          </Button>
          <Button 
            variant="primary"
            icon={<Search size={16} />}
          >
            New Scan
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Total Scans</p>
                <p className="text-2xl font-bold text-gray-900">{stats.total_scans}</p>
              </div>
              <Shield className="w-8 h-8 text-blue-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Repositories</p>
                <p className="text-2xl font-bold text-gray-900">{stats.total_repositories}</p>
              </div>
              <CheckCircle className="w-8 h-8 text-green-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Critical Issues</p>
                <p className="text-2xl font-bold text-red-600">{stats.findings_by_severity.critical}</p>
              </div>
              <AlertTriangle className="w-8 h-8 text-red-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">High Issues</p>
                <p className="text-2xl font-bold text-orange-600">{stats.findings_by_severity.high}</p>
              </div>
              <AlertTriangle className="w-8 h-8 text-orange-500" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Scans */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Scans</CardTitle>
        </CardHeader>
        <CardContent>
          {stats.recent_scans && stats.recent_scans.length > 0 ? (
            <div className="space-y-4">
              {stats.recent_scans.map((scan: any, index: number) => (
                <div key={scan.id || index} className="flex items-center justify-between p-4 border rounded-lg">
                  <div>
                    <h4 className="font-medium">{scan.repository?.name || 'Unknown Repository'}</h4>
                    <p className="text-sm text-gray-600">
                      Status: <span className={`font-medium ${scan.status === 'completed' ? 'text-green-600' : 'text-blue-600'}`}>
                        {scan.status}
                      </span>
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-medium">{scan.findings_count} findings</p>
                    <p className="text-xs text-gray-500">
                      {scan.started_at ? new Date(scan.started_at).toLocaleDateString() : 'N/A'}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              <Shield className="w-12 h-12 mx-auto mb-4 text-gray-300" />
              <p>No scans yet. Start your first security scan!</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};