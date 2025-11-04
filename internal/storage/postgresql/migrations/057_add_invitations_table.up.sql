-- Enable required PostgreSQL extensions (if not already enabled)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ========================================
-- Invitations Table (AUTH-03: Invitation-Only Registration System, v2.2.0)
-- ========================================
CREATE TABLE IF NOT EXISTS invitations (
	id TEXT PRIMARY KEY DEFAULT (encode(gen_random_bytes(16), 'hex')),
	token TEXT UNIQUE NOT NULL,
	
	-- Creation metadata
	created_by_user_id TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	
	-- Constraints
	email TEXT,                      -- Optional: bind to specific email
	expires_at TIMESTAMP,            -- Optional: expiration date
	max_uses INTEGER NOT NULL DEFAULT 1,  -- Default: single-use
	
	-- Usage tracking
	current_uses INTEGER NOT NULL DEFAULT 0,
	used_at TIMESTAMP,               -- First successful registration
	used_by_user_id TEXT,
	
	-- Revocation
	revoked_at TIMESTAMP,
	revoked_by_user_id TEXT,
	revoke_reason TEXT,
	
	-- Foreign keys
	FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (used_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
	FOREIGN KEY (revoked_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
	
	-- Constraints
	CHECK (current_uses <= max_uses)
);

-- Indexes for performance
CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_created_by ON invitations(created_by_user_id);
CREATE INDEX idx_invitations_status ON invitations(expires_at, revoked_at, current_uses, max_uses);
CREATE INDEX idx_invitations_email ON invitations(email) WHERE email IS NOT NULL;
	