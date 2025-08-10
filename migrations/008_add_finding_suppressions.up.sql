-- Add finding suppressions table for managing false positives
CREATE TABLE finding_suppressions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    rule_id VARCHAR(200) NOT NULL,
    file_path VARCHAR(1000),
    line_number INTEGER,
    reason TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add indexes for performance
CREATE INDEX idx_finding_suppressions_user_id ON finding_suppressions(user_id);
CREATE INDEX idx_finding_suppressions_rule_id ON finding_suppressions(rule_id);
CREATE INDEX idx_finding_suppressions_file_path ON finding_suppressions(file_path);
CREATE INDEX idx_finding_suppressions_expires_at ON finding_suppressions(expires_at);

-- Add composite index for common queries
CREATE INDEX idx_finding_suppressions_rule_file ON finding_suppressions(rule_id, file_path);

-- Update findings table to support better status management
ALTER TABLE findings ADD COLUMN IF NOT EXISTS updated_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE findings ADD COLUMN IF NOT EXISTS status_comment TEXT;

-- Add index for status queries
CREATE INDEX IF NOT EXISTS idx_findings_status_updated ON findings(status, updated_at);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_finding_suppressions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_finding_suppressions_updated_at
    BEFORE UPDATE ON finding_suppressions
    FOR EACH ROW
    EXECUTE FUNCTION update_finding_suppressions_updated_at();