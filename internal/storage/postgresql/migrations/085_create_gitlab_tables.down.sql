-- Rollback: Drop GitLab Integration Tables
-- Migration: 085_create_gitlab_tables

-- Drop triggers first
DROP TRIGGER IF EXISTS gitlab_projects_updated_at ON gitlab_projects;
DROP TRIGGER IF EXISTS gitlab_integrations_updated_at ON gitlab_integrations;

-- Drop function
DROP FUNCTION IF EXISTS update_gitlab_updated_at();

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS gitlab_webhook_events;
DROP TABLE IF EXISTS gitlab_analysis_jobs;
DROP TABLE IF EXISTS gitlab_mr_reviews;
DROP TABLE IF EXISTS gitlab_projects;
DROP TABLE IF EXISTS gitlab_integrations;

