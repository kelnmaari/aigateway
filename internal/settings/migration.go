// Package settings - Database migration for settings table
// Version: v3.0.9 - Phase 4
package settings

const MigrationSQL = `
-- Settings table (v3.0.9+: Configuration in database)
CREATE TABLE IF NOT EXISTS settings (
    id VARCHAR(255) PRIMARY KEY,  -- e.g., "server.port", "auth.enabled"
    category VARCHAR(50) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    type VARCHAR(20) NOT NULL,  -- string, int, bool, float, json, array, duration
    default_value TEXT,
    description TEXT,
    is_editable BOOLEAN DEFAULT FALSE,
    is_required BOOLEAN DEFAULT FALSE,
    is_migrated BOOLEAN DEFAULT FALSE,  -- Migrated from YAML config
    requires_restart BOOLEAN DEFAULT TRUE,  -- Phase 4: Hot-reload support
    validation_rule TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    
    UNIQUE(category, key)
);

CREATE INDEX IF NOT EXISTS idx_settings_category ON settings(category);
CREATE INDEX IF NOT EXISTS idx_settings_migrated ON settings(is_migrated);
CREATE INDEX IF NOT EXISTS idx_settings_editable ON settings(is_editable);
CREATE INDEX IF NOT EXISTS idx_settings_requires_restart ON settings(requires_restart);
`
