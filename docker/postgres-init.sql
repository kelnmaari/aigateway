-- PostgreSQL initialization script for Ollama-OpenAI Proxy
-- Version 1.3.0

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Set timezone
SET timezone = 'UTC';

-- Create schema
CREATE SCHEMA IF NOT EXISTS public;

-- Grant privileges
GRANT ALL PRIVILEGES ON SCHEMA public TO proxy_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO proxy_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO proxy_user;

-- Note: Tables will be created automatically by the application's migration system
-- This script only sets up the database extensions and permissions

-- Optional: Create admin user for database management
-- CREATE ROLE admin WITH LOGIN PASSWORD 'admin_password' SUPERUSER;

-- Optional: Enable query logging for debugging
-- ALTER SYSTEM SET log_statement = 'all';
-- ALTER SYSTEM SET log_duration = on;
-- SELECT pg_reload_conf();

