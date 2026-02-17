INSERT INTO changelogs (version, release_date, content) VALUES
('4.12.2', '2026-02-17', '## [4.12.2] - 2026-02-17

### Fixed
- **Model Registry**: Deleting a provider now cascades to delete its models from registry
- **Chat UI**: Models from deleted or disabled providers no longer appear in model dropdown

### Technical
- DeleteModelRegistryByProviderID() — explicit cleanup before provider deletion
- /v1/models filters out models whose provider is missing or disabled')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
