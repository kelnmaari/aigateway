INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.8', '2026-03-30', '## [5.2.8] - 2026-03-30

### Fixed

- **Anthropic→OpenAI message format conversion**: AIGateway now automatically converts Anthropic-format tool messages to OpenAI format before forwarding to inference providers. Kilo Code and other Claude SDK clients send type:"tool_use" / type:"tool_result" content blocks; vLLM/SGLang/TGI only understand tool_calls arrays and role:"tool" messages. Without this fix, every tool-augmented turn returned HTTP 400.
- **inference_proxy_handler: preserve all request fields**: Switched from struct-unmarshal+re-marshal to raw-body pass-through, so provider-specific fields are no longer silently dropped.

### Technical

- internal/api/handlers/message_converter.go — new file with normalizeMessagesInJSON, convertAssistantMessage, convertToolResultMessages, extractToolResultContent
- internal/api/handlers/inference_proxy_handler.go — HandleChatCompletions now reads raw body, normalizes messages, rewrites model field
- internal/api/handlers/chat_tools_handler.go — HandleChatWithTools normalizes messages before forwarding')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
