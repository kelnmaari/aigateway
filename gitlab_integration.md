# GitLab MR Review Integration

## 📋 Обзор

Интеграция AIGateway с GitLab для автоматического AI-ревью Merge Request'ов. Система получает webhook от GitLab при создании/обновлении MR, анализирует код с помощью LLM и публикует структурированный отзыв как комментарий к MR.

## 🎯 Цели

1. **Автоматизация Code Review** - AI анализирует каждый MR без участия человека
2. **Качество кода** - Выявление багов, уязвимостей, code smells
3. **Консистентность** - Единый стиль и стандарты для всех MR
4. **Ускорение процесса** - Быстрый первичный фидбек до человеческого ревью

## 🏗️ Архитектура

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              GitLab Server                                   │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐                                  │
│  │ Project │    │   MR    │    │ Webhook │                                  │
│  └────┬────┘    └────┬────┘    └────┬────┘                                  │
└───────┼──────────────┼──────────────┼───────────────────────────────────────┘
        │              │              │
        │              │              │ POST /api/webhooks/gitlab/:id
        │              │              ▼
┌───────┼──────────────┼──────────────────────────────────────────────────────┐
│       │              │         AIGateway                                     │
│       │              │              │                                        │
│       │              │    ┌─────────▼─────────┐                              │
│       │              │    │  Webhook Handler  │                              │
│       │              │    │  - Verify secret  │                              │
│       │              │    │  - Parse payload  │                              │
│       │              │    └─────────┬─────────┘                              │
│       │              │              │                                        │
│       │              │    ┌─────────▼─────────┐                              │
│       │              │    │   Job Queue       │                              │
│       │              │    │   (async proc.)   │                              │
│       │              │    └─────────┬─────────┘                              │
│       │              │              │                                        │
│       │    GET /changes             │                                        │
│       │◄─────────────┼──────────────┤                                        │
│       │              │              │                                        │
│       │              │    ┌─────────▼─────────┐                              │
│       │              │    │  GitLab Client    │                              │
│       │              │    │  - Fetch diff     │                              │
│       │              │    │  - Get files      │                              │
│       │              │    └─────────┬─────────┘                              │
│       │              │              │                                        │
│       │              │    ┌─────────▼─────────┐     ┌──────────────┐         │
│       │              │    │  Code Chunker     │     │   Qdrant     │         │
│       │              │    │  - Split by file  │────▶│  (temp coll) │         │
│       │              │    │  - AST parsing    │     │  embeddings  │         │
│       │              │    └─────────┬─────────┘     └──────┬───────┘         │
│       │              │              │                      │                 │
│       │              │    ┌─────────▼──────────────────────▼───┐             │
│       │              │    │         AI Analyzer                │             │
│       │              │    │  - Security check                  │             │
│       │              │    │  - Bug detection                   │             │
│       │              │    │  - Style review                    │             │
│       │              │    │  - Performance tips                │             │
│       │              │    └─────────┬──────────────────────────┘             │
│       │              │              │                                        │
│       │   POST /notes│              │                                        │
│       │◄─────────────┼──────────────┤                                        │
│       │              │    ┌─────────▼─────────┐                              │
│       │              │    │  Comment Builder  │                              │
│       │              │    │  - Format MD      │                              │
│       │              │    │  - Line comments  │                              │
│       │              │    └───────────────────┘                              │
│       │              │                                                       │
└───────┴──────────────┴───────────────────────────────────────────────────────┘
```

## 📦 Компоненты

### 1. Database Models

```go
// GitLab Integration - подключение к GitLab instance
type GitLabIntegration struct {
    ID            string    `json:"id"`
    Name          string    `json:"name"`              // "Company GitLab"
    BaseURL       string    `json:"base_url"`          // "https://gitlab.company.com"
    AccessToken   string    `json:"-"`                 // Encrypted PAT or OAuth token
    WebhookSecret string    `json:"-"`                 // For webhook verification
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
    Status        string    `json:"status"`            // active, disabled, error
    LastSyncAt    time.Time `json:"last_sync_at"`
}

