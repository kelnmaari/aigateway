
-- Add unique index on tenants.name for OIDC auto-provisioning (Version 1.11.2+)
-- This ensures tenant names are unique and speeds up GetTenantByName lookups

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_name_unique ON tenants(name);

-- Also create a regular index on tenants.slug if not already exists (for completeness)
CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
	