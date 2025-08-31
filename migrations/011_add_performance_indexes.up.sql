-- Performance optimization indexes for production deployment
-- This migration adds critical indexes for frequent query patterns identified in performance analysis

-- Critical composite indexes for scan_jobs table
-- For frequent queries filtering by repository and status
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_repo_status_created 
    ON scan_jobs(repository_id, status, created_at DESC);

-- For user scan queries with pagination
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_user_created_desc 
    ON scan_jobs(user_id, created_at DESC) 
    WHERE user_id IS NOT NULL;

-- For organization-level scan queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_org_status_created 
    ON scan_jobs(repository_id, status, created_at DESC) 
    INCLUDE (user_id, branch, commit_sha);

-- For scan queue management (admin queries)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_status_priority_created 
    ON scan_jobs(status, priority DESC, created_at ASC) 
    WHERE status IN ('queued', 'running');

-- Critical composite indexes for findings table
-- For scan result queries with severity filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_findings_scan_severity_status 
    ON findings(scan_job_id, severity, status) 
    INCLUDE (title, file_path, line_number);

-- For repository-level finding aggregation
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_findings_repo_severity_created 
    ON findings(scan_job_id, severity, created_at DESC) 
    INCLUDE (status, category, tool);

-- For finding search and filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_findings_file_tool_severity 
    ON findings(file_path, tool, severity) 
    WHERE status = 'open';

-- For consensus and confidence queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_findings_consensus_confidence 
    ON findings(consensus_score DESC, confidence DESC) 
    WHERE consensus_score IS NOT NULL AND status = 'open';

-- Critical composite indexes for repositories table
-- For organization repository listing with activity
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_repositories_org_active_scan 
    ON repositories(organization_id, last_scan_at DESC NULLS LAST) 
    INCLUDE (name, provider, default_branch);

-- For repository search and filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_repositories_provider_name 
    ON repositories(provider, name) 
    INCLUDE (organization_id, url, last_scan_at);

-- For language-based queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_repositories_languages_gin 
    ON repositories USING GIN(languages) 
    WHERE languages IS NOT NULL AND languages != '[]'::jsonb;

-- Critical indexes for scan_results table
-- For agent performance analysis
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_results_agent_status_duration 
    ON scan_results(agent_name, status, duration_ms DESC) 
    INCLUDE (findings_count, created_at);

-- For scan job result aggregation
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_results_scan_agent_status 
    ON scan_results(scan_job_id, agent_name, status) 
    INCLUDE (findings_count, duration_ms, error_message);

-- Performance indexes for organization_members table
-- For user organization lookup
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_org_members_user_role_active 
    ON organization_members(user_id, role) 
    INCLUDE (organization_id, created_at);

-- For organization member management
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_org_members_org_role_created 
    ON organization_members(organization_id, role, created_at DESC) 
    INCLUDE (user_id);

-- Performance indexes for user_feedback table
-- For ML training data queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_feedback_action_created 
    ON user_feedback(action, created_at DESC) 
    INCLUDE (finding_id, user_id);

-- For finding feedback aggregation
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_feedback_finding_action 
    ON user_feedback(finding_id, action) 
    INCLUDE (user_id, created_at, comment);

-- Partial indexes for common filtered queries
-- Active repositories only
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_repositories_active_last_scan 
    ON repositories(last_scan_at DESC NULLS LAST) 
    WHERE settings->>'is_active' = 'true' OR settings->>'is_active' IS NULL;

-- Recent scan jobs (last 30 days)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_recent_status 
    ON scan_jobs(status, created_at DESC) 
    WHERE created_at >= NOW() - INTERVAL '30 days';

-- High severity findings only
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_findings_high_severity_open 
    ON findings(scan_job_id, created_at DESC) 
    WHERE severity = 'high' AND status = 'open';

-- Failed scans for monitoring
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_failed_recent 
    ON scan_jobs(created_at DESC, error_message) 
    WHERE status = 'failed' AND created_at >= NOW() - INTERVAL '7 days';

