-- Rollback initial schema

-- Drop triggers
DROP TRIGGER IF EXISTS update_findings_updated_at ON findings;
DROP TRIGGER IF EXISTS update_scan_jobs_updated_at ON scan_jobs;
DROP TRIGGER IF EXISTS update_repositories_updated_at ON repositories;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS user_feedback;
DROP TABLE IF EXISTS findings;
DROP TABLE IF EXISTS scan_results;
DROP TABLE IF EXISTS scan_jobs;
DROP TABLE IF EXISTS repositories;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;

-- Drop enum types
DROP TYPE IF EXISTS scan_type;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS finding_status;
DROP TYPE IF EXISTS finding_severity;
DROP TYPE IF EXISTS scan_status;

-- Drop extension (only if no other objects depend on it)
-- DROP EXTENSION IF EXISTS "uuid-ossp";