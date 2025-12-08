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

# MR Reviews (with pagination)
GET    /api/admin/gitlab/reviews                   # List all reviews
GET    /api/admin/gitlab/projects/:id/reviews      # List project reviews
GET    /api/admin/gitlab/reviews/:id               # Get review details
POST   /api/admin/gitlab/reviews/:id/retry         # Retry failed review

# Queue Management
GET    /api/admin/gitlab/queue/status              # Queue stats & worker status
GET    /api/admin/gitlab/queue/jobs                # List pending/processing jobs
POST   /api/admin/gitlab/queue/jobs/:id/cancel     # Cancel pending job
POST   /api/admin/gitlab/queue/jobs/:id/retry      # Retry failed job
DELETE /api/admin/gitlab/queue/jobs/:id            # Delete job

# Webhook (public, verified by secret)
POST   /api/webhooks/gitlab/:integration_id        # Receive GitLab webhooks
```

### API Pagination

All list endpoints support pagination:

```
GET /api/admin/gitlab/reviews?page=1&limit=20&sort=created_at&order=desc

Response:
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 156,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  }
}
```

Query parameters:
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)
- `sort` - Sort field (default: created_at)
- `order` - Sort order: asc/desc (default: desc)
- `status` - Filter by status (for reviews/jobs)
- `project_id` - Filter by project

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

// ⚠️ ВАЖНО: Очистка перед новым анализом
func (s *Service) prepareCollection(ctx context.Context, collectionName string) error {
    // 1. Проверяем существует ли коллекция
    exists, err := s.qdrant.CollectionExists(ctx, collectionName)
    if err != nil {
        return err
    }
    
    // 2. Если существует - удаляем (новый анализ = чистые данные)
    if exists {
        s.logger.Info("Clearing existing collection for re-analysis", 
            "collection", collectionName)
        if err := s.qdrant.DeleteCollection(ctx, collectionName); err != nil {
            return fmt.Errorf("failed to clear collection: %w", err)
        }
    }
    
    // 3. Создаём новую коллекцию
    return s.qdrant.CreateCollection(ctx, collectionName, s.embeddingDimensions)
}

// Cleanup after analysis (или по TTL)
func (s *Service) cleanupMRCollection(collectionName string) error
```

### 7. Webhook Deduplication & Rate Limiting

GitLab может отправлять несколько webhook'ов подряд для одного MR:
- Push нескольких коммитов
- Обновление MR (rebase, force push)
- Изменение описания/labels
- CI pipeline events

```go
// Дедупликация через in-memory cache с debounce
// Реализовано в internal/gitlab/webhook/handler.go
type WebhookDeduplicator struct {
    cache      sync.Map              // In-memory cache для debounce
    debounce   time.Duration         // Время ожидания перед обработкой (e.g., 10s)
    processing sync.Map              // Активные обработки: key -> cancelFunc
}

// Ключ для дедупликации
func dedupKey(projectID int, mrIID int) string {
    return fmt.Sprintf("gitlab:mr:processing:%d:%d", projectID, mrIID)
}

// HandleWebhook с дедупликацией и debounce
func (d *WebhookDeduplicator) HandleWebhook(ctx context.Context, event MREvent) error {
    key := dedupKey(event.Project.ID, event.MergeRequest.IID)
    
    // 1. Проверяем, не обрабатывается ли уже этот MR
    if _, processing := d.processing.Load(key); processing {
        d.logger.Debug("MR already being processed, skipping duplicate webhook",
            "project_id", event.Project.ID,
            "mr_iid", event.MergeRequest.IID)
        return nil // Игнорируем дубликат
    }
    
    // 2. Проверяем debounce - был ли недавно webhook для этого MR
    lastWebhook, exists := d.cache.Get(ctx, key)
    if exists {
        // Отменяем предыдущий отложенный анализ
        if cancel, ok := d.processing.Load(key + ":cancel"); ok {
            cancel.(context.CancelFunc)()
        }
    }
    
    // 3. Сохраняем timestamp последнего webhook
    d.cache.Set(ctx, key, time.Now(), d.debounce*2)
    
    // 4. Запускаем отложенную обработку (debounce)
    ctx, cancel := context.WithCancel(ctx)
    d.processing.Store(key+":cancel", cancel)
    
    go func() {
        select {
        case <-time.After(d.debounce):
            // Debounce прошёл, начинаем анализ
            d.processing.Store(key, true)
            defer d.processing.Delete(key)
            defer d.processing.Delete(key + ":cancel")
            
            if err := d.processReview(ctx, event); err != nil {
                d.logger.Error("MR review failed", "error", err)
            }
            
        case <-ctx.Done():
            // Отменено из-за нового webhook
            d.logger.Debug("Review cancelled due to newer webhook")
        }
    }()
    
    return nil
}

// Статусы обработки для предотвращения дубликатов
type ReviewStatus string
const (
    ReviewStatusPending    ReviewStatus = "pending"     // В очереди
    ReviewStatusDebouncing ReviewStatus = "debouncing"  // Ожидает debounce
    ReviewStatusAnalyzing  ReviewStatus = "analyzing"   // Идёт анализ
    ReviewStatusCompleted  ReviewStatus = "completed"   // Завершён
    ReviewStatusFailed     ReviewStatus = "failed"      // Ошибка
)

// Проверка перед началом анализа
func (s *Service) canStartReview(ctx context.Context, projectID, mrIID int) (bool, string) {
    key := dedupKey(projectID, mrIID)
    
    status, exists := s.cache.Get(ctx, key+":status")
    if !exists {
        return true, ""
    }
    
    switch ReviewStatus(status.(string)) {
    case ReviewStatusAnalyzing:
        return false, "Analysis already in progress"
    case ReviewStatusDebouncing:
        return false, "Waiting for debounce"
    default:
        return true, ""
    }
}
```

### Webhook Flow с Deduplication

