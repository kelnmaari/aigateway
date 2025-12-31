INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.2', '2024-12-31', '## [4.2.2] - 2024-12-31

### Fixed

- **llama.cpp Flash Attention**: Fixed --flash-attn flag for new llama.cpp versions
  - New llama.cpp server requires value: --flash-attn [on|off|auto]
  - Changed from --flash-attn to --flash-attn on when enabled
  - Added --flash-attn off when disabled (explicit control)
  - Fixes: error while handling argument "--flash-attn": expected value for argument

### Technical

- internal/inference/provider_builders.go: Updated buildLlamaCppRequest to use explicit on/off values');
