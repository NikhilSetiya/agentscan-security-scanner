-- Remove supabase_id column from users table

DROP INDEX IF EXISTS idx_users_supabase_id;
ALTER TABLE users DROP COLUMN IF EXISTS supabase_id;