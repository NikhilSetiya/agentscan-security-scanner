-- Add repository management enhancements for production

-- Add missing columns to repositories table for better management
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS language VARCHAR(100);
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS scan_config JSONB DEFAULT '{}';

-- Add indexes for repository management
CREATE INDEX IF NOT EXISTS idx_repositories_name ON repositories(name);
CREATE INDEX IF NOT EXISTS idx_repositories_language ON repositories(language);
CREATE INDEX IF NOT EXISTS idx_repositories_is_active ON repositories(is_active);
CREATE INDEX IF NOT EXISTS idx_repositories_org_active ON repositories(organization_id, is_active);

-- Add dashboard statistics view for better performance
CREATE OR REPLACE VIEW dashboard_stats AS
SELECT 
    COUNT(DISTINCT sj.id) as total_scans,
    COUNT(DISTINCT r.id) as total_repositories,
    COUNT(CASE WHEN f.severity = 'critical' THEN 1 END) as critical_findings,
    COUNT(CASE WHEN f.severity = 'high' THEN 1 END) as high_findings,
    COUNT(CASE WHEN f.severity = 'medium' THEN 1 END) as medium_findings,
    COUNT(CASE WHEN f.severity = 'low' THEN 1 END) as low_findings,
    COUNT(CASE WHEN f.severity = 'info' THEN 1 END) as info_findings
FROM repositories r
LEFT JOIN scan_jobs sj ON r.id = sj.repository_id
LEFT JOIN findings f ON sj.id = f.scan_job_id
WHERE r.is_active = true;

-- Add repository statistics view
CREATE OR REPLACE VIEW repository_stats AS
SELECT 
    r.id,
    r.name,
    r.url,
    r.language,
    r.last_scan_at,
    COUNT(DISTINCT sj.id) as total_scans,
    COUNT(CASE WHEN sj.status = 'completed' THEN 1 END) as completed_scans,
    COUNT(CASE WHEN sj.status = 'failed' THEN 1 END) as failed_scans,
    COUNT(DISTINCT f.id) as total_findings,
    COUNT(CASE WHEN f.severity = 'critical' THEN 1 END) as critical_findings,
    COUNT(CASE WHEN f.severity = 'high' THEN 1 END) as high_findings,
    COUNT(CASE WHEN f.severity = 'medium' THEN 1 END) as medium_findings,
    COUNT(CASE WHEN f.severity = 'low' THEN 1 END) as low_findings,
    MAX(sj.created_at) as last_scan_created_at
FROM repositories r
LEFT JOIN scan_jobs sj ON r.id = sj.repository_id
LEFT JOIN findings f ON sj.id = f.scan_job_id
WHERE r.is_active = true
GROUP BY r.id, r.name, r.url, r.language, r.last_scan_at;

-- Add scan job statistics view
CREATE OR REPLACE VIEW scan_job_stats AS
SELECT 
    sj.id,
    sj.repository_id,
    sj.status,
    sj.scan_type,
    sj.created_at,
    sj.started_at,
    sj.completed_at,
    EXTRACT(EPOCH FROM (COALESCE(sj.completed_at, NOW()) - sj.started_at)) as duration_seconds,
    COUNT(DISTINCT f.id) as findings_count,
    COUNT(CASE WHEN f.severity = 'critical' THEN 1 END) as critical_count,
    COUNT(CASE WHEN f.severity = 'high' THEN 1 END) as high_count,
    COUNT(CASE WHEN f.severity = 'medium' THEN 1 END) as medium_count,
    COUNT(CASE WHEN f.severity = 'low' THEN 1 END) as low_count,
    COUNT(CASE WHEN f.severity = 'info' THEN 1 END) as info_count,
    r.name as repository_name,
    r.url as repository_url,
    r.language as repository_language
FROM scan_jobs sj
JOIN repositories r ON sj.repository_id = r.id
LEFT JOIN findings f ON sj.id = f.scan_job_id
GROUP BY sj.id, sj.repository_id, sj.status, sj.scan_type, sj.created_at, 
         sj.started_at, sj.completed_at, r.name, r.url, r.language;

-- Add function to update repository last_scan_at when scan completes
CREATE OR REPLACE FUNCTION update_repository_last_scan()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'completed' AND OLD.status != 'completed' THEN
        UPDATE repositories 
        SET last_scan_at = NEW.completed_at 
        WHERE id = NEW.repository_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update repository last_scan_at
DROP TRIGGER IF EXISTS trigger_update_repository_last_scan ON scan_jobs;
CREATE TRIGGER trigger_update_repository_last_scan
    AFTER UPDATE ON scan_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_repository_last_scan();

-- Add comments for documentation
COMMENT ON COLUMN repositories.language IS 'Primary programming language of the repository';
COMMENT ON COLUMN repositories.description IS 'Repository description';
COMMENT ON COLUMN repositories.is_active IS 'Whether the repository is actively being scanned';
COMMENT ON COLUMN repositories.scan_config IS 'Repository-specific scan configuration';

COMMENT ON VIEW dashboard_stats IS 'Aggregated statistics for dashboard display';
COMMENT ON VIEW repository_stats IS 'Per-repository statistics including scan and finding counts';
COMMENT ON VIEW scan_job_stats IS 'Detailed scan job statistics with findings breakdown';