```
┌─────────────────────────────────────────────────────────────────┐
│                     WEBHOOK PROCESSING FLOW                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Webhook #1 (push commit A)                                     │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────┐                                                │
│  │  Debounce   │◄─── Start 10s timer                            │
│  │   Timer     │                                                │
│  └──────┬──────┘                                                │
│         │                                                       │
│         │  Webhook #2 (push commit B) - 3s later                │
│         │       │                                               │
│         │       ▼                                               │
│         │  ┌─────────────┐                                      │
│         └──│  Cancel #1  │                                      │
│            │  Restart    │◄─── Reset timer to 10s               │
│            └──────┬──────┘                                      │
│                   │                                             │
│                   │  Webhook #3 (update title) - 2s later       │
│                   │       │                                     │
│                   │       ▼                                     │
│                   │  ┌─────────────┐                            │
│                   └──│  Cancel #2  │                            │
│                      │  Restart    │◄─── Reset timer to 10s     │
│                      └──────┬──────┘                            │
│                             │                                   │
│                             │  ... 10 seconds pass ...          │
│                             │                                   │
│                             ▼                                   │
│                      ┌─────────────┐                            │
│                      │   ENQUEUE   │◄─── Add to analysis queue  │
│                      │   JOB       │     (only ONE job per MR)  │
│                      └──────┬──────┘                            │
│                             │                                   │
│                             ▼                                   │
│               ┌─────────────────────────────┐                   │
│               │      ANALYSIS QUEUE         │                   │
│               │       (PostgreSQL)          │                   │
│               │                             │                   │
│               │  ┌─────────────────────┐    │                   │
│               │  │ Workers pick jobs   │    │                   │
│               │  │ and process them    │    │                   │
│               │  └─────────────────────┘    │                   │
│               └─────────────────────────────┘                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Complete Flow: Webhook → Debounce → Queue → Worker

```
┌─────────────────────────────────────────────────────────────────┐
│                    COMPLETE PROCESSING FLOW                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. WEBHOOK RECEIVED                                            │
│     │                                                           │
│     ▼                                                           │
│  ┌─────────────────┐                                            │
│  │ Signature Check │──── Invalid ──► 401 Unauthorized           │
│  └────────┬────────┘                                            │
│           │ Valid                                               │
│           ▼                                                     │
│  2. DEDUPLICATION CHECK                                         │
│     │                                                           │
│     ├── MR already in queue? ──► Cancel old job, continue       │
│     │                                                           │
│     ▼                                                           │
│  3. DEBOUNCE (10s)                                              │
│     │                                                           │
│     ├── New webhook for same MR? ──► Reset timer, go to step 2  │
│     │                                                           │
│     ▼ (timer expires)                                           │
│  4. CREATE JOB                                                  │
│     │                                                           │
│     │  AnalysisJob {                                            │
│     │    project_id, mr_iid,                                    │
│     │    embedding_model, analysis_model,                       │
│     │    priority, status: "pending"                            │
│     │  }                                                        │
│     │                                                           │
│     ▼                                                           │
│  5. ENQUEUE (PostgreSQL)                                        │
│     │                                                           │
│     │  queue.Enqueue(ctx, job)                                  │
│     │                                                           │
│     ▼                                                           │
│  6. WORKER PICKS JOB                                            │
│     │                                                           │
│     │  job = queue.Dequeue(ctx)                                 │
│     │  job.status = "processing"                                │
│     │  job.worker_id = "worker-1"                               │
│     │                                                           │
│     ▼                                                           │
│  7. ANALYSIS PIPELINE                                           │
│     │                                                           │
│     ├── Fetch MR changes from GitLab                            │
│     ├── Chunk code                                              │
│     ├── Clear & create Qdrant collection                        │
│     ├── Embed chunks                                            │
│     ├── Run LLM analysis                                        │
│     ├── Build markdown comment                                  │
│     └── Post to GitLab MR                                       │
│           │                                                     │
│           ▼                                                     │
│  8. COMPLETE                                                    │
│     │                                                           │
│     ├── Success: job.status = "completed"                       │
│     │            cleanup Qdrant collection                      │
│     │                                                           │
│     └── Failure: job.retry_count++                              │
│                  if retry_count < max_retries:                  │
│                      job.status = "pending" (retry)             │
│                  else:                                          │
│                      job.status = "failed"                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 8. Analysis Queue Architecture

Для обработки множества MR из разных проектов параллельно:

```
┌─────────────────────────────────────────────────────────────────┐
│                    ANALYSIS QUEUE ARCHITECTURE                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Webhooks from different projects:                              │
│                                                                 │
│  Project A (MR #1) ──┐                                          │
│  Project A (MR #2) ──┤                                          │
│  Project B (MR #5) ──┼──►  ┌─────────────────────────────────┐  │
│  Project C (MR #3) ──┤     │                                 │  │
│  Project B (MR #6) ──┘     │     ANALYSIS JOB QUEUE          │  │
│                            │        (PostgreSQL)             │  │
│                            │                                 │  │
│  After deduplication:      │  ┌───┬───┬───┬───┬───┬───┐     │  │
│                            │  │ 1 │ 2 │ 5 │ 3 │ 6 │...│     │  │
│                            │  └───┴───┴───┴───┴───┴───┘     │  │
│                            │     ↑                           │  │
│                            │     Priority queue (FIFO)       │  │
│                            └─────────────────────────────────┘  │
│                                          │                      │
│                                          │                      │
│                            ┌─────────────┴─────────────┐        │
│                            │                           │        │
│                            ▼                           ▼        │
│                   ┌─────────────┐             ┌─────────────┐   │
│                   │  Worker 1   │             │  Worker 2   │   │
│                   │  (busy)     │             │  (idle)     │   │
│                   │  MR #1      │             │  → takes    │   │
│                   │  Project A  │             │    MR #2    │   │
│                   └─────────────┘             └─────────────┘   │
│                                                                 │
│  Worker Pool: Configurable (default: 3 concurrent workers)      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

```go
// AnalysisJob представляет задачу на анализ MR
type AnalysisJob struct {
    ID            string           `json:"id"`
    IntegrationID string           `json:"integration_id"`
    ProjectID     int              `json:"project_id"`
    MRIID         int              `json:"mr_iid"`
    MRTitle       string           `json:"mr_title"`
    SourceBranch  string           `json:"source_branch"`
    TargetBranch  string           `json:"target_branch"`
    AuthorName    string           `json:"author_name"`
    
    // Configuration (из GitLabProject)
    EmbeddingModel string          `json:"embedding_model"`
    AnalysisModel  string          `json:"analysis_model"`
    ReviewPrompt   string          `json:"review_prompt,omitempty"`
    FileFilters    []string        `json:"file_filters,omitempty"`
    
    // Queue metadata
    Status         JobStatus       `json:"status"`
    Priority       int             `json:"priority"`      // Higher = more urgent
    CreatedAt      time.Time       `json:"created_at"`
    StartedAt      *time.Time      `json:"started_at,omitempty"`
    CompletedAt    *time.Time      `json:"completed_at,omitempty"`
    WorkerID       string          `json:"worker_id,omitempty"`
    RetryCount     int             `json:"retry_count"`
    MaxRetries     int             `json:"max_retries"`
    Error          string          `json:"error,omitempty"`
}