// GitLab Project - репозиторий для анализа
type GitLabProject struct {
    ID              string    `json:"id"`
    IntegrationID   string    `json:"integration_id"`
    GitLabProjectID int       `json:"gitlab_project_id"`
    Name            string    `json:"name"`
    PathWithNS      string    `json:"path_with_namespace"` // "group/project"
    WebhookID       int       `json:"webhook_id"`          // GitLab webhook ID
    AutoReview      bool      `json:"auto_review"`         // Auto-review new MRs
    
    // Model Configuration
    EmbeddingModel  string    `json:"embedding_model"`     // Model for code chunking/embedding
    AnalysisModel   string    `json:"analysis_model"`      // Model for code review (LLM)
    
    ReviewPrompt    string    `json:"review_prompt"`       // Custom prompt
    FileFilters     []string  `json:"file_filters"`        // ["*.go", "*.ts", "!vendor/*"]
    CreatedAt       time.Time `json:"created_at"`
    Status          string    `json:"status"`
}

// MR Review - результат анализа
type MRReview struct {
    ID              string    `json:"id"`
    ProjectID       string    `json:"project_id"`
    MRIid           int       `json:"mr_iid"`
    MRTitle         string    `json:"mr_title"`
    MRAuthor        string    `json:"mr_author"`
    SourceBranch    string    `json:"source_branch"`
    TargetBranch    string    `json:"target_branch"`
    Status          string    `json:"status"`           // pending, analyzing, completed, failed
    FilesAnalyzed   int       `json:"files_analyzed"`
    LinesChanged    int       `json:"lines_changed"`
    IssuesFound     int       `json:"issues_found"`
    ReviewResult    string    `json:"review_result"`    // JSON with structured review
    CommentID       int       `json:"comment_id"`       // GitLab note ID
    ProcessingTime  int       `json:"processing_time_ms"`
    TokensUsed      int       `json:"tokens_used"`
    Model           string    `json:"model"`
    CreatedAt       time.Time `json:"created_at"`
    CompletedAt     time.Time `json:"completed_at"`
    Error           string    `json:"error,omitempty"`
}
```

### 2. API Endpoints

```
# GitLab Integrations
GET    /api/admin/gitlab/integrations              # List all integrations
POST   /api/admin/gitlab/integrations              # Add new integration
GET    /api/admin/gitlab/integrations/:id          # Get integration details
PUT    /api/admin/gitlab/integrations/:id          # Update integration
DELETE /api/admin/gitlab/integrations/:id          # Delete integration
POST   /api/admin/gitlab/integrations/:id/test     # Test connection

# GitLab Projects
GET    /api/admin/gitlab/integrations/:id/projects          # List projects
POST   /api/admin/gitlab/integrations/:id/projects          # Add project
GET    /api/admin/gitlab/projects/:project_id               # Get project
PUT    /api/admin/gitlab/projects/:project_id               # Update project
DELETE /api/admin/gitlab/projects/:project_id               # Delete project
POST   /api/admin/gitlab/projects/:project_id/webhook       # Setup webhook

# MR Reviews
GET    /api/admin/gitlab/reviews                   # List all reviews
GET    /api/admin/gitlab/projects/:id/reviews      # List project reviews
GET    /api/admin/gitlab/reviews/:id               # Get review details
POST   /api/admin/gitlab/reviews/:id/retry         # Retry failed review

# Webhook (public, verified by secret)
POST   /api/webhooks/gitlab/:integration_id        # Receive GitLab webhooks
```

### 3. GitLab API Client

```go
type GitLabClient struct {
    baseURL     string
    accessToken string
    httpClient  *http.Client
}

// Core methods
func (c *GitLabClient) GetProject(projectID int) (*Project, error)
func (c *GitLabClient) GetMergeRequest(projectID, mrIID int) (*MergeRequest, error)
func (c *GitLabClient) GetMRChanges(projectID, mrIID int) (*MRChanges, error)
func (c *GitLabClient) GetMRDiff(projectID, mrIID int) ([]FileDiff, error)
func (c *GitLabClient) PostMRNote(projectID, mrIID int, body string) (*Note, error)
func (c *GitLabClient) PostMRDiscussion(projectID, mrIID int, discussion Discussion) (*Discussion, error)
func (c *GitLabClient) CreateWebhook(projectID int, url, secret string) (*Webhook, error)
func (c *GitLabClient) DeleteWebhook(projectID, webhookID int) error
```

### 4. Code Chunking Strategies

```go
type ChunkStrategy string

