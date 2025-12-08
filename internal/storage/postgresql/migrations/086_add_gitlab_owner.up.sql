-- Add owner_id and tenant_id to gitlab_integrations
-- Allows users to create their own GitLab integrations

ALTER TABLE gitlab_integrations 
ADD COLUMN IF NOT EXISTS owner_id TEXT REFERENCES users(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS tenant_id TEXT REFERENCES tenants(id) ON DELETE SET NULL;

-- Create indexes for owner and tenant queries
CREATE INDEX IF NOT EXISTS idx_gitlab_integrations_owner ON gitlab_integrations(owner_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_integrations_tenant ON gitlab_integrations(tenant_id);

-- Update existing integrations to have no owner (admin-created)
-- These can be managed by admins only
COMMENT ON COLUMN gitlab_integrations.owner_id IS 'User who created the integration. NULL means admin-created (global).';
COMMENT ON COLUMN gitlab_integrations.tenant_id IS 'Optional tenant for shared access within organization.';
