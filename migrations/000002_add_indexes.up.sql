-- Add performance indexes for common queries

-- Indexes for scan_jobs table
CREATE INDEX idx_scan_jobs_repository_status ON scan_jobs(repository_id, status);
CREATE INDEX idx_scan_jobs_created_at ON scan_jobs(created_at DESC);
CREATE INDEX idx_scan_jobs_status ON scan_jobs(status);
CREATE INDEX idx_scan_jobs_user_id ON scan_jobs(user_id);
CREATE INDEX idx_scan_jobs_repo_branch_commit ON scan_jobs(repository_id, branch, commit_sha);

-- Indexes for findings table
CREATE INDEX idx_findings_scan_job_id ON findings(scan_job_id);
CREATE INDEX idx_findings_scan_result_id ON findings(scan_result_id);
CREATE INDEX idx_findings_severity ON findings(severity);
CREATE INDEX idx_findings_status ON findings(status);
CREATE INDEX idx_findings_file_path ON findings(file_path);
CREATE INDEX idx_findings_tool ON findings(tool);
CREATE INDEX idx_findings_category ON findings(category);
CREATE INDEX idx_findings_scan_job_severity ON findings(scan_job_id, severity);
CREATE INDEX idx_findings_repo_status_severity ON findings(scan_job_id, status, severity);

-- Indexes for scan_results table
CREATE INDEX idx_scan_results_scan_job_id ON scan_results(scan_job_id);
CREATE INDEX idx_scan_results_agent_name ON scan_results(agent_name);
CREATE INDEX idx_scan_results_status ON scan_results(status);

-- Indexes for repositories table
CREATE INDEX idx_repositories_organization_id ON repositories(organization_id);
CREATE INDEX idx_repositories_provider ON repositories(provider);
CREATE INDEX idx_repositories_last_scan_at ON repositories(last_scan_at);

-- Indexes for organization_members table
CREATE INDEX idx_organization_members_organization_id ON organization_members(organization_id);
CREATE INDEX idx_organization_members_user_id ON organization_members(user_id);
CREATE INDEX idx_organization_members_role ON organization_members(role);

-- Indexes for user_feedback table
CREATE INDEX idx_user_feedback_finding_id ON user_feedback(finding_id);
CREATE INDEX idx_user_feedback_user_id ON user_feedback(user_id);
CREATE INDEX idx_user_feedback_action ON user_feedback(action);

-- Indexes for users table
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_github_id ON users(github_id);
CREATE INDEX idx_users_gitlab_id ON users(gitlab_id);

-- Indexes for organizations table
CREATE INDEX idx_organizations_slug ON organizations(slug);