type JobStatus string
const (
    JobStatusPending    JobStatus = "pending"
    JobStatusProcessing JobStatus = "processing"
    JobStatusCompleted  JobStatus = "completed"
    JobStatusFailed     JobStatus = "failed"
    JobStatusCancelled  JobStatus = "cancelled"
)

// AnalysisQueue интерфейс очереди
type AnalysisQueue interface {
    // Enqueue добавляет job в очередь
    Enqueue(ctx context.Context, job *AnalysisJob) error
    
    // Dequeue получает следующий job (блокирующий)
    Dequeue(ctx context.Context) (*AnalysisJob, error)
    
    // DequeueNonBlocking получает job без ожидания
    DequeueNonBlocking(ctx context.Context) (*AnalysisJob, error)
    
    // UpdateStatus обновляет статус job
    UpdateStatus(ctx context.Context, jobID string, status JobStatus, err error) error
    
    // GetJobsByMR получает все jobs для конкретного MR
    GetJobsByMR(ctx context.Context, projectID, mrIID int) ([]*AnalysisJob, error)
    
    // GetPendingCount возвращает количество pending jobs
    GetPendingCount(ctx context.Context) (int64, error)
    
    // CancelJobsForMR отменяет pending jobs для MR (при новом webhook)
    CancelJobsForMR(ctx context.Context, projectID, mrIID int) error
    
    // CleanupOld удаляет старые completed/failed jobs
    CleanupOld(ctx context.Context, olderThan time.Duration) error
}

// WorkerPool управляет воркерами
type WorkerPool struct {
    queue       AnalysisQueue
    analyzer    *Analyzer
    workers     int
    wg          sync.WaitGroup
    ctx         context.Context
    cancel      context.CancelFunc
    logger      *logrus.Logger
}

// NewWorkerPool создаёт пул воркеров
func NewWorkerPool(queue AnalysisQueue, analyzer *Analyzer, workers int, logger *logrus.Logger) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    return &WorkerPool{
        queue:    queue,
        analyzer: analyzer,
        workers:  workers,
        ctx:      ctx,
        cancel:   cancel,
        logger:   logger,
    }
}

// Start запускает воркеры
func (p *WorkerPool) Start() {
    p.logger.WithField("workers", p.workers).Info("Starting analysis worker pool")
    
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.runWorker(fmt.Sprintf("worker-%d", i))
    }
}

// Stop gracefully останавливает воркеры
func (p *WorkerPool) Stop() {
    p.logger.Info("Stopping analysis worker pool")
    p.cancel()
    p.wg.Wait()
    p.logger.Info("All workers stopped")
}

// runWorker основной цикл воркера
func (p *WorkerPool) runWorker(workerID string) {
    defer p.wg.Done()
    
    logger := p.logger.WithField("worker_id", workerID)
    logger.Info("Worker started")
    
    for {
        select {
        case <-p.ctx.Done():
            logger.Info("Worker shutting down")
            return
        default:
            // Получаем следующую задачу
            job, err := p.queue.Dequeue(p.ctx)
            if err != nil {
                if p.ctx.Err() != nil {
                    return // Context cancelled
                }
                logger.WithError(err).Error("Failed to dequeue job")
                time.Sleep(time.Second) // Backoff on error
                continue
            }
            
            if job == nil {
                time.Sleep(100 * time.Millisecond) // No jobs, wait a bit
                continue
            }
            
            // Обрабатываем задачу
            p.processJob(logger, job)
        }
    }
}

// processJob обрабатывает одну задачу
func (p *WorkerPool) processJob(logger *logrus.Entry, job *AnalysisJob) {
    logger = logger.WithFields(logrus.Fields{
        "job_id":     job.ID,
        "project_id": job.ProjectID,
        "mr_iid":     job.MRIID,
    })
    
    logger.Info("Processing analysis job")
    startTime := time.Now()
    
    // Обновляем статус на processing
    job.Status = JobStatusProcessing
    now := time.Now()
    job.StartedAt = &now
    if err := p.queue.UpdateStatus(p.ctx, job.ID, JobStatusProcessing, nil); err != nil {
        logger.WithError(err).Error("Failed to update job status")
    }
    
    // Выполняем анализ
    err := p.analyzer.AnalyzeMR(p.ctx, job)
    
    duration := time.Since(startTime)
    completedAt := time.Now()
    job.CompletedAt = &completedAt
    
    if err != nil {
        job.Error = err.Error()
        job.RetryCount++
        
        if job.RetryCount < job.MaxRetries {
            // Retry
            logger.WithError(err).WithField("retry", job.RetryCount).Warn("Job failed, will retry")
            job.Status = JobStatusPending
            p.queue.UpdateStatus(p.ctx, job.ID, JobStatusPending, err)
        } else {
            // Max retries exceeded
            logger.WithError(err).Error("Job failed permanently")
            job.Status = JobStatusFailed
            p.queue.UpdateStatus(p.ctx, job.ID, JobStatusFailed, err)
        }
    } else {
        logger.WithField("duration", duration).Info("Job completed successfully")
        job.Status = JobStatusCompleted
        p.queue.UpdateStatus(p.ctx, job.ID, JobStatusCompleted, nil)
    }
}
```

### Queue Storage: PostgreSQL ✅ IMPLEMENTED

```go
// PostgreSQL Queue - основная реализация в internal/gitlab/storage/postgres_review.go
type PostgresStore struct {
    db *sql.DB
}

