
INSERT INTO changelogs (version, release_date, content) VALUES
('2.4.5', '2025-10-28', '## [2.4.5] - 2025-10-28

### Fixed

- **CRITICAL: WebSocket Panic Fix**: Server panic "send on closed channel" when client disconnects during streaming
  - **Root Cause**: Goroutine continues streaming after client''s Send channel closed
  - **Solution**: 
    - Added panic recovery in sendMessage() with defer recover()
    - Added 5-second timeout to detect disconnected clients
    - Stop streaming immediately on send error (graceful goroutine exit)
    - Ignore errors on final "done" message (client may be disconnected)
  - **Impact**: Server now handles client disconnections gracefully without crashes

### Technical

- **File Modified**: internal/websocket/chat_handler.go
  - sendMessage(): Added defer recover() to catch panic, 5s timeout with time.After()
  - processStreamingResponse(): Check sendMessage() error on every chunk, exit immediately on error
- **Design Patterns**: Panic recovery, timeout pattern, graceful goroutine shutdown
- **Documentation**: Added WEBSOCKET_PANIC_FIX.md with detailed analysis')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	