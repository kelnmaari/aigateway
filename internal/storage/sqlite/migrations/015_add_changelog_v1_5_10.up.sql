
-- ========================================
-- Add Changelog v1.5.10 (Migration v15)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.10', '2025-10-10', '## [1.5.10] - 2025-10-10

### Fixed
- **Tenant Member Count**: Количество участников не отображалось
  - SQL подзапрос COUNT(*)
  - Tenant model: +MemberCount
  - scanTenantWithRole updated

### Technical
- SQL Subquery optimization
- Minimal code changes
- Frontend compatibility');
	