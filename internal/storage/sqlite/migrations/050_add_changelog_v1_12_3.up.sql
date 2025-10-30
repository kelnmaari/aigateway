
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.12.3', '2025-10-26', '## [1.12.3] - 2025-10-26

### Added
- **Conversation Export/Import**: Полная система экспорта и импорта conversations
  - Export форматы: JSON, Markdown, Text
  - Single conversation export через GET /api/conversations/{id}/export?format=json|markdown|text
  - Bulk export через POST /api/conversations/bulk-export
  - Import из JSON через POST /api/conversations/import
  - Import опции: Merge into existing, Preserve timestamps/IDs
  - Metadata export: total messages, tokens used, model, dates

### Technical
- Новый сервис internal/services/export/conversation_exporter.go
- Новый сервис internal/services/export/conversation_importer.go
- Новый handler internal/api/handlers/conversation_export.go
- Data Models в internal/models/conversation_export.go

### Security
- **Access Control**: Verify conversation ownership при export/import
- **User Isolation**: Импорт только в свой tenant/user scope');
    