# GitLab Integration API

## Overview

AIGateway provides a complete Admin API for managing GitLab integrations, enabling automated MR code reviews using LLM models.

**Base URL:** `/api/admin/gitlab`

**Authentication:** All endpoints require JWT authentication with admin role.

---

## Integrations

### List Integrations

```
GET /api/admin/gitlab/integrations
```

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 20 | Items per page (max 100) |
| offset | int | 0 | Pagination offset |
| status | string | - | Filter by status: `active`, `inactive`, `error` |
| search | string | - | Search by name or URL |

**Response:**
```json
{
  "integrations": [
    {
      "id": "uuid",
      "name": "GitLab Production",
      "gitlab_url": "https://gitlab.example.com",
      "status": "active",
      "webhook_secret": "***masked***",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-02T00:00:00Z",
      "last_sync_at": "2025-01-02T12:00:00Z",
      "last_error": null
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

### Create Integration

```
POST /api/admin/gitlab/integrations
```

**Request Body:**
```json
{
  "name": "GitLab Production",
  "gitlab_url": "https://gitlab.example.com",
  "access_token": "glpat-xxxxxxxxxxxxxxxxxxxx",
  "webhook_secret": "optional-secret-for-webhook-verification"
}
```

**Response:** Created integration object (201 Created)

### Get Integration

```
GET /api/admin/gitlab/integrations/:id
```

**Response:** Integration object with full details

### Update Integration

```
PUT /api/admin/gitlab/integrations/:id
```

**Request Body:**
```json
{
  "name": "Updated Name",
  "access_token": "new-token-if-rotating",
  "status": "active"
}
```

### Delete Integration

```
DELETE /api/admin/gitlab/integrations/:id
```

**Response:** 204 No Content

### Test Integration Connection

```
POST /api/admin/gitlab/integrations/:id/test
```

Tests connectivity to GitLab API using the configured credentials.

**Response:**
```json
{
  "success": true,
  "user": {
    "id": 1,
    "username": "admin",
    "name": "Administrator"
  },
  "api_version": "v4",
  "gitlab_version": "16.5.0"
}
```

---

## Projects

### List Projects

```
GET /api/admin/gitlab/integrations/:id/projects
```

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 20 | Items per page |
| offset | int | 0 | Pagination offset |
| status | string | - | Filter by status |
| search | string | - | Search by name |

**Response:**
```json
{
  "projects": [
    {
      "id": "uuid",
      "integration_id": "uuid",
      "gitlab_project_id": 123,
      "name": "my-project",
      "path_with_namespace": "group/my-project",
      "web_url": "https://gitlab.example.com/group/my-project",
      "status": "active",
      "auto_review": true,
      "analysis_model_id": "model-uuid",
      "embedding_model_id": null,
      "review_prompt": "Custom review instructions...",
      "webhook_id": 456,
      "settings": {
        "include_patterns": ["*.go", "*.ts"],
        "exclude_patterns": ["vendor/**", "*.lock"],
        "max_files_per_mr": 50,
        "max_lines_per_file": 1000,
        "skip_draft_mrs": true,
        "skip_bots": true,
        "chunk_size": 4000,
        "chunk_overlap": 200
      },
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Add Project

```
POST /api/admin/gitlab/integrations/:id/projects
```

**Request Body:**
```json
{
  "gitlab_project_id": 123,
  "analysis_model_id": "model-uuid",
  "embedding_model_id": "embedding-model-uuid",
  "auto_review": true,
  "review_prompt": "Focus on security and performance...",
  "settings": {
    "include_patterns": ["*.go", "*.ts", "*.py"],
    "exclude_patterns": ["vendor/**", "node_modules/**"],
    "max_files_per_mr": 50,
    "max_lines_per_file": 1000,
    "skip_draft_mrs": true,
    "skip_bots": true,
    "chunk_size": 4000,
    "chunk_overlap": 200
  }
}
```

### Get Project

```
GET /api/admin/gitlab/projects/:project_id
```

### Update Project

```
PUT /api/admin/gitlab/projects/:project_id
```

**Request Body:** Same as Add Project (partial update supported)

### Delete Project

```
DELETE /api/admin/gitlab/projects/:project_id
```

### Setup Project Webhook

```
POST /api/admin/gitlab/projects/:project_id/webhook
```

Creates a webhook in GitLab for the project to receive MR events.

**Response:**
```json
{
  "success": true,
  "webhook_id": 456,
  "webhook_url": "https://aigateway.example.com/webhook/gitlab"
}
```

---

## Reviews

### List Reviews

```
GET /api/admin/gitlab/reviews
```

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 20 | Items per page |
| offset | int | 0 | Pagination offset |
| project_id | string | - | Filter by project |
| integration_id | string | - | Filter by integration |
| status | string | - | Filter by status: `pending`, `processing`, `completed`, `failed` |
| mr_iid | int | - | Filter by MR IID |

**Response:**
```json
{
  "reviews": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "mr_iid": 42,
      "mr_title": "Add new feature",
      "mr_author": "developer",
      "source_branch": "feature/new-thing",
      "target_branch": "main",
      "status": "completed",
      "model_used": "gpt-4",
      "files_analyzed": 5,
      "lines_changed": 150,
      "issues_found": 3,
      "tokens_used": 4500,
      "processing_time_ms": 12500,
      "retry_count": 0,
      "note_id": 123456,
      "discussion_id": "abc123",
      "error": null,
      "created_at": "2025-01-01T10:00:00Z",
      "completed_at": "2025-01-01T10:00:12Z"
    }
  ],
  "total": 100
}
```

### Get Review

```
GET /api/admin/gitlab/reviews/:id
```

Returns full review details including the analysis result.

### Retry Review

```
POST /api/admin/gitlab/reviews/:id/retry
```

Re-queues a failed review for processing.

---

## Queue Management

### Get Queue Status

```
GET /api/admin/gitlab/queue/status
```

**Response:**
```json
{
  "queue": {
    "pending": 5,
    "processing": 2,
    "completed_today": 45,
    "failed_today": 3
  },
  "workers": {
    "active": 2,
    "total": 4,
    "idle": 2
  },
  "stats": {
    "avg_processing_time_ms": 8500,
    "avg_wait_time_ms": 1200,
    "success_rate": 0.94
  }
}
```

### List Queue Jobs

```
GET /api/admin/gitlab/queue/jobs
```

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 20 | Items per page |
| offset | int | 0 | Pagination offset |
| status | string | - | Filter by status |

**Response:**
```json
{
  "jobs": [
    {
      "id": "uuid",
      "review_id": "uuid",
      "status": "processing",
      "priority": 0,
      "attempt": 1,
      "max_attempts": 3,
      "worker_id": "worker-1",
      "started_at": "2025-01-01T10:00:00Z",
      "error": null
    }
  ],
  "total": 10
}
```

### Cancel Job

```
POST /api/admin/gitlab/queue/jobs/:id/cancel
```

### Retry Job

```
POST /api/admin/gitlab/queue/jobs/:id/retry
```

---

## Model Selection

### List Active Models

```
GET /api/admin/gitlab/models
```

Returns models available for selection in GitLab projects.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| capability | string | - | Filter by capability: `chat`, `embedding` |

**Response:**
```json
{
  "models": [
    {
      "id": "uuid",
      "model_id": "gpt-4",
      "name": "GPT-4",
      "provider_id": "openai",
      "capabilities": ["chat", "function-calling"],
      "description": "Most capable GPT-4 model"
    }
  ],
  "total": 5
}
```

### List Analysis Models

```
GET /api/admin/gitlab/models/analysis
```

Returns models suitable for code analysis (with `chat` capability).

### List Embedding Models

```
GET /api/admin/gitlab/models/embedding
```

Returns models suitable for code embeddings.

---

## Model Usage Check

### Check Model Usage in GitLab

```
GET /api/admin/models/:id/gitlab-usage
```

Checks if a model is used in any GitLab projects (prevents deactivation).

**Response:**
```json
{
  "model_id": "uuid",
  "is_used_in_gitlab": true,
  "can_deactivate": false,
  "gitlab_projects": [
    {
      "integration_id": "uuid",
      "integration_name": "GitLab Production",
      "project_id": "uuid",
      "project_name": "my-project",
      "usage_type": "analysis"
    }
  ],
  "blocking_reason": "Model is used in 2 GitLab project(s). Change model in these projects before deactivating."
}
```

---

## Webhook Endpoint

### GitLab Webhook Receiver

```
POST /webhook/gitlab
```

**Note:** This endpoint does not require authentication. It uses the webhook secret for verification.

**Headers:**
| Header | Description |
|--------|-------------|
| X-Gitlab-Event | Event type (e.g., `Merge Request Hook`) |
| X-Gitlab-Token | Webhook secret for verification |

**Supported Events:**
- `Merge Request Hook` - MR opened, updated, merged, closed

---

## Error Responses

All endpoints return errors in a consistent format:

```json
{
  "error": "Error message",
  "details": "Additional context if available"
}
```

**HTTP Status Codes:**
| Code | Description |
|------|-------------|
| 400 | Bad Request - Invalid parameters |
| 401 | Unauthorized - Missing or invalid token |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 409 | Conflict - Resource conflict (e.g., model in use) |
| 500 | Internal Server Error |

---

## Rate Limiting

The GitLab Admin API shares rate limits with other admin endpoints:
- 100 requests per minute per user
- 1000 requests per hour per user

---

## Prometheus Metrics

GitLab integration exposes the following Prometheus metrics:

### Queue Metrics
- `aigateway_gitlab_queue_jobs_total{status}` - Total jobs processed
- `aigateway_gitlab_queue_jobs_current{status}` - Current jobs by status
- `aigateway_gitlab_queue_job_duration_seconds` - Job processing duration
- `aigateway_gitlab_queue_wait_time_seconds` - Queue wait time

### Worker Metrics
- `aigateway_gitlab_workers_active` - Active workers
- `aigateway_gitlab_workers_total` - Total configured workers

### Webhook Metrics
- `aigateway_gitlab_webhooks_received_total{event_type,integration_id}` - Webhooks received
- `aigateway_gitlab_webhooks_deduplicated_total` - Deduplicated webhooks
- `aigateway_gitlab_webhook_processing_duration_seconds` - Processing duration

### Review Metrics
- `aigateway_gitlab_reviews_created_total{project_id,integration_id}` - Reviews created
- `aigateway_gitlab_reviews_completed_total{status}` - Reviews completed
- `aigateway_gitlab_review_analysis_duration_seconds{model_id}` - Analysis duration
- `aigateway_gitlab_review_files_analyzed` - Files analyzed per review
- `aigateway_gitlab_review_issues_found` - Issues found per review

### LLM Metrics
- `aigateway_gitlab_llm_tokens_used_total{model_id,type}` - Tokens used
- `aigateway_gitlab_llm_request_duration_seconds{model_id}` - LLM request duration
- `aigateway_gitlab_llm_errors_total{model_id,error_type}` - LLM errors

### GitLab API Metrics
- `aigateway_gitlab_api_requests_total{integration_id,endpoint,status}` - API requests
- `aigateway_gitlab_api_latency_seconds{integration_id,endpoint}` - API latency
- `aigateway_gitlab_rate_limit_remaining{integration_id}` - Rate limit remaining