// Job management methods:
// - CreateJob(ctx, job) - создание задачи
// - GetJob(ctx, id) - получение задачи по ID  
// - GetNextPendingJob(ctx) - получение следующей задачи (FOR UPDATE SKIP LOCKED)
// - ClaimJob(ctx, jobID, workerID) - захват задачи воркером
// - CompleteJob(ctx, jobID) - завершение задачи
// - FailJob(ctx, jobID, error) - пометка ошибки
// - RetryJob(ctx, jobID, nextRetryAt) - повтор задачи
// - CancelJob(ctx, jobID) - отмена задачи
// - GetQueueStats(ctx) - статистика очереди
// - CleanupOldJobs(ctx, olderThanDays) - очистка старых задач
```

**Особенности PostgreSQL реализации:**
- `FOR UPDATE SKIP LOCKED` для concurrent доступа воркеров
- JSONB для хранения конфигурации и результатов
- Индексы на status + priority для быстрого dequeue
- Cascade delete при удалении интеграции/проекта

### Queue Configuration

```yaml
# config.yaml
gitlab:
  enabled: true
  workers: 3              # Concurrent analysis workers
  max_retries: 3          # Max retry attempts per job
  job_timeout: "30m"      # Max time per job
  cleanup_interval: "1h"
  cleanup_older_than: "7d"
  debounce_delay: "10s"   # Debounce для webhook'ов
  
  analysis:
    default_priority: 0
    urgent_priority: 10   # For force push to main branch
```

**Note:** Очередь использует PostgreSQL таблицу `gitlab_analysis_jobs`. Отдельный Redis не требуется.

### Queue Metrics

```go
// Prometheus metrics
var (
    queueSize = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "gitlab_analysis_queue_size",
        Help: "Current number of pending jobs in queue",
    })
    
    jobsProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "gitlab_analysis_jobs_processed_total",
        Help: "Total number of processed jobs",
    }, []string{"status"}) // completed, failed, cancelled
    
    jobDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "gitlab_analysis_job_duration_seconds",
        Help:    "Time spent processing a job",
        Buckets: []float64{10, 30, 60, 120, 300, 600, 1800},
    })
    
    activeWorkers = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "gitlab_analysis_active_workers",
        Help: "Number of workers currently processing jobs",
    })
)
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

### Queue Monitor (Admin Dashboard)

