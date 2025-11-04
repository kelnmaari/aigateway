
INSERT INTO changelogs (version, release_date, content) VALUES
('2.4.4', '2025-10-28', '## [2.4.4] - 2025-10-28

### Added

- **Desktop-Specific WebUI Features** (DESKTOP-04): Enhanced device management experience
  - **Device Details Modal**: Full-screen modal с полной информацией об устройстве
    - Клик на device card/icon открывает modal
    - Display: OS, version, hostname, API key ID, dates, usage stats
    - Action buttons: Rename, Revoke Access, Close
    - Smooth animations (fadeIn, slideIn)
  - **Bulk Device Revoke**: Checkbox selection + bulk actions bar
    - Bulk actions bar (fixed bottom, slide up animation)
    - Bulk revoke с confirmation modal
    - Protection: нельзя bulk revoke current device
    - Success/error toast notifications
  - **Enhanced UI/UX**: View Details button, event.stopPropagation, animations

### Changed

- **Device Cards**: Добавлен checkbox для bulk selection, клик на icon/name открывает modal
- **Actions**: Добавлена кнопка "View Details"

### Technical

- **Frontend**: profile-devices.html, profile-devices.js (180+ lines new code)
- **Methods**: openDeviceDetails, renderDeviceDetailsHTML, enableBulkSelection, bulkRevokeDevices
- **CSS**: modal-overlay, device-details-modal, -- animations (fadeIn, slideIn, slideUp)
- **Security**: Current device НЕ может быть bulk revoked
- **Performance**: Modal рендерится on-demand, async bulk operations')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	