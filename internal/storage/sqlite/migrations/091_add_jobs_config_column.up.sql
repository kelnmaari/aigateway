-- Add missing config column to gitlab_analysis_jobs table
-- Migration: 091_add_jobs_config_column

ALTER TABLE gitlab_analysis_jobs ADD COLUMN config TEXT;