```
┌─────────────────────────────────────────────────────────────────┐
│  GitLab Analysis Queue                            [⟳ Refresh]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Workers Status                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  Worker 1   │  │  Worker 2   │  │  Worker 3   │              │
│  │  🟢 Active  │  │  🟢 Active  │  │  ⚪ Idle    │              │
│  │  MR #142    │  │  MR #89     │  │             │              │
│  │  2m 15s     │  │  45s        │  │             │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│                                                                 │
│  Queue Statistics                                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  📊 Pending: 5  │  ⚙️ Processing: 2  │  ✅ Today: 47     │  │
│  │  ❌ Failed: 1   │  ⏱️ Avg time: 2m   │  📈 Total: 1,234  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Pending Jobs                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ #  │ Project           │ MR    │ Priority │ Queued     │  │  │
│  ├────┼───────────────────┼───────┼──────────┼────────────┤  │  │
│  │ 1  │ backend/api       │ #156  │ 🔴 High  │ 30s ago    │  │  │
│  │ 2  │ frontend/web      │ #89   │ ⚪ Normal │ 1m ago     │  │  │
│  │ 3  │ backend/api       │ #155  │ ⚪ Normal │ 2m ago     │  │  │
│  │ 4  │ infra/terraform   │ #23   │ ⚪ Normal │ 3m ago     │  │  │
│  │ 5  │ mobile/ios        │ #67   │ ⚪ Normal │ 5m ago     │  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Failed Jobs (retry exhausted)                    [Clear All]   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ backend/api #148 │ "Timeout after 30m" │ [Retry] [Delete] │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
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

### Phase 1: Foundation (Database & Models) ✅ COMPLETED
- [x] **GITLAB-001**: Создать миграции для таблиц `gitlab_integrations`, `gitlab_projects`, `mr_reviews`
- [x] **GITLAB-002**: Создать Go models в `internal/models/gitlab.go`
- [x] **GITLAB-003**: Создать repository layer `internal/gitlab/storage/postgres.go`
- [x] **GITLAB-004**: Добавить конфигурацию в `config.go` (gitlab.encryption_key, gitlab.webhook_base_url)

### Phase 2: GitLab API Client ✅ COMPLETED
- [x] **GITLAB-005**: Создать `internal/gitlab/client/client.go` с базовыми методами
- [x] **GITLAB-006**: Реализовать `GetProject`, `GetMergeRequest`, `GetMRChanges`
- [x] **GITLAB-007**: Реализовать `PostMRNote`, `PostMRDiscussion` (inline comments)
- [x] **GITLAB-008**: Реализовать `CreateWebhook`, `DeleteWebhook`
- [x] **GITLAB-009**: Добавить retry logic и rate limiting
- [x] **GITLAB-010**: Написать unit тесты для GitLab client
  - `internal/gitlab/client/client_test.go` - 10 тест-функций, 20+ тест-кейсов

### Phase 3: Storage Layer ✅ COMPLETED
- [x] **GITLAB-011**: Создать `internal/gitlab/storage/interface.go` - интерфейсы хранилища
- [x] **GITLAB-012**: Создать `internal/gitlab/storage/postgres.go` - PostgreSQL реализация
- [x] **GITLAB-013**: Создать `internal/gitlab/storage/postgres_review.go` - reviews, jobs, events
- [x] **GITLAB-014**: CRUD для integrations, projects, reviews
- [x] **GITLAB-015**: Model usage tracking (FindProjectsByModel, IsModelUsed)

### Phase 4: Webhook Handler ✅ COMPLETED
- [x] **GITLAB-016**: Создать `internal/gitlab/webhook/handler.go`
- [x] **GITLAB-017**: Реализовать signature verification (X-Gitlab-Token)
- [x] **GITLAB-018**: Обработка событий: `merge_request` (open, update, reopen)
- [x] **GITLAB-019**: Debounce timer для группировки webhook'ов
- [x] **GITLAB-020**: Защита от concurrent analysis одного MR

### Phase 5: Code Analysis Service ✅ COMPLETED
- [x] **GITLAB-021**: Создать `internal/gitlab/analyzer/analyzer.go`
- [x] **GITLAB-022**: Реализовать структурированный промпт для code review (`prompt.go`)
- [x] **GITLAB-023**: Реализовать парсинг JSON response от LLM (`parser.go`)
- [x] **GITLAB-024**: Реализовать per-file analysis (analyzeLargeMR)
- [x] **GITLAB-025**: Реализовать aggregation результатов (aggregateResults)
- [x] **GITLAB-026**: Типы для code embeddings (`types.go`)
- [x] **GITLAB-027**: ⚠️ Qdrant integration для context retrieval (опционально)
  - `internal/gitlab/rag/qdrant.go` - QdrantClient для vector database
  - `internal/gitlab/rag/service.go` - RAGService для code context retrieval

### Phase 6: Code Chunking ✅ COMPLETED
- [x] **GITLAB-028**: Создать `internal/gitlab/chunker/chunker.go` interface
- [x] **GITLAB-029**: Реализовать `FileChunker` - чанкинг по файлам (chunkByFile)
- [x] **GITLAB-030**: Реализовать `HunkChunker` - чанкинг по блокам diff (chunkByHunk)
- [x] **GITLAB-031**: Добавить language detection для chunks (DetectLanguage)
- [x] **GITLAB-032**: Реализовать `FunctionChunker` - чанкинг по функциям (chunkByFunction)

### Phase 7: Comment Builder ✅ COMPLETED
- [x] **GITLAB-039**: Создать `internal/gitlab/comment/builder.go`
- [x] **GITLAB-040**: Реализовать форматирование результата в Markdown
- [x] **GITLAB-041**: Реализовать inline comments для конкретных строк
- [x] **GITLAB-042**: Добавить collapsible sections для длинных ревью
- [x] **GITLAB-043**: Добавить emoji и визуальные индикаторы
- [x] **GITLAB-044**: ⚠️ Обработка лимитов GitLab (split long comments) - `internal/gitlab/comment/splitter.go`

### Phase 8: Admin API ✅ COMPLETED
- [x] **GITLAB-045**: Создать `internal/api/handlers/gitlab_admin.go`
- [x] **GITLAB-046**: CRUD для integrations
- [x] **GITLAB-047**: CRUD для projects
- [x] **GITLAB-048**: Endpoint для test connection
- [x] **GITLAB-049**: Endpoint для manual webhook setup
- [x] **GITLAB-050**: Endpoints для reviews (list, details, retry)
- [x] **GITLAB-051**: Добавить роуты в `internal/api/router/gitlab_routes.go`

### Phase 9: Admin UI (Svelte) ✅ COMPLETED
- [x] **GITLAB-052**: Создать страницу `/admin/gitlab` - список интеграций
- [x] **GITLAB-053**: Создать модалку добавления/редактирования интеграции
- [x] **GITLAB-054**: Создать страницу `/admin/gitlab/:id/projects` - список проектов
- [x] **GITLAB-055**: Создать модалку настройки проекта:
  - [x] **GITLAB-055a**: Выпадающий список Analysis Model (LLM для ревью)
  - [x] **GITLAB-055b**: Выпадающий список Embedding Model (для чанкинизации)
  - [x] **GITLAB-055c**: Поля Include/Exclude patterns
    - UI: textarea с glob patterns (один на строку)
    - API: передаётся как `settings.include_patterns` / `settings.exclude_patterns`
  - [x] **GITLAB-055d**: Textarea для custom prompt
  - [x] **GITLAB-055e**: Advanced settings (chunk size, overlap, max files)
    - UI: collapsible "Advanced Settings" секция
    - Поля: chunk_size, chunk_overlap, max_files_per_mr, max_lines_per_file, skip_draft_mrs, skip_bots
- [x] **GITLAB-056**: Создать страницу `/admin/gitlab/reviews` - список ревью с пагинацией
- [x] **GITLAB-057**: Создать страницу `/admin/gitlab/reviews/:id` - детали ревью (modal)
- [x] **GITLAB-058**: Добавить в sidebar меню "GitLab" (admin only)
- [x] **GITLAB-059**: API endpoint для получения списка доступных моделей (только active!)
  - `GET /api/admin/gitlab/models` - все активные модели
  - `GET /api/admin/gitlab/models/analysis` - модели с capability chat
  - `GET /api/admin/gitlab/models/embedding` - модели с capability embedding
- [x] **GITLAB-059a**: ⚠️ Блокировка деактивации модели используемой в GitLab
  - Проверка в `UpdateModel` и `DeleteModel` в registry_handler.go
  - Возвращает 409 Conflict с информацией о проектах использующих модель

### Phase 9.1: Queue Monitor UI ✅ COMPLETED
- [x] **GITLAB-060**: Создать компонент `/admin/gitlab/queue` - мониторинг очереди
- [x] **GITLAB-061**: Отображение статуса воркеров (active/idle, current job, duration)
- [x] **GITLAB-062**: Отображение статистики очереди (pending, processing, completed, failed)
- [x] **GITLAB-063**: Таблица pending jobs с возможностью отмены
- [x] **GITLAB-064**: Таблица failed jobs с retry/delete
- [x] **GITLAB-065**: Auto-refresh каждые 5 секунд

### Phase 10: Analysis Queue & Workers ✅ COMPLETED
- [x] **GITLAB-066**: Создать `internal/gitlab/storage/interface.go` - Store interfaces (включая Queue)
- [x] **GITLAB-067**: PostgreSQL queue implementation в `postgres_review.go`
- [x] **GITLAB-068**: Job management (CreateJob, GetJob, ClaimJob, CompleteJob, FailJob)
- [x] **GITLAB-069**: Queue stats (GetQueueStats, GetNextPendingJob)
- [x] **GITLAB-070**: Создать `internal/gitlab/worker/pool.go` - WorkerPool
- [x] **GITLAB-071**: Реализовать configurable worker count (default: 3)
- [x] **GITLAB-072**: Реализовать graceful shutdown с drain queue
- [x] **GITLAB-073**: Реализовать retry logic с exponential backoff
- [x] **GITLAB-074**: Добавить job priority (urgent for main branch)
- [x] **GITLAB-075**: Добавить job timeout handling
- [x] **GITLAB-076**: Добавить queue cleanup (старые completed/failed jobs)
- [x] **GITLAB-077**: Добавить Prometheus metrics для очереди
  - `internal/gitlab/metrics/prometheus.go` - полный набор метрик

### Phase 11: Testing & Documentation
- [x] **GITLAB-078**: Integration тесты с mock GitLab server
  - `internal/gitlab/testing/mock_server.go` - Mock GitLab API server
  - `internal/gitlab/testing/integration_test.go` - Integration tests
- [x] **GITLAB-079**: E2E тест полного flow: webhook → queue → analysis → comment
  - `internal/gitlab/testing/e2e_test.go` - E2E tests
- [x] **GITLAB-080**: Load test очереди (100+ concurrent webhooks)
  - `internal/gitlab/testing/load_test.go` - Load tests (skip with -short)
- [x] **GITLAB-081**: Тест split комментариев при превышении лимита
  - `internal/gitlab/comment/splitter_test.go` - Splitter unit tests
  - Исправлен баг с обновлением footer при динамическом количестве частей
- [x] **GITLAB-082**: Документация API endpoints (с пагинацией)
  - `docs/gitlab-api.md` - полная документация API с примерами
- [x] **GITLAB-083**: Документация по настройке GitLab webhook
  - `docs/gitlab-webhook-setup.md` - пошаговая инструкция
- [x] **GITLAB-084**: README с примерами использования
  - `docs/gitlab-readme.md` - обзор, quick start, примеры

### Phase 12: Enhancements (Future)
- [x] **GITLAB-085**: Поддержка GitHub (дополнительно к GitLab)
  - `internal/git/provider/interface.go` - Unified provider interface
  - `internal/git/provider/github/client.go` - GitHub API client
  - `internal/git/provider/github/types.go` - GitHub types
- [x] **GITLAB-086**: Поддержка Bitbucket
  - `internal/git/provider/bitbucket/client.go` - Bitbucket API client
  - `internal/git/provider/bitbucket/types.go` - Bitbucket types
- [x] **GITLAB-087**: Custom prompts per language
  - `internal/gitlab/analyzer/language_prompts.go` - Language-specific prompts
  - Поддержка: Go, TypeScript, JavaScript, Python, Rust, Java, C/C++, SQL, Shell, YAML, Dockerfile, Terraform
- [x] **GITLAB-088**: Integration с Telegram для нотификаций
  - `internal/notifications/telegram/client.go` - Telegram Bot API client
  - `internal/notifications/telegram/notifier.go` - Review notifier service
- [x] **GITLAB-089**: Статистика и аналитика по ревью
  - `internal/gitlab/analytics/service.go` - Analytics service with dashboard
  - Metrics: по проектам, моделям, пользователям, категориям, trends
- [x] **GITLAB-090**: Обучение на feedback (approve/reject комментариев)
  - `internal/gitlab/feedback/service.go` - Feedback collection service
  - Training data export, few-shot examples, model quality metrics
- [x] **GITLAB-091**: Auto-suggest optimal models based on codebase language
  - `internal/gitlab/model_suggest/service.go` - Model suggestion service
  - Scoring: language fit, quality, speed, context size, feedback
- [x] **GITLAB-092**: Priority queue для main/release branches
  - `internal/gitlab/queue/priority.go` - Priority queue with branch rules
  - Default rules: main/master=Critical, release/hotfix=High, feature=Normal, wip=Low
- [x] **GITLAB-093**: WebSocket для real-time queue updates
  - `internal/gitlab/realtime/websocket.go` - WebSocket hub and client management
  - Events: job_created, job_started, job_completed, job_failed, worker_status, stats

---

## ⚠️ GitLab API Limits

### Comment Size Limits

| Limit | Value | Notes |
|-------|-------|-------|
| **Note body** | 1,000,000 chars | Max length for MR comment |
| **Description** | 1,048,576 chars | Max length for MR description |
| **Discussions** | 1,000,000 chars | Per discussion thread |
| **Inline comment** | 1,000,000 chars | Per line comment |

**Практический лимит:** ~50,000-100,000 символов для читаемости.

### Handling Large Reviews

```go
const (
    // GitLab limits
    MaxNoteLength        = 1_000_000   // GitLab max
    SafeNoteLength       = 50_000      // Practical limit for readability
    MaxInlineCommentLen  = 10_000      // Per-line comment limit
    
    // Split settings
    SplitThreshold       = 45_000      // When to start splitting
    PartOverlap          = 500         // Context overlap between parts
)

