
-- Add OIDC authentication fields to users table (Version 1.11.1+: Keycloak SSO Integration)

-- auth_provider: Authentication provider type ('local', 'oidc', 'ldap')
ALTER TABLE users ADD COLUMN auth_provider TEXT DEFAULT 'local' NOT NULL;

-- oidc_subject: OIDC 'sub' claim (unique identifier from OIDC provider)
ALTER TABLE users ADD COLUMN oidc_subject TEXT;

-- oidc_issuer: OIDC issuer URL (e.g., https://keycloak.example.com/realms/myrealm)
ALTER TABLE users ADD COLUMN oidc_issuer TEXT;

-- Create index for fast lookup by OIDC subject
CREATE INDEX IF NOT EXISTS idx_users_oidc_subject ON users(oidc_subject) WHERE oidc_subject IS NOT NULL;

-- Create index for filtering by auth provider
CREATE INDEX IF NOT EXISTS idx_users_auth_provider ON users(auth_provider);

-- Create composite index for OIDC issuer + subject (for multi-provider scenarios)
CREATE INDEX IF NOT EXISTS idx_users_oidc_issuer_subject ON users(oidc_issuer, oidc_subject) 
    WHERE oidc_issuer IS NOT NULL AND oidc_subject IS NOT NULL;

-- Add unique constraint for OIDC subject (within same issuer)
-- Note: SQLite doesn't support adding unique constraints to existing columns directly,
-- so we create a unique index instead
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_unique ON users(oidc_issuer, oidc_subject) 
    WHERE oidc_issuer IS NOT NULL AND oidc_subject IS NOT NULL;
	