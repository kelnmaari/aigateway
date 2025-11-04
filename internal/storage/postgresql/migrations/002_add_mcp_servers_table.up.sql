
-- ========================================
-- MCP Servers Table (WEBUI-07: v1.4.5)
-- ========================================
CREATE TABLE IF NOT EXISTS mcp_servers (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL,
	category TEXT NOT NULL,
	installation_guide TEXT NOT NULL,
	website_url TEXT,
	github_url TEXT,
	tags JSONB, --  array stored as TEXT
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mcp_servers_category ON mcp_servers(category);
CREATE INDEX idx_mcp_servers_is_active ON mcp_servers(is_active);
CREATE INDEX idx_mcp_servers_created_at ON mcp_servers(created_at DESC);
	