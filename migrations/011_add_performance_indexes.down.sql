-- Rollback performance optimization indexes

-- Drop materialized view and function
DROP FUNCTION IF EXISTS refresh_dashboard_stats();
DROP MATERIALIZED VIEW IF EXISTS dashboard_stats;

-- Drop performance indexes (in reverse order of creation)
DROP INDEX CONCURRENTLY IF EXISTS idx_dashboard_stats_org_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_findings_fix_suggestions_gin;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_jobs_metadata_gin;
DROP INDEX CONCURRENTLY IF EXISTS idx_repositories_settings_gin;
DROP INDEX CONCURRENTLY IF EXISTS idx_findings_daily_severity;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_jobs_daily_stats;
DROP INDEX CONCURRENTLY IF EXISTS idx_repos_health_metrics;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_jobs_failed_recent;
DROP INDEX CONCURRENTLY IF EXISTS idx_findings_high_severity_open;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_jobs_recent_status;
DROP INDEX CONCURRENTLY IF EXISTS idx_repositories_active_last_scan;
DROP INDEX CONCURRENTLY IF EXISTS idx_user_feedback_finding_action;
DROP INDEX CONCURRENTLY IF EXISTS idx_user_feedback_action_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_org_members_org_role_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_org_members_user_role_active;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_results_scan_agent_status;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_results_agent_status_duration;
DROP INDEX CONCURRENTLY IF EXISTS idx_repositories_languages_gin;
DROP INDEX CONCURRENTLY IF EXISTS idx_repositories_provider_name;
DROP INDEX CONCURRENTLY IF EXISTS idx_repositories_org_active_scan;
DROP INDEX CONCURRENTLY IF EXISTS idx_findings_consensus_confidence;
DROP INDEX CONCURRENTLY IF EXISTS idx_findings_file_tool_severity;
DROP INDEX CONCURRENTLY IF EXISTS idx_findings_repo_severity_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_findings_scan_severity_status;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_jobs_status_priority_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_jobs_org_status_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_jobs_user_created_desc;
DROP INDEX CONCURRENTLY IF EXISTS idx_scan_jobs_repo_status_created;