INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.9', '2026-03-30', '## [5.2.9] - 2026-03-30

### Fixed

- **Streaming error forwarding**: When vLLM returns a non-200 response on a streaming request, it is now forwarded as a plain JSON response instead of being wrapped in text/event-stream. Previously, Kilo Code reported "400 status code (no body)" because the SSE parser found no events in a raw JSON error body.
- **Mixed Anthropic content blocks preserved**: User messages with tool_result + text blocks no longer silently drop text blocks. Text blocks are forwarded as a separate role:user message after the converted role:tool messages.

### Technical

- internal/api/handlers/inference_proxy_handler.go — stream only on HTTP 200
- internal/api/handlers/chat_tools_handler.go — same fix in forwardRawRequest
- internal/api/handlers/message_converter.go — convertToolResultMessages returns optional extra role:user message')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
