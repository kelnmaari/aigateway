
-- ========================================
-- Add Changelog v1.5.9 (Migration v14)
-- ========================================

INSERT INTO changelogs (version, release_date, content) VALUES
('1.5.9', '2025-10-10', '## [1.5.9] - 2025-10-10

### Fixed
- **Members Display Issue**: Username и email не отображались
  - SQL JOIN с таблицей users
  - TenantMember model: +Username, +Email
  - scanTenantMemberWithUserInfo()

- **Search Results UX**: Улучшена контрастность
  - Черный текст на белом фоне
  - Светло-зеленый hover
  - Темно-серый email

- **GetTenant Response**: Исправлен wrapper
  - Frontend: response.tenant extraction
  - currentTenant.id fix

### Technical
- SQL JOIN optimization
- Nullable fields handling
- Frontend state management')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	