
-- Add LDAP DN column for LDAP/Active Directory authentication (Version 1.11.3+)
-- ldap_dn stores the LDAP Distinguished Name of the user

ALTER TABLE users ADD COLUMN ldap_dn TEXT;

-- Create unique index for LDAP DN (cannot use UNIQUE constraint in ALTER TABLE ADD COLUMN)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_ldap_dn_unique ON users(ldap_dn) WHERE ldap_dn IS NOT NULL;
	