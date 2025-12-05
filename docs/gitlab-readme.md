# GitLab MR Review Integration

Automated code review for GitLab Merge Requests using LLM models.

## Features

- 🔍 **Automatic Code Review** - AI-powered analysis of MR changes
- 🎯 **Configurable Models** - Use any LLM available in AIGateway
- 📁 **File Filtering** - Include/exclude patterns for targeted review
- 💬 **GitLab Comments** - Review results posted directly to MR
- 🔄 **Queue System** - Async processing with retry support
- 📊 **Prometheus Metrics** - Full observability
- 🔐 **Secure** - Webhook verification, token encryption

## Quick Start

### 1. Enable GitLab Integration

Add to your `config.yaml`:

```yaml
gitlab:
  enabled: true
  queue:
    workers: 4
    max_retries: 3
    retry_delay: 30s
  webhook:
    base_url: "https://aigateway.example.com"
    dedup_window: 5m
```

### 2. Create Integration

```bash
curl -X POST "http://localhost:8080/api/admin/gitlab/integrations" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My GitLab",
    "gitlab_url": "https://gitlab.example.com",
    "access_token": "glpat-xxxx"
  }'
```

### 3. Add Project

```bash
curl -X POST "http://localhost:8080/api/admin/gitlab/integrations/<id>/projects" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "gitlab_project_id": 123,
    "analysis_model_id": "<model-uuid>",
    "auto_review": true
  }'
```

### 4. Setup Webhook

```bash
curl -X POST "http://localhost:8080/api/admin/gitlab/projects/<id>/webhook" \
  -H "Authorization: Bearer <token>"
```

Done! New MRs will be automatically reviewed.

## Configuration

### Integration Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `name` | Display name | Required |
| `gitlab_url` | GitLab instance URL | Required |
| `access_token` | Personal Access Token | Required |
| `webhook_secret` | Secret for verification | Auto-generated |

### Project Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `analysis_model_id` | LLM for code review | Required |
| `embedding_model_id` | Model for RAG | Optional |
| `auto_review` | Enable auto-review | `true` |
| `review_prompt` | Custom instructions | Default prompt |

### Advanced Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `include_patterns` | Files to include | All files |
| `exclude_patterns` | Files to exclude | None |
| `max_files_per_mr` | Max files to analyze | 50 |
| `max_lines_per_file` | Max lines per file | 1000 |
| `skip_draft_mrs` | Skip WIP/Draft MRs | `true` |
| `skip_bots` | Skip bot authors | `true` |
| `chunk_size` | Code chunk size (tokens) | 4000 |
| `chunk_overlap` | Chunk overlap (tokens) | 200 |

## Example: Custom Review Prompt

```json
{
  "review_prompt": "Review this code with focus on:\n1. Security vulnerabilities\n2. Performance issues\n3. Go idioms and best practices\n4. Error handling\n\nProvide specific line references for each issue."
}
```

## Example: File Filtering

```json
{
  "settings": {
    "include_patterns": [
      "*.go",
      "*.ts",
      "src/**/*.js"
    ],
    "exclude_patterns": [
      "vendor/**",
      "node_modules/**",
      "*.generated.go",
      "*_test.go",
      "*.lock"
    ]
  }
}
```

## Review Output

The AI reviewer posts a comment like this:

```markdown
## 🔍 AI Code Review

**Summary:** 3 issues found in 5 files (150 lines changed)

### Critical Issues
1. **SQL Injection** in `db/queries.go:42`
   ```go
   query := "SELECT * FROM users WHERE id = " + userID
   ```
   Use parameterized queries to prevent SQL injection.

### Warnings
2. **Potential nil pointer** in `handler.go:87`
   Check `user` for nil before accessing fields.

### Suggestions
3. **Consider using context** in `service.go:23`
   Pass context for cancellation support.

---
*Reviewed by AIGateway using gpt-4 • 4.5k tokens • 12.5s*
```

## Queue Management

### View Queue Status

```bash
curl "http://localhost:8080/api/admin/gitlab/queue/status" \
  -H "Authorization: Bearer <token>"
```

Response:
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
    "total": 4
  }
}
```

### Retry Failed Review

```bash
curl -X POST "http://localhost:8080/api/admin/gitlab/reviews/<id>/retry" \
  -H "Authorization: Bearer <token>"
```

## Monitoring

### Prometheus Metrics

Key metrics to monitor:

```promql
# Queue depth
aigateway_gitlab_queue_jobs_current{status="pending"}

# Processing time (p95)
histogram_quantile(0.95, aigateway_gitlab_queue_job_duration_seconds_bucket)

# Success rate
sum(aigateway_gitlab_reviews_completed_total{status="completed"}) / 
sum(aigateway_gitlab_reviews_completed_total)

# LLM token usage
sum(rate(aigateway_gitlab_llm_tokens_used_total[1h])) by (model_id)
```

### Grafana Dashboard

Import the GitLab dashboard:
- Dashboard ID: TBD
- Or use JSON from `grafana/gitlab-dashboard.json`

## Troubleshooting

### MR Not Being Reviewed

1. Check project `auto_review` is enabled
2. Check MR is not a draft (if `skip_draft_mrs` enabled)
3. Check author is not a bot (if `skip_bots` enabled)
4. Check files match `include_patterns`
5. Check logs for errors

### Review Taking Too Long

1. Check model availability
2. Check queue depth
3. Reduce `max_files_per_mr` or `max_lines_per_file`
4. Add more workers

### Rate Limit Errors

1. Reduce concurrent projects
2. Check GitLab rate limit headers
3. Increase `rate_limit_per_sec` in config

## Architecture

```
┌─────────────┐     Webhook      ┌─────────────┐
│   GitLab    │ ───────────────► │  AIGateway  │
│             │                  │             │
│   MR Event  │                  │  Handler    │
└─────────────┘                  └──────┬──────┘
                                        │
                                        ▼
                                 ┌─────────────┐
                                 │    Queue    │
                                 │  (PostgreSQL)│
                                 └──────┬──────┘
                                        │
                         ┌──────────────┼──────────────┐
                         ▼              ▼              ▼
                    ┌─────────┐   ┌─────────┐   ┌─────────┐
                    │ Worker 1│   │ Worker 2│   │ Worker N│
                    └────┬────┘   └────┬────┘   └────┬────┘
                         │              │              │
                         └──────────────┼──────────────┘
                                        ▼
                                 ┌─────────────┐
                                 │  LLM Model  │
                                 │  (Analysis) │
                                 └──────┬──────┘
                                        │
                                        ▼
                                 ┌─────────────┐
                                 │   GitLab    │
                                 │  (Comment)  │
                                 └─────────────┘
```

## API Reference

See [GitLab API Documentation](./gitlab-api.md) for complete API reference.

## Webhook Setup

See [Webhook Setup Guide](./gitlab-webhook-setup.md) for detailed setup instructions.

## License

Part of AIGateway. See LICENSE file.

