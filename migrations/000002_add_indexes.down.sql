-- Remove performance indexes

-- Drop indexes for organizations table
DROP INDEX IF EXISTS idx_organizations_slug;

-- Drop indexes for users table
DROP INDEX IF EXISTS idx_users_gitlab_id;
DROP INDEX IF EXISTS idx_users_github_id;
DROP INDEX IF EXISTS idx_users_email;

-- Drop indexes for user_feedback table
DROP INDEX IF EXISTS idx_user_feedback_action;
DROP INDEX IF EXISTS idx_user_feedback_user_id;
DROP INDEX IF EXISTS idx_user_feedback_finding_id;

-- Drop indexes for organization_members table
DROP INDEX IF EXISTS idx_organization_members_role;
DROP INDEX IF EXISTS idx_organization_members_user_id;
DROP INDEX IF EXISTS idx_organization_members_organization_id;

-- Drop indexes for repositories table
DROP INDEX IF EXISTS idx_repositories_last_scan_at;
DROP INDEX IF EXISTS idx_repositories_provider;
DROP INDEX IF EXISTS idx_repositories_organization_id;

-- Drop indexes for scan_results table
DROP INDEX IF EXISTS idx_scan_results_status;
DROP INDEX IF EXISTS idx_scan_results_agent_name;
DROP INDEX IF EXISTS idx_scan_results_scan_job_id;

-- Drop indexes for findings table
DROP INDEX IF EXISTS idx_findings_repo_status_severity;
DROP INDEX IF EXISTS idx_findings_scan_job_severity;
DROP INDEX IF EXISTS idx_findings_category;
DROP INDEX IF EXISTS idx_findings_tool;
DROP INDEX IF EXISTS idx_findings_file_path;
DROP INDEX IF EXISTS idx_findings_status;
DROP INDEX IF EXISTS idx_findings_severity;
DROP INDEX IF EXISTS idx_findings_scan_result_id;
DROP INDEX IF EXISTS idx_findings_scan_job_id;

-- Drop indexes for scan_jobs table
DROP INDEX IF EXISTS idx_scan_jobs_repo_branch_commit;
DROP INDEX IF EXISTS idx_scan_jobs_user_id;
DROP INDEX IF EXISTS idx_scan_jobs_status;
DROP INDEX IF EXISTS idx_scan_jobs_created_at;
DROP INDEX IF EXISTS idx_scan_jobs_repository_status;