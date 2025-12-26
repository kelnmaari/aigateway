-- Add missing columns to gitlab_webhook_events table
-- Migration: 090_add_webhook_events_payload

ALTER TABLE gitlab_webhook_events ADD COLUMN IF NOT EXISTS payload JSONB;
ALTER TABLE gitlab_webhook_events ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'pending';

