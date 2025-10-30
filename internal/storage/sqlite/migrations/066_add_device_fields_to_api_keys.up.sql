
-- ========================================
-- Add Device Fields to api_keys (Migration v66)
-- Version 2.4.0: Desktop Client Support
-- ========================================

-- Add device metadata columns
ALTER TABLE api_keys ADD COLUMN device_name TEXT;
ALTER TABLE api_keys ADD COLUMN device_os TEXT;
ALTER TABLE api_keys ADD COLUMN device_hostname TEXT;
ALTER TABLE api_keys ADD COLUMN device_version TEXT;
ALTER TABLE api_keys ADD COLUMN device_fingerprint TEXT;
ALTER TABLE api_keys ADD COLUMN last_seen_at DATETIME;
ALTER TABLE api_keys ADD COLUMN auto_expire_at DATETIME;

-- Create indices for device-related queries
CREATE INDEX IF NOT EXISTS idx_api_keys_device_fingerprint ON api_keys(device_fingerprint);
CREATE INDEX IF NOT EXISTS idx_api_keys_last_seen ON api_keys(last_seen_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_device_os ON api_keys(device_os);

-- Note: device_fingerprint будет использоваться для duplicate detection
-- last_seen_at автоматически обновляется middleware при каждом API request
	