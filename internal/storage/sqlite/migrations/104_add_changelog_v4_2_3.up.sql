INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.3', '2024-12-31', '## [4.2.3] - 2024-12-31

### Fixed

- **Vulnerability Details in Create Issue**: Fixed "undefined (N/A)" in issue descriptions
  - Added cve_id and title fields to Vulnerability struct
  - Extracting CVE ID from OSV aliases (e.g., CVE-2024-XXXX)
  - Using summary as vulnerability title
  - Now shows proper CVE IDs and descriptions in GitLab issues

### Technical

- internal/gitlab/dependencies/security/osv.go: Added CVEID, Title fields
- Added getPrimaryID() helper to extract CVE from OSV aliases');
