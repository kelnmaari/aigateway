-- Rollback: Drop invitations table
DROP INDEX IF EXISTS idx_invitations_token;
DROP INDEX IF EXISTS idx_invitations_created_by;
DROP INDEX IF EXISTS idx_invitations_status;
DROP INDEX IF EXISTS idx_invitations_email;
DROP TABLE IF EXISTS invitations;
