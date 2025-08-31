-- Rollback database views and materialized views

-- Drop functions
DROP FUNCTION IF EXISTS refresh_all_materialized_views();
DROP FUNCTION IF EXISTS refresh_finding_trends();
DROP FUNCTION IF EXISTS refresh_daily_scan_statistics();

-- Drop views (in reverse order of dependencies)
DROP VIEW IF EXISTS user_activity_summary;
DROP VIEW IF EXISTS agent_performance_summary;
DROP MATERIALIZED VIEW IF EXISTS finding_trends;
DROP MATERIALIZED VIEW IF EXISTS daily_scan_statistics;
DROP VIEW IF EXISTS repository_health_summary;
DROP VIEW IF EXISTS scan_jobs_with_details;

-- Drop indexes created for views
DROP INDEX IF EXISTS idx_finding_trends_lookup;
DROP INDEX IF EXISTS idx_daily_scan_statistics_date_org;
DROP INDEX IF EXISTS idx_scan_jobs_with_details_lookup;