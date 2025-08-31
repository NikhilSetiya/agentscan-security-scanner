-- Create database views for complex queries and analytics
-- This migration creates views and materialized views to optimize frequently used complex queries

-- View: scan_jobs_with_details
-- Provides scan jobs with repository and user information in a single query
CREATE OR REPLACE VIEW scan_jobs_with_details AS
SELECT 
    -- Scan job fields
    sj.id,
    sj.repository_id,
    sj.user_id,
    sj.branch,
    sj.commit_sha,
    sj.scan_type,
    sj.priority,
    sj.status,
    sj.agents_requested,
    sj.agents_completed,
    sj.started_at,
    sj.completed_at,
    sj.error_message,
    sj.metadata,
    sj.created_at,
    sj.updated_at,
    
    -- Repository information
    r.name as repository_name,
    r.url as repository_url,
    r.provider as repository_provider,
    r.provider_id as repository_provider_id,
    r.default_branch as repository_default_branch,
    r.organization_id,
    
    -- User information (nullable)
    u.email as user_email,
    u.name as user_name,
    u.avatar_url as user_avatar_url,
    
    -- Computed fields
    CASE 
        WHEN sj.started_at IS NOT NULL AND sj.completed_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (sj.completed_at - sj.started_at))
        ELSE NULL 
    END as duration_seconds,
    
    CASE 
        WHEN sj.status = 'completed' THEN 100
        WHEN sj.status = 'running' AND sj.started_at IS NOT NULL THEN 50
        WHEN sj.status = 'failed' OR sj.status = 'cancelled' THEN 100
        ELSE 0
    END as progress_percentage,
    
    -- Findings statistics (computed via subquery)
    COALESCE(f_stats.total_findings, 0) as findings_count,
    COALESCE(f_stats.high_findings, 0) as high_findings_count,
    COALESCE(f_stats.medium_findings, 0) as medium_findings_count,
    COALESCE(f_stats.low_findings, 0) as low_findings_count,
    COALESCE(f_stats.info_findings, 0) as info_findings_count,
    COALESCE(f_stats.open_findings, 0) as open_findings_count,
    COALESCE(f_stats.fixed_findings, 0) as fixed_findings_count

FROM scan_jobs sj
INNER JOIN repositories r ON sj.repository_id = r.id
LEFT JOIN users u ON sj.user_id = u.id
LEFT JOIN (
    SELECT 
        scan_job_id,
        COUNT(*) as total_findings,
        COUNT(CASE WHEN severity = 'high' THEN 1 END) as high_findings,
        COUNT(CASE WHEN severity = 'medium' THEN 1 END) as medium_findings,
        COUNT(CASE WHEN severity = 'low' THEN 1 END) as low_findings,
        COUNT(CASE WHEN severity = 'info' THEN 1 END) as info_findings,
        COUNT(CASE WHEN status = 'open' THEN 1 END) as open_findings,
        COUNT(CASE WHEN status = 'fixed' THEN 1 END) as fixed_findings
    FROM findings
    GROUP BY scan_job_id
) f_stats ON sj.id = f_stats.scan_job_id;

-- Index for the view
CREATE INDEX IF NOT EXISTS idx_scan_jobs_with_details_lookup 
    ON scan_jobs(repository_id, status, created_at DESC);

