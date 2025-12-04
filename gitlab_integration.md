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
// Дедупликация через Redis/in-memory с debounce
type WebhookDeduplicator struct {
    cache      cache.Cache           // Redis или in-memory
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
│                     WEBHOOK PROCESSING FLOW                      │
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
│               │  (Redis/PostgreSQL/Memory)  │                   │
│               │                             │                   │
│               │  ┌─────────────────────┐   │                   │
│               │  │ Workers pick jobs   │   │                   │
│               │  │ and process them    │   │                   │
│               │  └─────────────────────┘   │                   │
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
│  5. ENQUEUE (Redis/PostgreSQL/Memory)                           │
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
│                            │     (Redis / PostgreSQL)        │  │
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

### Queue Storage Options

```go
// Option 1: Redis Queue (рекомендуется для distributed setup)
type RedisAnalysisQueue struct {
    client    *redis.Client
    queueKey  string // "gitlab:analysis:queue"
    jobsKey   string // "gitlab:analysis:jobs:{id}"
    logger    *logrus.Logger
}

// Option 2: PostgreSQL Queue (если Redis недоступен)
type PostgresAnalysisQueue struct {
    db     *sqlx.DB
    logger *logrus.Logger
}

// CREATE TABLE analysis_jobs (
//     id VARCHAR(36) PRIMARY KEY,
//     integration_id VARCHAR(36) NOT NULL,
//     project_id INTEGER NOT NULL,
//     mr_iid INTEGER NOT NULL,
//     status VARCHAR(20) NOT NULL DEFAULT 'pending',
//     priority INTEGER NOT NULL DEFAULT 0,
//     payload JSONB NOT NULL,
//     worker_id VARCHAR(50),
//     retry_count INTEGER DEFAULT 0,
//     max_retries INTEGER DEFAULT 3,
//     error TEXT,
//     created_at TIMESTAMP DEFAULT NOW(),
//     started_at TIMESTAMP,
//     completed_at TIMESTAMP,
//     UNIQUE(project_id, mr_iid, status) WHERE status = 'pending'
// );
// CREATE INDEX idx_analysis_jobs_pending ON analysis_jobs(status, priority DESC, created_at ASC) 
//     WHERE status = 'pending';

// Option 3: In-Memory Queue (для development/single instance)
type MemoryAnalysisQueue struct {
    jobs     []*AnalysisJob
    jobsMap  map[string]*AnalysisJob
    mu       sync.RWMutex
    cond     *sync.Cond
    logger   *logrus.Logger
}
```

### Queue Configuration

```yaml
# config.yaml
gitlab:
  queue:
    type: "redis"      # redis | postgres | memory
    workers: 3         # Concurrent analysis workers
    max_retries: 3     # Max retry attempts per job
    job_timeout: "30m" # Max time per job
    cleanup_interval: "1h"
    cleanup_older_than: "7d"
    
  analysis:
    default_priority: 0
    urgent_priority: 10  # For force push to main branch
```

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
- [ ] **GITLAB-027**: ⚠️ Очистка существующей коллекции перед повторным анализом MR

### Phase 5.1: Webhook Deduplication
- [ ] **GITLAB-028**: Создать `internal/gitlab/dedup/deduplicator.go`
- [ ] **GITLAB-029**: Реализовать debounce timer (10s default) для группировки webhook'ов
- [ ] **GITLAB-030**: Реализовать отмену pending анализа при новом webhook
- [ ] **GITLAB-031**: Добавить Redis/memory кэш для статусов обработки
- [ ] **GITLAB-032**: Добавить защиту от concurrent analysis одного MR

### Phase 6: AI Analysis Pipeline
- [ ] **GITLAB-033**: Создать `internal/gitlab/analyzer/analyzer.go`
- [ ] **GITLAB-034**: Реализовать структурированный промпт для code review
- [ ] **GITLAB-035**: Реализовать парсинг JSON response от LLM
- [ ] **GITLAB-036**: Реализовать per-file analysis с контекстом из Qdrant
- [ ] **GITLAB-037**: Реализовать aggregation результатов
- [ ] **GITLAB-038**: Написать unit тесты для analyzer

### Phase 7: Comment Builder
- [ ] **GITLAB-039**: Создать `internal/gitlab/comment/builder.go`
- [ ] **GITLAB-040**: Реализовать форматирование результата в Markdown
- [ ] **GITLAB-041**: Реализовать inline comments для конкретных строк
- [ ] **GITLAB-042**: Добавить collapsible sections для длинных ревью
- [ ] **GITLAB-043**: Добавить emoji и визуальные индикаторы

### Phase 8: Admin API
- [ ] **GITLAB-044**: Создать `internal/api/handlers/gitlab_admin.go`
- [ ] **GITLAB-045**: CRUD для integrations
- [ ] **GITLAB-046**: CRUD для projects
- [ ] **GITLAB-047**: Endpoint для test connection
- [ ] **GITLAB-048**: Endpoint для manual webhook setup
- [ ] **GITLAB-049**: Endpoints для reviews (list, details, retry)
- [ ] **GITLAB-050**: Добавить роуты в router.go

### Phase 9: Admin UI (Svelte)
- [ ] **GITLAB-051**: Создать страницу `/admin/gitlab` - список интеграций
- [ ] **GITLAB-052**: Создать модалку добавления/редактирования интеграции
- [ ] **GITLAB-053**: Создать страницу `/admin/gitlab/:id/projects` - список проектов
- [ ] **GITLAB-054**: Создать модалку настройки проекта:
  - [ ] **GITLAB-054a**: Выпадающий список Analysis Model (LLM для ревью)
  - [ ] **GITLAB-054b**: Выпадающий список Embedding Model (для чанкинизации)
  - [ ] **GITLAB-054c**: Поля Include/Exclude patterns
  - [ ] **GITLAB-054d**: Textarea для custom prompt
  - [ ] **GITLAB-054e**: Advanced settings (chunk size, overlap, max files)
- [ ] **GITLAB-055**: Создать страницу `/admin/gitlab/reviews` - список ревью с пагинацией
- [ ] **GITLAB-056**: Создать страницу `/admin/gitlab/reviews/:id` - детали ревью
- [ ] **GITLAB-057**: Добавить в sidebar меню "GitLab" (admin only)
- [ ] **GITLAB-058**: API endpoint для получения списка доступных моделей (LLM + Embedding)

### Phase 9.1: Queue Monitor UI
- [ ] **GITLAB-059**: Создать компонент `/admin/gitlab/queue` - мониторинг очереди
- [ ] **GITLAB-060**: Отображение статуса воркеров (active/idle, current job, duration)
- [ ] **GITLAB-061**: Отображение статистики очереди (pending, processing, completed, failed)
- [ ] **GITLAB-062**: Таблица pending jobs с возможностью отмены
- [ ] **GITLAB-063**: Таблица failed jobs с retry/delete
- [ ] **GITLAB-064**: Auto-refresh каждые 5 секунд (или WebSocket)

### Phase 10: Analysis Queue & Workers
- [ ] **GITLAB-065**: Создать `internal/gitlab/queue/interface.go` - AnalysisQueue interface
- [ ] **GITLAB-066**: Создать `internal/gitlab/queue/redis.go` - Redis implementation
- [ ] **GITLAB-067**: Создать `internal/gitlab/queue/postgres.go` - PostgreSQL implementation
- [ ] **GITLAB-068**: Создать `internal/gitlab/queue/memory.go` - In-memory implementation
- [ ] **GITLAB-069**: Создать `internal/gitlab/worker/pool.go` - WorkerPool
- [ ] **GITLAB-070**: Реализовать configurable worker count (default: 3)
- [ ] **GITLAB-071**: Реализовать graceful shutdown с drain queue
- [ ] **GITLAB-072**: Реализовать retry logic с exponential backoff
- [ ] **GITLAB-073**: Добавить job priority (urgent for main branch)
- [ ] **GITLAB-074**: Добавить job timeout handling
- [ ] **GITLAB-075**: Добавить queue cleanup (старые completed/failed jobs)
- [ ] **GITLAB-076**: Добавить Prometheus metrics для очереди

### Phase 11: Testing & Documentation
- [ ] **GITLAB-077**: Integration тесты с mock GitLab server
- [ ] **GITLAB-078**: E2E тест полного flow: webhook → queue → analysis → comment
- [ ] **GITLAB-079**: Load test очереди (100+ concurrent webhooks)
- [ ] **GITLAB-080**: Документация API endpoints (с пагинацией)
- [ ] **GITLAB-081**: Документация по настройке GitLab webhook
- [ ] **GITLAB-082**: README с примерами использования

### Phase 12: Enhancements (Future)
- [ ] **GITLAB-083**: Поддержка GitHub (дополнительно к GitLab)
- [ ] **GITLAB-084**: Поддержка Bitbucket
- [ ] **GITLAB-085**: Custom prompts per language
- [ ] **GITLAB-086**: Integration с Slack/Teams для нотификаций
- [ ] **GITLAB-087**: Статистика и аналитика по ревью
- [ ] **GITLAB-088**: Обучение на feedback (approve/reject комментариев)
- [ ] **GITLAB-089**: Auto-suggest optimal models based on codebase language
- [ ] **GITLAB-090**: Priority queue для main/release branches
- [ ] **GITLAB-091**: WebSocket для real-time queue updates

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
| **Total Tasks** | 91 + 5 subtasks = **96 tasks** |
| **Phases** | 13 (including Phase 5.1 and 9.1) |
| **Estimated Time** | 120-160 hours |
| **Priority** | Medium-High |
| **Dependencies** | Qdrant, Redis/PostgreSQL, Embedding Service |

### Key Features
- ✅ Webhook deduplication с debounce
- ✅ Очистка Qdrant перед повторным анализом
- ✅ **Analysis Queue** с configurable workers
- ✅ **Parallel processing** нескольких MR
- ✅ **Redis/PostgreSQL/Memory** queue backends
- ✅ Выбор Analysis Model (LLM) per project
- ✅ Выбор Embedding Model per project
- ✅ Custom review prompts
- ✅ File filters (include/exclude patterns)
- ✅ Inline comments на строки кода
- ✅ GitLab native discussions API
- ✅ **Priority queue** для urgent branches
- ✅ **Retry logic** с exponential backoff
- ✅ **Queue Monitor UI** со статусом воркеров
- ✅ **API Pagination** для всех списков
- ✅ **Complete Flow диаграмма** (Webhook → Queue → Worker)

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

