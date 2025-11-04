
INSERT INTO changelogs (version, release_date, content) VALUES
('2.4.3', '2025-10-28', '## [2.4.3] - 2025-10-28

### Added

- **WebSocket Streaming для Desktop** (DESKTOP-03): Real-time chat через WebSocket
  - **WebSocket Chat Endpoint**: GET /ws/chat?token={api_key}
    - API key authentication через query parameter
    - Bcrypt validation для безопасности
    - Device tracking: last_seen_at updates
  - **Chat Message Types**: chat_request, chat_chunk, chat_done, chat_error, ping/pong
  - **Per-User Message Routing**: Hub.SendToUser для targeted messaging
  - **ChatHandler**: Ollama integration, async processing, token counting

### Technical

- **Backend**: chat_handler.go (267 lines), handler.go updates, hub.go per-user tracking
- **Security**: Bcrypt validation, status/expiration checks
- **Performance**: Ping/Pong heartbeat (54s), async processing
- **Backward Compatibility**: SSE endpoints продолжают работать')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	