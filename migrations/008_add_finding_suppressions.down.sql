-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_finding_suppressions_updated_at ON finding_suppressions;
DROP FUNCTION IF EXISTS update_finding_suppressions_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_finding_suppressions_rule_file;
DROP INDEX IF EXISTS idx_finding_suppressions_expires_at;
DROP INDEX IF EXISTS idx_finding_suppressions_file_path;
DROP INDEX IF EXISTS idx_finding_suppressions_rule_id;
DROP INDEX IF EXISTS idx_finding_suppressions_user_id;
DROP INDEX IF EXISTS idx_findings_status_updated;

-- Remove columns from findings table
ALTER TABLE findings DROP COLUMN IF EXISTS status_comment;
ALTER TABLE findings DROP COLUMN IF EXISTS updated_by;

-- Drop finding suppressions table
DROP TABLE IF EXISTS finding_suppressions;