INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.7', '2026-03-25', '## [4.13.7] - 2026-03-25

### Fixed
- **Web Search Error Flash**: Fixed error messages from web search flashing briefly in chat UI and disappearing before user could read them
- **Empty Assistant Message on Tool Failure**: Tool flow errors now show as persistent error messages instead of empty chat bubbles
- **SSE Stream Not Terminated on Error**: Backend now sends `data: [DONE]` after error tool events for proper stream termination
- **streamFinalResponse Silent Failure**: Added LLM response status check — errors from downstream LLM are no longer silently swallowed
- **Content Field Omitted in Tool-Calling Messages**: Removed `omitempty` from `Content` JSON tag so `null` content is preserved for tool-calling messages

### Technical
- Fixed `ChatMessage.Content` JSON tag, error event handling, and status checks in `chat_tools_handler.go`
- Frontend error tool events are now converted to persistent assistant error messages in `+page.svelte`')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