// CommentSplitter разбивает длинные комментарии
type CommentSplitter struct {
    maxLength int
    overlap   int
}

// Split разбивает комментарий на части если превышен лимит
func (s *CommentSplitter) Split(content string) []string {
    if len(content) <= s.maxLength {
        return []string{content}
    }
    
    var parts []string
    remaining := content
    partNum := 1
    totalParts := (len(content) / s.maxLength) + 1
    
    for len(remaining) > 0 {
        // Find split point (prefer splitting at section boundary)
        splitAt := s.findSplitPoint(remaining, s.maxLength)
        
        part := remaining[:splitAt]
        
        // Add part header/footer
        header := fmt.Sprintf("## 📄 AI Review (Part %d/%d)\n\n", partNum, totalParts)
        footer := "\n\n---\n*Continued in next comment...*"
        
        if partNum == totalParts {
            footer = "\n\n---\n*End of review*"
        }
        
        parts = append(parts, header + part + footer)
        
        // Move to next part with overlap for context
        if splitAt < len(remaining) {
            overlap := min(s.overlap, len(remaining)-splitAt)
            remaining = remaining[splitAt-overlap:]
        } else {
            remaining = ""
        }
        partNum++
    }
    
    return parts
}

// findSplitPoint finds best place to split (prefer ## headers, then ---, then \n\n)
func (s *CommentSplitter) findSplitPoint(content string, maxLen int) int {
    if len(content) <= maxLen {
        return len(content)
    }
    
    searchStart := maxLen - 1000 // Search last 1000 chars for good split point
    if searchStart < 0 {
        searchStart = 0
    }
    
    // Try to find section header (## )
    if idx := strings.LastIndex(content[searchStart:maxLen], "\n## "); idx != -1 {
        return searchStart + idx
    }
    
    // Try to find horizontal rule (---)
    if idx := strings.LastIndex(content[searchStart:maxLen], "\n---"); idx != -1 {
        return searchStart + idx
    }
    
    // Try to find paragraph break
    if idx := strings.LastIndex(content[searchStart:maxLen], "\n\n"); idx != -1 {
        return searchStart + idx
    }
    
    // Fallback: split at maxLen
    return maxLen
}

