INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.6', '2026-03-16', '## [4.13.6] - 2026-03-16

### Fixed
- **Chat Empty Response Crash**: Fixed 400 error when streaming returns empty response — previously `CreateMessageRequest.Content required` validation rejected empty assistant messages
- **Frontend Empty Save**: Frontend no longer attempts to save empty assistant responses to backend after streaming completes with no content

### Technical
- Changed `Content` field in `CreateMessageRequest` from `binding:"required"` to no binding constraint
- Added `fullResponse.trim()` guard before saving assistant message in SvelteKit chat page')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