-- Indexes for dashboard queries
-- Repository health metrics
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_repos_health_metrics 
    ON repositories(organization_id, last_scan_at DESC) 
    INCLUDE (name, provider, settings) 
    WHERE last_scan_at IS NOT NULL;

-- Scan statistics for trends
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_daily_stats 
    ON scan_jobs(DATE(created_at), status) 
    INCLUDE (repository_id, user_id);

-- Finding trends by day
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_findings_daily_severity 
    ON findings(DATE(created_at), severity, status) 
    INCLUDE (scan_job_id, category, tool);

-- Performance indexes for JSON columns
-- Repository settings queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_repositories_settings_gin 
    ON repositories USING GIN(settings) 
    WHERE settings IS NOT NULL AND settings != '{}'::jsonb;

-- Scan job metadata queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_metadata_gin 
    ON scan_jobs USING GIN(metadata) 
    WHERE metadata IS NOT NULL AND metadata != '{}'::jsonb;

-- Finding fix suggestions
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_findings_fix_suggestions_gin 
    ON findings USING GIN(fix_suggestion) 
    WHERE fix_suggestion IS NOT NULL AND fix_suggestion != '{}'::jsonb;

-- Add statistics for query planner optimization
ANALYZE users;
ANALYZE organizations;
ANALYZE organization_members;
ANALYZE repositories;
ANALYZE scan_jobs;
ANALYZE scan_results;
ANALYZE findings;
ANALYZE user_feedback;

-- Create materialized view for dashboard statistics (will be refreshed periodically)
CREATE MATERIALIZED VIEW IF NOT EXISTS dashboard_stats AS
SELECT 
    r.organization_id,
    COUNT(DISTINCT r.id) as total_repositories,
    COUNT(DISTINCT sj.id) as total_scans,
    COUNT(DISTINCT CASE WHEN sj.status = 'completed' THEN sj.id END) as completed_scans,
    COUNT(DISTINCT CASE WHEN sj.status = 'failed' THEN sj.id END) as failed_scans,
    COUNT(DISTINCT f.id) as total_findings,
    COUNT(DISTINCT CASE WHEN f.severity = 'high' AND f.status = 'open' THEN f.id END) as high_severity_open,
    COUNT(DISTINCT CASE WHEN f.severity = 'medium' AND f.status = 'open' THEN f.id END) as medium_severity_open,
    COUNT(DISTINCT CASE WHEN f.severity = 'low' AND f.status = 'open' THEN f.id END) as low_severity_open,
    MAX(sj.created_at) as last_scan_date,
    AVG(CASE WHEN sj.status = 'completed' AND sj.started_at IS NOT NULL AND sj.completed_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (sj.completed_at - sj.started_at)) END) as avg_scan_duration_seconds
FROM repositories r
LEFT JOIN scan_jobs sj ON r.id = sj.repository_id
LEFT JOIN findings f ON sj.id = f.scan_job_id
WHERE sj.created_at >= NOW() - INTERVAL '90 days' OR sj.created_at IS NULL
GROUP BY r.organization_id;

-- Index for the materialized view
CREATE UNIQUE INDEX IF NOT EXISTS idx_dashboard_stats_org_id 
    ON dashboard_stats(organization_id);

-- Create a function to refresh dashboard stats
CREATE OR REPLACE FUNCTION refresh_dashboard_stats()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY dashboard_stats;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON INDEX idx_scan_jobs_repo_status_created IS 'Optimizes repository scan listing with status filtering';
COMMENT ON INDEX idx_scan_jobs_user_created_desc IS 'Optimizes user scan history queries';
COMMENT ON INDEX idx_findings_scan_severity_status IS 'Optimizes scan result detail queries with severity filtering';
COMMENT ON INDEX idx_repositories_org_active_scan IS 'Optimizes organization repository listing with last scan info';
COMMENT ON MATERIALIZED VIEW dashboard_stats IS 'Pre-computed dashboard statistics for performance';
COMMENT ON FUNCTION refresh_dashboard_stats() IS 'Refreshes dashboard statistics materialized view';