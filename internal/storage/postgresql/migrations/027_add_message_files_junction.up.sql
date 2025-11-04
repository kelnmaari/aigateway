
-- ========================================
-- Message Files Junction Table (FILE-STORAGE-01: Phase 4)
-- ========================================
-- Связь между сообщениями и прикрепленными файлами (many-to-many)
CREATE TABLE IF NOT EXISTS message_files (
	message_id TEXT NOT NULL,
	file_id TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	PRIMARY KEY (message_id, file_id),
	FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
	FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_message_files_message_id ON message_files(message_id);
CREATE INDEX IF NOT EXISTS idx_message_files_file_id ON message_files(file_id);
	