-- View: repository_health_summary
-- Provides repository health metrics and statistics
CREATE OR REPLACE VIEW repository_health_summary AS
SELECT 
    r.id,
    r.organization_id,
    r.name,
    r.url,
    r.provider,
    r.provider_id,
    r.default_branch,
    r.languages,
    r.last_scan_at,
    r.created_at,
    r.updated_at,
    
    -- Scan statistics (last 30 days)
    COALESCE(scan_stats.total_scans, 0) as total_scans_30d,
    COALESCE(scan_stats.completed_scans, 0) as completed_scans_30d,
    COALESCE(scan_stats.failed_scans, 0) as failed_scans_30d,
    COALESCE(scan_stats.avg_duration_seconds, 0) as avg_scan_duration_seconds,
    
    -- Finding statistics (current open findings)
    COALESCE(finding_stats.total_findings, 0) as total_open_findings,
    COALESCE(finding_stats.high_findings, 0) as high_open_findings,
    COALESCE(finding_stats.medium_findings, 0) as medium_open_findings,
    COALESCE(finding_stats.low_findings, 0) as low_open_findings,
    COALESCE(finding_stats.info_findings, 0) as info_open_findings,
    
    -- Health score calculation
    CASE 
        WHEN COALESCE(finding_stats.high_findings, 0) = 0 AND COALESCE(finding_stats.total_findings, 0) <= 5 THEN 95.0
        WHEN COALESCE(finding_stats.high_findings, 0) <= 2 AND COALESCE(finding_stats.total_findings, 0) <= 15 THEN 85.0
        WHEN COALESCE(finding_stats.high_findings, 0) <= 5 AND COALESCE(finding_stats.total_findings, 0) <= 30 THEN 75.0
        WHEN COALESCE(finding_stats.high_findings, 0) <= 10 AND COALESCE(finding_stats.total_findings, 0) <= 50 THEN 65.0
        WHEN COALESCE(finding_stats.high_findings, 0) <= 20 AND COALESCE(finding_stats.total_findings, 0) <= 100 THEN 55.0
        ELSE 45.0
    END as health_score,
    
    -- Last scan information
    last_scan.id as last_scan_id,
    last_scan.status as last_scan_status,
    last_scan.created_at as last_scan_created_at,
    last_scan.duration_seconds as last_scan_duration_seconds

FROM repositories r
LEFT JOIN (
    SELECT 
        repository_id,
        COUNT(*) as total_scans,
        COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_scans,
        COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_scans,
        AVG(CASE 
            WHEN status = 'completed' AND started_at IS NOT NULL AND completed_at IS NOT NULL 
            THEN EXTRACT(EPOCH FROM (completed_at - started_at))
            ELSE NULL 
        END) as avg_duration_seconds
    FROM scan_jobs
    WHERE created_at >= NOW() - INTERVAL '30 days'
    GROUP BY repository_id
) scan_stats ON r.id = scan_stats.repository_id
LEFT JOIN (
    SELECT 
        sj.repository_id,
        COUNT(f.id) as total_findings,
        COUNT(CASE WHEN f.severity = 'high' THEN 1 END) as high_findings,
        COUNT(CASE WHEN f.severity = 'medium' THEN 1 END) as medium_findings,
        COUNT(CASE WHEN f.severity = 'low' THEN 1 END) as low_findings,
        COUNT(CASE WHEN f.severity = 'info' THEN 1 END) as info_findings
    FROM scan_jobs sj
    INNER JOIN findings f ON sj.id = f.scan_job_id
    WHERE f.status = 'open'
    GROUP BY sj.repository_id
) finding_stats ON r.id = finding_stats.repository_id
LEFT JOIN LATERAL (
    SELECT 
        id, 
        status, 
        created_at,
        CASE 
            WHEN started_at IS NOT NULL AND completed_at IS NOT NULL 
            THEN EXTRACT(EPOCH FROM (completed_at - started_at))
            ELSE NULL 
        END as duration_seconds
    FROM scan_jobs
    WHERE repository_id = r.id
    ORDER BY created_at DESC
    LIMIT 1
) last_scan ON true;

-- Materialized view: daily_scan_statistics
-- Pre-computed daily statistics for dashboard trends
CREATE MATERIALIZED VIEW IF NOT EXISTS daily_scan_statistics AS
SELECT 
    DATE(sj.created_at) as date,
    r.organization_id,
    COUNT(*) as total_scans,
    COUNT(CASE WHEN sj.status = 'completed' THEN 1 END) as completed_scans,
    COUNT(CASE WHEN sj.status = 'failed' THEN 1 END) as failed_scans,
    COUNT(CASE WHEN sj.status = 'cancelled' THEN 1 END) as cancelled_scans,
    COUNT(DISTINCT sj.repository_id) as repositories_scanned,
    COUNT(DISTINCT sj.user_id) as users_active,
    AVG(CASE 
        WHEN sj.status = 'completed' AND sj.started_at IS NOT NULL AND sj.completed_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (sj.completed_at - sj.started_at))
        ELSE NULL 
    END) as avg_duration_seconds,
    
    -- Finding statistics for the day
    COALESCE(SUM(f_stats.total_findings), 0) as total_findings_created,
    COALESCE(SUM(f_stats.high_findings), 0) as high_findings_created,
    COALESCE(SUM(f_stats.medium_findings), 0) as medium_findings_created,
    COALESCE(SUM(f_stats.low_findings), 0) as low_findings_created,
    COALESCE(SUM(f_stats.info_findings), 0) as info_findings_created

