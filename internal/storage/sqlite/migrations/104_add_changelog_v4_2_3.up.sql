INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.3', '2024-12-31', '## [4.2.3] - 2024-12-31

### Fixed

- **Vulnerability Details in Create Issue**: Fixed "undefined (N/A)" in issue descriptions
  - Added cve_id and title fields to Vulnerability struct
  - Extracting CVE ID from OSV aliases (e.g., CVE-2024-XXXX)
  - Using summary as vulnerability title
  - Now shows proper CVE IDs and descriptions in GitLab issues

- **MR Creation with Default Branch**: Fixed "invalid reference name ''main''" error
  - CreateProject now saves default_branch from GitLab API
  - UpdateProject supports updating default_branch
  - Added fallback to "main" if default branch is not set
  - Added "Default Branch" field in project edit modal (WebUI)

### Technical

- internal/gitlab/dependencies/security/osv.go: Added CVEID, Title fields
- internal/gitlab/storage/postgres.go: Fixed CreateProject/UpdateProject for default_branch
- internal/api/handlers/gitlab_testgen.go, gitlab_autodoc.go: Added fallback for empty DefaultBranch
- web-svelte: Added Default Branch field to project edit modal');