// PostReviewComment posts review, splitting if necessary
func (c *GitLabClient) PostReviewComment(ctx context.Context, projectID, mrIID int, review string) error {
    splitter := &CommentSplitter{
        maxLength: SafeNoteLength,
        overlap:   PartOverlap,
    }
    
    parts := splitter.Split(review)
    
    for i, part := range parts {
        note, err := c.PostMRNote(ctx, projectID, mrIID, part)
        if err != nil {
            return fmt.Errorf("failed to post part %d/%d: %w", i+1, len(parts), err)
        }
        
        // Rate limit between posts
        if i < len(parts)-1 {
            time.Sleep(500 * time.Millisecond)
        }
    }
    
    return nil
}
```

### Rate Limits

| Limit | Value | Scope |
|-------|-------|-------|
| **Authenticated API** | 2,000 req/min | Per user |
| **Unauthenticated** | 500 req/min | Per IP |
| **Webhooks** | - | Unlimited (from GitLab) |
| **File uploads** | 10 MB | Per file |

```go
// Rate limiter for GitLab API
type GitLabRateLimiter struct {
    limiter *rate.Limiter  // golang.org/x/time/rate
}

func NewGitLabRateLimiter() *GitLabRateLimiter {
    // 30 requests per second = 1800/min (below 2000 limit)
    return &GitLabRateLimiter{
        limiter: rate.NewLimiter(rate.Limit(30), 10),
    }
}

func (r *GitLabRateLimiter) Wait(ctx context.Context) error {
    return r.limiter.Wait(ctx)
}
```

---

## 🤖 Model Selection Rules

### ⚠️ ВАЖНО: Бизнес-правила для моделей

#### 1. Только активные модели для выбора

```go
// GET /api/admin/gitlab/models/available
type AvailableModelsResponse struct {
    LLMModels       []ModelOption `json:"llm_models"`       // Для analysis
    EmbeddingModels []ModelOption `json:"embedding_models"` // Для chunking
}

type ModelOption struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Provider string `json:"provider"` // ollama, openai, yzma, etc.
    Type     string `json:"type"`     // chat, embedding
}

// Фильтрация: только модели со статусом "active"
func (s *Service) GetAvailableModels(ctx context.Context) (*AvailableModelsResponse, error) {
    // Получаем только активные модели
    models, err := s.modelStore.ListByStatus(ctx, models.StatusActive)
    if err != nil {
        return nil, err
    }
    
    var llm, embedding []ModelOption
    for _, m := range models {
        opt := ModelOption{
            ID:       m.ID,
            Name:     m.Name,
            Provider: m.Provider,
            Type:     m.Type,
        }
        
        switch m.Type {
        case "chat", "completion":
            llm = append(llm, opt)
        case "embedding":
            embedding = append(embedding, opt)
        }
    }
    
    return &AvailableModelsResponse{
        LLMModels:       llm,
        EmbeddingModels: embedding,
    }, nil
}
```

#### 2. Блокировка деактивации используемой модели

```go
// При попытке деактивировать модель - проверяем использование в GitLab

// GET /api/admin/models/:id/usage
type ModelUsageResponse struct {
    ModelID         string              `json:"model_id"`
    IsUsedInGitLab  bool                `json:"is_used_in_gitlab"`
    GitLabProjects  []GitLabProjectRef  `json:"gitlab_projects,omitempty"`
    CanDeactivate   bool                `json:"can_deactivate"`
    BlockingReason  string              `json:"blocking_reason,omitempty"`
}

type GitLabProjectRef struct {
    IntegrationID   string `json:"integration_id"`
    IntegrationName string `json:"integration_name"`
    ProjectID       int64  `json:"project_id"`
    ProjectName     string `json:"project_name"`
    UsageType       string `json:"usage_type"` // "analysis" | "embedding"
}

// Проверка перед деактивацией
func (s *ModelService) CanDeactivate(ctx context.Context, modelID string) (*ModelUsageResponse, error) {
    // Ищем проекты где модель используется
    projects, err := s.gitlabStore.FindProjectsByModel(ctx, modelID)
    if err != nil {
        return nil, err
    }
    
    resp := &ModelUsageResponse{
        ModelID:        modelID,
        IsUsedInGitLab: len(projects) > 0,
        GitLabProjects: projects,
        CanDeactivate:  len(projects) == 0,
    }
    
    if len(projects) > 0 {
        resp.BlockingReason = fmt.Sprintf(
            "Model is used in %d GitLab project(s). "+
            "Change model in these projects before deactivating.",
            len(projects),
        )
    }
    
    return resp, nil
}

