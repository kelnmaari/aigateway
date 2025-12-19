-- Add missing columns to gitlab_webhook_events table
-- Migration: 090_add_webhook_events_payload

ALTER TABLE gitlab_webhook_events ADD COLUMN payload TEXT;
ALTER TABLE gitlab_webhook_events ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';