FROM scan_jobs sj
INNER JOIN repositories r ON sj.repository_id = r.id
LEFT JOIN (
    SELECT 
        scan_job_id,
        COUNT(*) as total_findings,
        COUNT(CASE WHEN severity = 'high' THEN 1 END) as high_findings,
        COUNT(CASE WHEN severity = 'medium' THEN 1 END) as medium_findings,
        COUNT(CASE WHEN severity = 'low' THEN 1 END) as low_findings,
        COUNT(CASE WHEN severity = 'info' THEN 1 END) as info_findings
    FROM findings
    GROUP BY scan_job_id
) f_stats ON sj.id = f_stats.scan_job_id
WHERE sj.created_at >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE(sj.created_at), r.organization_id
ORDER BY date DESC, r.organization_id;

-- Unique index for the materialized view
CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_scan_statistics_date_org 
    ON daily_scan_statistics(date, organization_id);

-- Materialized view: finding_trends
-- Pre-computed finding trends for analytics
CREATE MATERIALIZED VIEW IF NOT EXISTS finding_trends AS
SELECT 
    DATE(f.created_at) as date,
    r.organization_id,
    f.severity,
    f.category,
    f.tool,
    COUNT(*) as findings_count,
    COUNT(CASE WHEN f.status = 'open' THEN 1 END) as open_count,
    COUNT(CASE WHEN f.status = 'fixed' THEN 1 END) as fixed_count,
    COUNT(CASE WHEN f.status = 'ignored' THEN 1 END) as ignored_count,
    COUNT(CASE WHEN f.status = 'false_positive' THEN 1 END) as false_positive_count,
    AVG(f.confidence) as avg_confidence,
    AVG(f.consensus_score) as avg_consensus_score

FROM findings f
INNER JOIN scan_jobs sj ON f.scan_job_id = sj.id
INNER JOIN repositories r ON sj.repository_id = r.id
WHERE f.created_at >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE(f.created_at), r.organization_id, f.severity, f.category, f.tool
ORDER BY date DESC, r.organization_id, f.severity, f.category, f.tool;

-- Index for finding trends
CREATE INDEX IF NOT EXISTS idx_finding_trends_lookup 
    ON finding_trends(organization_id, date DESC, severity, category);

-- View: agent_performance_summary
-- Provides agent performance metrics
CREATE OR REPLACE VIEW agent_performance_summary AS
SELECT 
    sr.agent_name,
    r.organization_id,
    COUNT(*) as total_runs,
    COUNT(CASE WHEN sr.status = 'completed' THEN 1 END) as successful_runs,
    COUNT(CASE WHEN sr.status = 'failed' THEN 1 END) as failed_runs,
    ROUND(
        COUNT(CASE WHEN sr.status = 'completed' THEN 1 END)::DECIMAL / 
        NULLIF(COUNT(*), 0) * 100, 2
    ) as success_rate_percent,
    
    AVG(sr.duration_ms) as avg_duration_ms,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY sr.duration_ms) as median_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY sr.duration_ms) as p95_duration_ms,
    
    AVG(sr.findings_count) as avg_findings_per_run,
    SUM(sr.findings_count) as total_findings_found,
    
    MIN(sr.created_at) as first_run_at,
    MAX(sr.created_at) as last_run_at

FROM scan_results sr
INNER JOIN scan_jobs sj ON sr.scan_job_id = sj.id
INNER JOIN repositories r ON sj.repository_id = r.id
WHERE sr.created_at >= NOW() - INTERVAL '30 days'
GROUP BY sr.agent_name, r.organization_id
ORDER BY r.organization_id, success_rate_percent DESC, avg_duration_ms ASC;

