-- Rollback: Drop mcp_servers table
DROP INDEX IF EXISTS idx_mcp_servers_category;
DROP INDEX IF EXISTS idx_mcp_servers_is_active;
DROP INDEX IF EXISTS idx_mcp_servers_created_at;
DROP TABLE IF EXISTS mcp_servers;
