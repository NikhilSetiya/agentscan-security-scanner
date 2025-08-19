-- Add supabase_id column to users table for Supabase authentication integration

ALTER TABLE users ADD COLUMN supabase_id VARCHAR(255) UNIQUE;

-- Create index for faster lookups by supabase_id
CREATE INDEX idx_users_supabase_id ON users(supabase_id);

-- Add comment to document the column
COMMENT ON COLUMN users.supabase_id IS 'Supabase Auth user ID for authentication integration';