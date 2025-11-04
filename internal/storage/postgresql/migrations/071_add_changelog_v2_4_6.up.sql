
INSERT INTO changelogs (version, release_date, content) VALUES
('2.4.6', '2025-10-29', '## [2.4.6] - 2025-10-29

### Fixed

- **CRITICAL: WebUI Modal Overlay Conflict**: On "My Devices" page, all buttons become unclickable (including top menu)
  - **Root Cause**: CSS class name conflict between device details modal (.modal-overlay in profile-devices.html) and confirmation modals (.modal-overlay in notifications.js)
  - **Problem**: When confirmation modal closes, it removes modal-show class but device details CSS has display: flex always visible, leaving invisible overlay blocking clicks
  - **Solution**: Renamed device details modal class from .modal-overlay to .device-modal-overlay
  - **Impact**: All buttons and navigation now work correctly after using confirmation modals
  - **Files Modified**: web/profile-devices.html and web/js/profile-devices.js

### Technical

- **CSS Specificity Issue**: Global class names like .modal-overlay should be avoided or properly scoped
- **Best Practice**: Use component-specific class names (e.g., .device-modal-overlay, .tenant-modal-overlay)
- **Future**: Consider CSS modules or scoped styles to prevent conflicts')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	