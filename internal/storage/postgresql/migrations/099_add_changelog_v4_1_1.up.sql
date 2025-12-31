INSERT INTO changelogs (version, release_date, content) VALUES
('4.1.1', '2024-12-31', '## [4.1.1] - 2024-12-31

### Fixed

- **LLM API Authorization for Code Analysis**: Fixed 401 errors for Code Quality, Dead Code, AutoDoc, TestGen
  - Added `gitlabAPIKey` field to Router struct
  - All analysis handlers now use the same API key as MR Review workers
  - Fixes "Authentication required" errors in Code Quality Score modal

- **Analytics Page Data Format**: Fixed empty Analytics tab in GitLab UI
  - Handler now returns data in format expected by frontend (overview, top_projects, etc.)
  - Added success_rate calculation
  - Added reviews_by_day to response

- **Feedback Page Data Format**: Fixed empty Feedback tab in GitLab UI
  - Handler now returns stats, category_accuracy, model_accuracy, recent
  - Added feedback statistics calculation (approval_rate, accuracy_rate)
  - Added category accuracy breakdown

- **Missing Route Aliases**: Fixed 404 errors for Test Generation and Auto-Documentation
  - Added `/scan-docs` alias for `/scan-undocumented`
  - Added `/scan-tests` alias for `/scan-testable`

### Technical

- Router struct: added `gitlabAPIKey string` field for storing internal API key
- GetAnalytics handler: converted response format to match frontend expectations
- ListFeedback handler: added aggregate statistics calculation')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

