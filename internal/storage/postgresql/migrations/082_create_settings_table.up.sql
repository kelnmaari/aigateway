-- Migration: Create settings table for PostgreSQL-backed configuration management
-- Version: v3.0.9
-- Phase: Phase 1 - Initial Settings Infrastructure

CREATE TABLE IF NOT EXISTS settings (
    -- Identity
    id VARCHAR(255) PRIMARY KEY,
    category VARCHAR(50) NOT NULL,
    key VARCHAR(100) NOT NULL,
    
    -- Value and metadata
    value TEXT NOT NULL,
    type VARCHAR(20) NOT NULL, -- 'string', 'int', 'float', 'bool', 'duration'
    default_value TEXT,
    
    -- Documentation
    description TEXT,
    
    -- Flags
    is_editable BOOLEAN NOT NULL DEFAULT FALSE,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    is_migrated BOOLEAN NOT NULL DEFAULT FALSE,
    requires_restart BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Validation
    validation_rule TEXT,
    
    -- Audit
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(100) DEFAULT 'system',
    
    -- Constraints
    UNIQUE(category, key)
);

-- Indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_settings_category ON settings(category);
CREATE INDEX IF NOT EXISTS idx_settings_is_editable ON settings(is_editable);
CREATE INDEX IF NOT EXISTS idx_settings_is_migrated ON settings(is_migrated);
CREATE INDEX IF NOT EXISTS idx_settings_requires_restart ON settings(requires_restart);
CREATE INDEX IF NOT EXISTS idx_settings_updated_at ON settings(updated_at);

-- Comments
COMMENT ON TABLE settings IS 'PostgreSQL-backed configuration management system (v3.0.9)';
COMMENT ON COLUMN settings.id IS 'Unique setting identifier (e.g., server.host)';
COMMENT ON COLUMN settings.category IS 'Setting category for grouping (server, auth, database, etc.)';
COMMENT ON COLUMN settings.key IS 'Setting key within category';
COMMENT ON COLUMN settings.value IS 'Current setting value (stored as string, parsed by type)';
COMMENT ON COLUMN settings.type IS 'Value data type for validation and parsing';
COMMENT ON COLUMN settings.is_editable IS 'Whether setting can be edited via UI (Phase 3)';
COMMENT ON COLUMN settings.is_migrated IS 'Whether setting was migrated from YAML config';
COMMENT ON COLUMN settings.requires_restart IS 'Whether changing this setting requires server restart (Phase 4)';
COMMENT ON COLUMN settings.validation_rule IS 'Regex pattern for value validation';

