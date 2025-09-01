-- Rollback production setup migration

BEGIN;

-- Drop RLS policies
DROP POLICY IF EXISTS users_own_data ON users;
DROP POLICY IF EXISTS scan_jobs_own_data ON scan_jobs;
DROP POLICY IF EXISTS findings_own_data ON findings;

-- Disable RLS
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE scan_jobs DISABLE ROW LEVEL SECURITY;
ALTER TABLE findings DISABLE ROW LEVEL SECURITY;
ALTER TABLE repositories DISABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences DISABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys DISABLE ROW LEVEL SECURITY;
ALTER TABLE scan_templates DISABLE ROW LEVEL SECURITY;
ALTER TABLE webhook_endpoints DISABLE ROW LEVEL SECURITY;

-- Drop materialized view
DROP MATERIALIZED VIEW IF EXISTS dashboard_stats;

-- Drop functions
DROP FUNCTION IF EXISTS refresh_dashboard_stats();
DROP FUNCTION IF EXISTS log_audit_event(UUID, VARCHAR(100), VARCHAR(50), UUID, JSONB, INET, TEXT);
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS cleanup_old_audit_logs();
DROP FUNCTION IF EXISTS create_monthly_partition(text, date);

-- Drop partitioned tables
DROP TABLE IF EXISTS scan_results_y2024m01;
DROP TABLE IF EXISTS scan_results_y2024m02;
DROP TABLE IF EXISTS scan_results_partitioned;

-- Drop tables
DROP TABLE IF EXISTS performance_metrics;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhook_endpoints;
DROP TABLE IF EXISTS scan_templates;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS system_config;
DROP TABLE IF EXISTS audit_logs;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_email_active;
DROP INDEX IF EXISTS idx_scan_jobs_user_status_created;
DROP INDEX IF EXISTS idx_findings_scan_severity_created;
DROP INDEX IF EXISTS idx_repositories_org_active_updated;

-- Revoke permissions
REVOKE ALL ON SCHEMA public FROM agentscan_readonly;
DROP ROLE IF EXISTS agentscan_readonly;

COMMIT;