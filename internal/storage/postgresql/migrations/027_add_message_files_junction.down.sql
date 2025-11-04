-- Rollback: Drop message_files table
DROP INDEX IF EXISTS idx_message_files_message_id;
DROP INDEX IF EXISTS idx_message_files_file_id;
DROP TABLE IF EXISTS message_files;
