
-- Create audit_events table for comprehensive security and compliance logging (Version 1.11.4+)
CREATE TABLE IF NOT EXISTS audit_events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info',
    
    -- Actor (who performed the action)
    actor_id TEXT NOT NULL,
    actor_type TEXT NOT NULL DEFAULT 'user',
    
    -- Target (what was affected)
    target_id TEXT,
    target_type TEXT,
    
    -- Context
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    status TEXT NOT NULL,
    error_msg TEXT,
    metadata JSONB, 
    
    -- Request info
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_actor_id ON audit_events(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_severity ON audit_events(severity);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource ON audit_events(resource);
CREATE INDEX IF NOT EXISTS idx_audit_events_target_id ON audit_events(target_id) WHERE target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_events_status ON audit_events(status);
	