-- Add comment documenting that user_type now supports 'admin' value
-- The column VARCHAR(20) already has enough space for 'admin'
-- This migration serves as documentation of the schema change

COMMENT ON COLUMN users.user_type IS 'User type: guest, premium, or admin';
