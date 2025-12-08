-- Rollback: Remove owner_id and tenant_id from gitlab_integrations

DROP INDEX IF EXISTS idx_gitlab_integrations_owner;
DROP INDEX IF EXISTS idx_gitlab_integrations_tenant;

ALTER TABLE gitlab_integrations 
DROP COLUMN IF EXISTS owner_id,
DROP COLUMN IF EXISTS tenant_id;
