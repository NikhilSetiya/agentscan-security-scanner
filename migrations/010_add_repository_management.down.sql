-- Rollback repository management enhancements

-- Drop triggers and functions
DROP TRIGGER IF EXISTS trigger_update_repository_last_scan ON scan_jobs;
DROP FUNCTION IF EXISTS update_repository_last_scan();

-- Drop views
DROP VIEW IF EXISTS scan_job_stats;
DROP VIEW IF EXISTS repository_stats;
DROP VIEW IF EXISTS dashboard_stats;

-- Drop indexes
DROP INDEX IF EXISTS idx_repositories_org_active;
DROP INDEX IF EXISTS idx_repositories_is_active;
DROP INDEX IF EXISTS idx_repositories_language;
DROP INDEX IF EXISTS idx_repositories_name;

-- Remove columns from repositories table
ALTER TABLE repositories DROP COLUMN IF EXISTS scan_config;
ALTER TABLE repositories DROP COLUMN IF EXISTS is_active;
ALTER TABLE repositories DROP COLUMN IF EXISTS description;
ALTER TABLE repositories DROP COLUMN IF EXISTS language;