const (
    ChunkByFile     ChunkStrategy = "file"      // Один чанк = один файл
    ChunkByFunction ChunkStrategy = "function"  // AST-based, по функциям
    ChunkByHunk     ChunkStrategy = "hunk"      // По блокам изменений в diff
    ChunkByLines    ChunkStrategy = "lines"     // Sliding window по N строк
)

type CodeChunk struct {
    ID          string            `json:"id"`
    FilePath    string            `json:"file_path"`
    ChangeType  string            `json:"change_type"`  // added, modified, deleted
    LineStart   int               `json:"line_start"`
    LineEnd     int               `json:"line_end"`
    Content     string            `json:"content"`
    Language    string            `json:"language"`
    Metadata    map[string]string `json:"metadata"`
}

type CodeChunker interface {
    Chunk(diff []FileDiff) ([]CodeChunk, error)
}
```

### 5. AI Analysis Pipeline

```go
type AnalysisPipeline struct {
    embedder     EmbeddingService
    vectorStore  VectorStore       // Qdrant
    llmClient    LLMClient
    promptConfig PromptConfig
}

type AnalysisResult struct {
    Summary       string           `json:"summary"`
    OverallScore  int              `json:"overall_score"`  // 0-100
    Categories    []CategoryResult `json:"categories"`
    FileReviews   []FileReview     `json:"file_reviews"`
    Suggestions   []Suggestion     `json:"suggestions"`
}

type CategoryResult struct {
    Name     string `json:"name"`      // security, performance, style, bugs
    Score    int    `json:"score"`
    Issues   int    `json:"issues"`
    Details  string `json:"details"`
}

type FileReview struct {
    FilePath    string        `json:"file_path"`
    Score       int           `json:"score"`
    LineIssues  []LineIssue   `json:"line_issues"`
    Summary     string        `json:"summary"`
}

type LineIssue struct {
    Line     int    `json:"line"`
    Severity string `json:"severity"`  // critical, warning, info
    Message  string `json:"message"`
    Category string `json:"category"`
}
```

### 6. Qdrant Integration

```go
// Temporary collection for MR analysis
collectionName := fmt.Sprintf("mr_review_%s_%d", projectID, mrIID)

// Store chunks with metadata
type MRChunkPayload struct {
    MRReviewID  string `json:"mr_review_id"`
    FilePath    string `json:"file_path"`
    ChangeType  string `json:"change_type"`
    LineStart   int    `json:"line_start"`
    LineEnd     int    `json:"line_end"`
    Language    string `json:"language"`
    ChunkIndex  int    `json:"chunk_index"`
}

// Cleanup after analysis
func (s *Service) cleanupMRCollection(collectionName string) error
```

## 🎨 UI Design (Admin Panel)

### GitLab Integrations Page

```
┌─────────────────────────────────────────────────────────────────┐
│  GitLab Integrations                              [+ Add New]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ 🦊 Company GitLab                              ● Active   │  │
│  │    https://gitlab.company.com                             │  │
│  │    3 projects • 127 reviews • Last sync: 5 min ago        │  │
│  │                                        [Edit] [Projects]  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ 🦊 GitLab.com                                  ● Active   │  │
│  │    https://gitlab.com                                     │  │
│  │    1 project • 23 reviews • Last sync: 1 hour ago         │  │
│  │                                        [Edit] [Projects]  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Add Integration Modal

```
┌─────────────────────────────────────────────────────────────────┐
│  Add GitLab Integration                                    [X]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Name *                                                         │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Company GitLab                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  GitLab URL *                                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ https://gitlab.company.com                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Access Token *                               [How to get?]     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ glpat-xxxxxxxxxxxxxxxxxxxx                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ℹ️ Token needs: api, read_repository, write_repository scopes  │
│                                                                 │
│                                    [Cancel]  [Test & Save]      │
└─────────────────────────────────────────────────────────────────┘
```