// PATCH /api/admin/models/:id/status
func (h *ModelHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
    modelID := chi.URLParam(r, "id")
    
    var req struct {
        Status string `json:"status"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Если деактивация - проверяем использование
    if req.Status == "inactive" || req.Status == "disabled" {
        usage, err := h.service.CanDeactivate(r.Context(), modelID)
        if err != nil {
            respondError(w, err)
            return
        }
        
        if !usage.CanDeactivate {
            respondJSON(w, http.StatusConflict, map[string]any{
                "error":   "model_in_use",
                "message": usage.BlockingReason,
                "usage":   usage.GitLabProjects,
            })
            return
        }
    }
    
    // Продолжаем деактивацию
    // ...
}
```

#### 3. UI Warning при выборе модели

```svelte
<!-- project-settings-modal.svelte -->
<script>
    let availableModels = { llm_models: [], embedding_models: [] };
    let selectedAnalysisModel = '';
    let selectedEmbeddingModel = '';
    
    onMount(async () => {
        // Загружаем только активные модели
        const resp = await fetch('/api/admin/gitlab/models/available');
        availableModels = await resp.json();
    });
</script>

<div class="form-group">
    <label>Analysis Model (LLM)</label>
    <select bind:value={selectedAnalysisModel}>
        <option value="">-- Select Model --</option>
        {#each availableModels.llm_models as model}
            <option value={model.id}>
                {model.name} ({model.provider})
            </option>
        {/each}
    </select>
    {#if availableModels.llm_models.length === 0}
        <p class="warning">⚠️ No active LLM models available. 
           <a href="/admin/models">Activate a model</a> first.</p>
    {/if}
</div>

<div class="form-group">
    <label>Embedding Model</label>
    <select bind:value={selectedEmbeddingModel}>
        <option value="">-- Select Model --</option>
        {#each availableModels.embedding_models as model}
            <option value={model.id}>
                {model.name} ({model.provider})
            </option>
        {/each}
    </select>
    {#if availableModels.embedding_models.length === 0}
        <p class="warning">⚠️ No active embedding models available.</p>
    {/if}
</div>
```

#### 4. UI Warning при деактивации модели

```svelte
<!-- model-status-toggle.svelte -->
<script>
    async function toggleStatus(modelId, newStatus) {
        if (newStatus === 'inactive') {
            // Проверяем использование
            const usage = await fetch(`/api/admin/models/${modelId}/usage`);
            const data = await usage.json();
            
            if (!data.can_deactivate) {
                showModal({
                    title: '⚠️ Cannot Deactivate Model',
                    message: data.blocking_reason,
                    details: data.gitlab_projects.map(p => 
                        `• ${p.integration_name} / ${p.project_name} (${p.usage_type})`
                    ).join('\n'),
                    actions: [
                        { label: 'Go to GitLab Settings', href: '/admin/gitlab' },
                        { label: 'Cancel', close: true }
                    ]
                });
                return;
            }
        }
        
        // Proceed with status change
        await fetch(`/api/admin/models/${modelId}/status`, {
            method: 'PATCH',
            body: JSON.stringify({ status: newStatus })
        });
    }
</script>
```

#### 5. Cascade Check при удалении интеграции

```go
// При удалении GitLab интеграции - освобождаем модели
func (s *Service) DeleteIntegration(ctx context.Context, id string) error {
    // Удаляем все проекты интеграции (модели автоматически "освобождаются")
    if err := s.store.DeleteProjectsByIntegration(ctx, id); err != nil {
        return err
    }
    
    // Удаляем интеграцию
    return s.store.DeleteIntegration(ctx, id)
}
```

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

## 📋 Summary

| Metric | Value |
|--------|-------|
| **Total Tasks** | 93 + 6 subtasks = **99 tasks** |
| **Completed** | ~70 tasks (Phases 1-7, 10) |
| **Phases** | 12 |
| **Estimated Time** | 40-60 hours remaining |
| **Priority** | Medium-High |
| **Dependencies** | PostgreSQL, Qdrant (optional), Embedding Service |

### Implemented Features ✅
- ✅ **Database Models** - GitLabIntegration, GitLabProject, MRReview, AnalysisJob
- ✅ **PostgreSQL Storage** - полная реализация CRUD операций
- ✅ **GitLab API Client** - с rate limiting (30 req/sec)
- ✅ **Webhook Handler** - с debounce и deduplication
- ✅ **Code Analyzer** - LLM-based code review с промптами
- ✅ **Code Chunker** - file/hunk/function/lines стратегии
- ✅ **JSON Parser** - парсинг LLM ответов с error recovery
- ✅ **Comment Builder** - форматирование в Markdown
- ✅ **Comment Splitter** - обход лимитов GitLab (50K chars)
- ✅ **Worker Pool** - configurable workers, graceful shutdown
- ✅ **Job Queue** - PostgreSQL-based с FOR UPDATE SKIP LOCKED
- ✅ **Retry Logic** - exponential backoff
- ✅ **Model Usage Tracking** - FindProjectsByModel, IsModelUsed

### Planned Features
- ⏳ Admin API (Phase 8)
- ⏳ Admin UI Svelte (Phase 9)
- ⏳ Prometheus Metrics
- ⏳ Tests & Documentation

---

## 🚀 Quick Start (после реализации)

1. **Добавить интеграцию** в Admin → GitLab → Add Integration
2. **Создать Access Token** в GitLab: Settings → Access Tokens (scopes: api, read_repository)
3. **Добавить проект** и настроить webhook
4. **Создать MR** - автоматически получите AI review!

---

*Документ создан: 2025-12-04*
*Версия: 1.1.0*
*Изменения: добавлены Queue Monitor UI, API Pagination, Complete Flow диаграмма*

