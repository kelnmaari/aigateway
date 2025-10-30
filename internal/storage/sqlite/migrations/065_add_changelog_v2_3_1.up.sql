
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.3.1', '2025-10-28', '## [2.3.1] - 2025-10-28

### Changed

- **Admin Panel UI Refactoring**: Реорганизация табов для улучшения навигации
  - **Users & RBAC**: Объединены Users, Invitations и RBAC в один таб с подтабами
  - **Models**: Объединены Available Models и Model Registry в один таб с подтабами
  - **System & Logs**: Объединены Performance, Audit и Logs в один таб с подтабами
  - **Итого**: Сокращение с 11 до 8 основных табов

### Fixed

- **Logs Tab Loading**: Исправлена ошибка дублирования переменной logsViewer
  - Обновлен селектор в logs.js для работы с подтабами
  - Логи корректно загружаются при переключении на подтаб

### Technical

- **Frontend**: Добавлена система подтабов (.sub-tabs, .sub-tab-btn, .sub-tab-pane)
- **JavaScript**: Логика переключения подтабов, loadAuditPreview() функция
- **CSS**: Стили для визуального разделения основных табов и подтабов');
	