### Projects List

```
┌─────────────────────────────────────────────────────────────────┐
│  ← Company GitLab / Projects                      [+ Add]       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 📁 backend/api-service                       ● Active    │   │
│  │    Auto-review: ON • 45 reviews                          │   │
│  │    🧠 Analysis: llama-3.1-8b • 📊 Embed: nomic-embed     │   │
│  │    Filters: *.go, *.yaml, !vendor/*                      │   │
│  │                              [Configure] [View Reviews]  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 📁 frontend/web-app                          ○ Paused    │   │
│  │    Auto-review: OFF • 12 reviews                         │   │
│  │    🧠 Analysis: gpt-4 • 📊 Embed: text-embedding-3       │   │
│  │    Filters: *.ts, *.tsx, *.vue                           │   │
│  │                              [Configure] [View Reviews]  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Configure Project Modal

```
┌─────────────────────────────────────────────────────────────────┐
│  Configure Project: backend/api-service                    [X]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ☑ Enable Auto-Review                                    │   │
│  │   Automatically analyze new and updated MRs             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ═══════════════════════════════════════════════════════════   │
│  🤖 MODEL CONFIGURATION                                         │
│  ═══════════════════════════════════════════════════════════   │
│                                                                 │
│  Analysis Model (LLM for code review) *                         │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ llama-3.1-8b-instruct                              [▼]  │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ℹ️ Used for analyzing code and generating review comments      │
│  💡 Recommended: llama-3.1-70b, qwen2.5-coder, deepseek-coder   │
│                                                                 │
│  Embedding Model (for code chunking) *                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ nomic-embed-text                                   [▼]  │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ℹ️ Used to create vector embeddings for semantic code search   │
│  💡 Recommended: nomic-embed-text, mxbai-embed-large            │
│                                                                 │
│  ═══════════════════════════════════════════════════════════   │
│  📁 FILE FILTERS                                                │
│  ═══════════════════════════════════════════════════════════   │
│                                                                 │
│  Include Patterns (one per line)                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ *.go                                                    │   │
│  │ *.yaml                                                  │   │
│  │ *.sql                                                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Exclude Patterns (one per line)                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ vendor/*                                                │   │
│  │ *_test.go                                               │   │
│  │ *.generated.go                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ═══════════════════════════════════════════════════════════   │
│  📝 CUSTOM REVIEW PROMPT (optional)                             │
│  ═══════════════════════════════════════════════════════════   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Focus on security vulnerabilities and SQL injection.    │   │
│  │ Our coding standards require...                         │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ℹ️ Leave empty to use default review prompt                    │
│                                                                 │
│  ═══════════════════════════════════════════════════════════   │
│  ⚙️ ADVANCED SETTINGS                                           │
│  ═══════════════════════════════════════════════════════════   │
│                                                                 │
│  Max files to analyze     Chunk size (tokens)                   │
│  ┌───────────────────┐   ┌───────────────────┐                 │
│  │ 50                │   │ 1000              │                 │
│  └───────────────────┘   └───────────────────┘                 │
│                                                                 │
│  Context window overlap   Min confidence score                  │
│  ┌───────────────────┐   ┌───────────────────┐                 │
│  │ 100 tokens        │   │ 0.7               │                 │
│  └───────────────────┘   └───────────────────┘                 │
│                                                                 │
│                                    [Cancel]  [Save Changes]     │
└─────────────────────────────────────────────────────────────────┘
```

### MR Review Details

```
┌─────────────────────────────────────────────────────────────────┐
│  MR #142: Add user authentication                               │
│  backend/api-service • feat/auth → main                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │  Overall Score  │  │  Files Changed  │  │  Issues Found   │  │
│  │      78/100     │  │       12        │  │       5         │  │
│  │    🟡 Good      │  │  +342 / -89     │  │  2⚠️ 3ℹ️        │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│                                                                 │
│  Category Breakdown                                             │
│  ├── 🔒 Security      ████████░░  80%  (1 issue)               │
│  ├── 🐛 Bugs          █████████░  90%  (0 issues)              │
│  ├── 🎨 Code Style    ██████░░░░  60%  (3 issues)              │
│  └── ⚡ Performance   █████████░  85%  (1 issue)               │
│                                                                 │
│  ─────────────────────────────────────────────────────────────  │
│                                                                 │
│  📄 auth/handler.go                                    Score: 72│
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Line 45 ⚠️ Password stored without hashing              │   │
│  │ Line 78 ℹ️ Consider using constant-time comparison      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  📄 auth/middleware.go                                 Score: 85│
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Line 23 ℹ️ Token validation could be extracted          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│                          [View on GitLab] [Re-analyze]          │
└─────────────────────────────────────────────────────────────────┘
```

## 🤖 Recommended Models

### Analysis Models (LLM for Code Review)

| Model | Size | Best For | Notes |
|-------|------|----------|-------|
| `llama-3.1-70b-instruct` | 70B | Production, complex codebases | Best quality, slower |
| `llama-3.1-8b-instruct` | 8B | Development, quick reviews | Good balance speed/quality |
| `qwen2.5-coder-32b` | 32B | Code-specific tasks | Trained on code |
| `deepseek-coder-v2` | 16B | Multi-language projects | Strong on many languages |
| `codellama-34b` | 34B | Large codebases | Optimized for code |

### Embedding Models (for Code Chunking/Vectorization)

| Model | Dimensions | Best For | Notes |
|-------|------------|----------|-------|
| `nomic-embed-text` | 768 | General code | Good all-around |
| `mxbai-embed-large` | 1024 | High precision | Better semantic match |
| `text-embedding-3-small` | 1536 | OpenAI API | If using OpenAI |
| `bge-large-en-v1.5` | 1024 | English code | MTEB benchmark leader |
| `codebert-base` | 768 | Code-specific | Trained specifically on code |

### Model Selection Guidelines

```
┌─────────────────────────────────────────────────────────────────┐
│                    MODEL SELECTION GUIDE                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Small Team / Fast Feedback:                                    │
│  • Analysis: llama-3.1-8b-instruct                             │
│  • Embedding: nomic-embed-text                                  │
│  • ⚡ ~30 sec per MR                                            │
│                                                                 │
│  Enterprise / High Quality:                                     │
│  • Analysis: llama-3.1-70b-instruct or qwen2.5-coder-32b       │
│  • Embedding: mxbai-embed-large                                 │
│  • 🎯 ~2-5 min per MR, better issue detection                  │
│                                                                 │
│  Code-Heavy Projects:                                           │
│  • Analysis: deepseek-coder-v2 or codellama-34b                │
│  • Embedding: codebert-base                                     │
│  • 💻 Optimized for code understanding                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📝 Default Review Prompt

```markdown
You are an expert code reviewer. Analyze the following code changes and provide a structured review.

## Context
- Project: {{project_name}}
- MR Title: {{mr_title}}
- Author: {{mr_author}}
- Target Branch: {{target_branch}}

## Files Changed
{{files_list}}

## Your Task
Analyze each changed file and provide:

1. **Security Review**
   - SQL injection, XSS, CSRF vulnerabilities
   - Hardcoded secrets or credentials
   - Improper input validation
   - Authentication/authorization issues

2. **Bug Detection**
   - Logic errors
   - Null pointer dereferences
   - Resource leaks
   - Race conditions
   - Error handling issues

3. **Code Quality**
   - Code duplication
   - Complex/unreadable code
   - Missing documentation
   - Naming conventions
   - SOLID principles violations

4. **Performance**
   - N+1 queries
   - Unnecessary allocations
   - Missing caching opportunities
   - Inefficient algorithms

## Output Format
Provide your review as JSON:
{
  "summary": "Brief overall assessment",
  "overall_score": 0-100,
  "categories": [
    {"name": "security", "score": 0-100, "issues": [...]}
  ],
  "file_reviews": [
    {
      "file": "path/to/file",
      "score": 0-100,
      "issues": [
        {"line": 42, "severity": "warning", "category": "security", "message": "..."}
      ]
    }
  ],
  "suggestions": ["..."]
}
```

---

## ✅ Задачи

### Phase 1: Foundation (Database & Models)
- [ ] **GITLAB-001**: Создать миграции для таблиц `gitlab_integrations`, `gitlab_projects`, `mr_reviews`
- [ ] **GITLAB-002**: Создать Go models в `internal/models/gitlab.go`
- [ ] **GITLAB-003**: Создать repository layer `internal/storage/gitlab_repository.go`
- [ ] **GITLAB-004**: Добавить конфигурацию в `config.go` (gitlab.encryption_key, gitlab.webhook_base_url)

### Phase 2: GitLab API Client
- [ ] **GITLAB-005**: Создать `internal/gitlab/client.go` с базовыми методами
- [ ] **GITLAB-006**: Реализовать `GetProject`, `GetMergeRequest`, `GetMRChanges`
- [ ] **GITLAB-007**: Реализовать `PostMRNote`, `PostMRDiscussion` (inline comments)
- [ ] **GITLAB-008**: Реализовать `CreateWebhook`, `DeleteWebhook`
- [ ] **GITLAB-009**: Добавить retry logic и rate limiting
- [ ] **GITLAB-010**: Написать unit тесты для GitLab client

### Phase 3: Webhook Handler
- [ ] **GITLAB-011**: Создать `internal/api/handlers/gitlab_webhook.go`
- [ ] **GITLAB-012**: Реализовать signature verification (X-Gitlab-Token)
- [ ] **GITLAB-013**: Обработка событий: `merge_request` (open, update, reopen)
- [ ] **GITLAB-014**: Добавить в router: `POST /api/webhooks/gitlab/:integration_id`
- [ ] **GITLAB-015**: Создать job queue для async processing

### Phase 4: Code Chunking
- [ ] **GITLAB-016**: Создать `internal/gitlab/chunker/chunker.go` interface
- [ ] **GITLAB-017**: Реализовать `FileChunker` - чанкинг по файлам
- [ ] **GITLAB-018**: Реализовать `HunkChunker` - чанкинг по блокам diff
- [ ] **GITLAB-019**: Добавить language detection для chunks
- [ ] **GITLAB-020**: Написать unit тесты для chunkers

### Phase 5: Qdrant Integration
- [ ] **GITLAB-021**: Создать `internal/gitlab/embedding/service.go`
- [ ] **GITLAB-022**: Реализовать создание временной коллекции для MR
- [ ] **GITLAB-023**: Реализовать embedding и upsert chunks
- [ ] **GITLAB-024**: Реализовать semantic search по чанкам
- [ ] **GITLAB-025**: Реализовать cleanup коллекции после анализа
- [ ] **GITLAB-026**: Добавить TTL для автоматической очистки старых коллекций

### Phase 6: AI Analysis Pipeline
- [ ] **GITLAB-027**: Создать `internal/gitlab/analyzer/analyzer.go`
- [ ] **GITLAB-028**: Реализовать структурированный промпт для code review
- [ ] **GITLAB-029**: Реализовать парсинг JSON response от LLM
- [ ] **GITLAB-030**: Реализовать per-file analysis с контекстом из Qdrant
- [ ] **GITLAB-031**: Реализовать aggregation результатов
- [ ] **GITLAB-032**: Написать unit тесты для analyzer

### Phase 7: Comment Builder
- [ ] **GITLAB-033**: Создать `internal/gitlab/comment/builder.go`
- [ ] **GITLAB-034**: Реализовать форматирование результата в Markdown
- [ ] **GITLAB-035**: Реализовать inline comments для конкретных строк
- [ ] **GITLAB-036**: Добавить collapsible sections для длинных ревью
- [ ] **GITLAB-037**: Добавить emoji и визуальные индикаторы

### Phase 8: Admin API
- [ ] **GITLAB-038**: Создать `internal/api/handlers/gitlab_admin.go`
- [ ] **GITLAB-039**: CRUD для integrations
- [ ] **GITLAB-040**: CRUD для projects
- [ ] **GITLAB-041**: Endpoint для test connection
- [ ] **GITLAB-042**: Endpoint для manual webhook setup
- [ ] **GITLAB-043**: Endpoints для reviews (list, details, retry)
- [ ] **GITLAB-044**: Добавить роуты в router.go

### Phase 9: Admin UI (Svelte)
- [ ] **GITLAB-045**: Создать страницу `/admin/gitlab` - список интеграций
- [ ] **GITLAB-046**: Создать модалку добавления/редактирования интеграции
- [ ] **GITLAB-047**: Создать страницу `/admin/gitlab/:id/projects` - список проектов
- [ ] **GITLAB-048**: Создать модалку настройки проекта:
  - [ ] **GITLAB-048a**: Выпадающий список Analysis Model (LLM для ревью)
  - [ ] **GITLAB-048b**: Выпадающий список Embedding Model (для чанкинизации)
  - [ ] **GITLAB-048c**: Поля Include/Exclude patterns
  - [ ] **GITLAB-048d**: Textarea для custom prompt
  - [ ] **GITLAB-048e**: Advanced settings (chunk size, overlap, max files)
- [ ] **GITLAB-049**: Создать страницу `/admin/gitlab/reviews` - список ревью
- [ ] **GITLAB-050**: Создать страницу `/admin/gitlab/reviews/:id` - детали ревью
- [ ] **GITLAB-051**: Добавить в sidebar меню "GitLab" (admin only)
- [ ] **GITLAB-052**: API endpoint для получения списка доступных моделей (LLM + Embedding)

### Phase 10: Background Processing
- [ ] **GITLAB-053**: Создать `internal/gitlab/worker/worker.go`
- [ ] **GITLAB-054**: Реализовать job queue (in-memory или Redis)
- [ ] **GITLAB-055**: Реализовать graceful shutdown
- [ ] **GITLAB-056**: Добавить retry logic для failed jobs
- [ ] **GITLAB-057**: Добавить metrics и logging

### Phase 11: Testing & Documentation
- [ ] **GITLAB-058**: Integration тесты с mock GitLab server
- [ ] **GITLAB-059**: E2E тест полного flow: webhook → analysis → comment
- [ ] **GITLAB-060**: Документация API endpoints
- [ ] **GITLAB-061**: Документация по настройке GitLab webhook
- [ ] **GITLAB-062**: README с примерами использования

### Phase 12: Enhancements (Future)
- [ ] **GITLAB-063**: Поддержка GitHub (дополнительно к GitLab)
- [ ] **GITLAB-064**: Поддержка Bitbucket
- [ ] **GITLAB-065**: Custom prompts per language
- [ ] **GITLAB-066**: Integration с Slack/Teams для нотификаций
- [ ] **GITLAB-067**: Статистика и аналитика по ревью
- [ ] **GITLAB-068**: Обучение на feedback (approve/reject комментариев)
- [ ] **GITLAB-069**: Auto-suggest optimal models based on codebase language

---

## 🔐 Security Considerations

1. **Token Storage** - Access tokens шифруются в БД (AES-256-GCM)
2. **Webhook Verification** - Проверка X-Gitlab-Token header
3. **Rate Limiting** - Защита от webhook flood
4. **Audit Log** - Логирование всех действий
5. **Scope Validation** - Проверка минимальных scopes токена

## 📊 Metrics

- `gitlab_webhooks_received_total` - Всего полученных webhooks
- `gitlab_reviews_completed_total` - Завершённых ревью
- `gitlab_reviews_failed_total` - Неудачных ревью
- `gitlab_review_duration_seconds` - Время анализа
- `gitlab_tokens_used_total` - Использованных токенов LLM

---

## 🚀 Quick Start (после реализации)

1. **Добавить интеграцию** в Admin → GitLab → Add Integration
2. **Создать Access Token** в GitLab: Settings → Access Tokens (scopes: api, read_repository)
3. **Добавить проект** и настроить webhook
4. **Создать MR** - автоматически получите AI review!

---

*Документ создан: 2025-12-04*
*Версия: 1.0.0*