-- View: user_activity_summary
-- Provides user activity metrics
CREATE OR REPLACE VIEW user_activity_summary AS
SELECT 
    u.id as user_id,
    u.email,
    u.name,
    om.organization_id,
    om.role as organization_role,
    
    -- Scan activity (last 30 days)
    COALESCE(scan_activity.total_scans, 0) as scans_initiated_30d,
    COALESCE(scan_activity.completed_scans, 0) as scans_completed_30d,
    COALESCE(scan_activity.repositories_scanned, 0) as repositories_scanned_30d,
    
    -- Feedback activity (last 30 days)
    COALESCE(feedback_activity.total_feedback, 0) as feedback_given_30d,
    COALESCE(feedback_activity.findings_fixed, 0) as findings_marked_fixed_30d,
    COALESCE(feedback_activity.findings_ignored, 0) as findings_marked_ignored_30d,
    
    -- Last activity
    GREATEST(
        COALESCE(scan_activity.last_scan_at, '1970-01-01'::timestamp),
        COALESCE(feedback_activity.last_feedback_at, '1970-01-01'::timestamp)
    ) as last_activity_at

FROM users u
INNER JOIN organization_members om ON u.id = om.user_id
LEFT JOIN (
    SELECT 
        sj.user_id,
        COUNT(*) as total_scans,
        COUNT(CASE WHEN sj.status = 'completed' THEN 1 END) as completed_scans,
        COUNT(DISTINCT sj.repository_id) as repositories_scanned,
        MAX(sj.created_at) as last_scan_at
    FROM scan_jobs sj
    WHERE sj.created_at >= NOW() - INTERVAL '30 days'
      AND sj.user_id IS NOT NULL
    GROUP BY sj.user_id
) scan_activity ON u.id = scan_activity.user_id
LEFT JOIN (
    SELECT 
        uf.user_id,
        COUNT(*) as total_feedback,
        COUNT(CASE WHEN uf.action = 'fixed' THEN 1 END) as findings_fixed,
        COUNT(CASE WHEN uf.action = 'ignored' THEN 1 END) as findings_ignored,
        MAX(uf.created_at) as last_feedback_at
    FROM user_feedback uf
    WHERE uf.created_at >= NOW() - INTERVAL '30 days'
    GROUP BY uf.user_id
) feedback_activity ON u.id = feedback_activity.user_id
ORDER BY om.organization_id, last_activity_at DESC;

-- Functions to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_daily_scan_statistics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY daily_scan_statistics;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION refresh_finding_trends()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY finding_trends;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION refresh_all_materialized_views()
RETURNS void AS $$
BEGIN
    PERFORM refresh_dashboard_stats();
    PERFORM refresh_daily_scan_statistics();
    PERFORM refresh_finding_trends();
END;
$$ LANGUAGE plpgsql;

-- Create a scheduled job to refresh materialized views (requires pg_cron extension)
-- This would typically be set up separately in production
-- SELECT cron.schedule('refresh-materialized-views', '0 */6 * * *', 'SELECT refresh_all_materialized_views();');

-- Add comments for documentation
COMMENT ON VIEW scan_jobs_with_details IS 'Denormalized view of scan jobs with repository and user details for efficient querying';
COMMENT ON VIEW repository_health_summary IS 'Repository health metrics including scan statistics and finding counts';
COMMENT ON MATERIALIZED VIEW daily_scan_statistics IS 'Pre-computed daily scan statistics for dashboard trends';
COMMENT ON MATERIALIZED VIEW finding_trends IS 'Pre-computed finding trends by date, severity, category, and tool';
COMMENT ON VIEW agent_performance_summary IS 'Agent performance metrics including success rates and duration statistics';
COMMENT ON VIEW user_activity_summary IS 'User activity summary including scan and feedback activity';

COMMENT ON FUNCTION refresh_daily_scan_statistics() IS 'Refreshes the daily scan statistics materialized view';
COMMENT ON FUNCTION refresh_finding_trends() IS 'Refreshes the finding trends materialized view';
COMMENT ON FUNCTION refresh_all_materialized_views() IS 'Refreshes all materialized views used for analytics';