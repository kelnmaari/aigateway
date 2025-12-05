# GitLab Webhook Setup Guide

## Overview

This guide explains how to configure GitLab webhooks for automatic MR code review using AIGateway.

## Prerequisites

1. **AIGateway** running and accessible from GitLab
2. **GitLab Personal Access Token** with the following scopes:
   - `api` - Full API access
   - `read_repository` - Read repository content
   - `write_repository` (optional) - For posting comments

3. **Network Access** - GitLab must be able to reach AIGateway webhook endpoint

## Step 1: Create GitLab Integration in AIGateway

### Via Web UI

1. Navigate to **Admin** → **GitLab**
2. Click **Add Integration**
3. Fill in the form:
   - **Name**: Descriptive name (e.g., "GitLab Production")
   - **GitLab URL**: Your GitLab instance URL (e.g., `https://gitlab.example.com`)
   - **Personal Access Token**: Your GitLab PAT
   - **Webhook Secret**: (Optional) Secret for webhook verification
4. Click **Create**
5. Test the connection using the **Test** button

### Via API

```bash
curl -X POST "https://aigateway.example.com/api/admin/gitlab/integrations" \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GitLab Production",
    "gitlab_url": "https://gitlab.example.com",
    "access_token": "glpat-xxxxxxxxxxxxxxxxxxxx",
    "webhook_secret": "your-secure-secret"
  }'
```

## Step 2: Add Project for Monitoring

### Via Web UI

1. Go to your integration detail page
2. Click **Add Project** in the Projects tab
3. Fill in the form:
   - **GitLab Project ID**: Find this in GitLab → Project → Settings → General
   - **Analysis Model**: Select an LLM model for code review
   - **Embedding Model**: (Optional) For RAG context retrieval
   - **Enable Auto Review**: Toggle on for automatic reviews
   - **Custom Prompt**: (Optional) Custom review instructions

4. Configure **Include/Exclude Patterns**:
   ```
   # Include (one pattern per line)
   *.go
   *.ts
   *.py
   src/**/*.js
   
   # Exclude (one pattern per line)
   vendor/**
   node_modules/**
   *.lock
   *.min.js
   ```

5. **Advanced Settings** (optional):
   - Chunk Size: 4000 tokens (default)
   - Chunk Overlap: 200 tokens
   - Max Files per MR: 50
   - Max Lines per File: 1000
   - Skip Draft MRs: Yes
   - Skip Bot Authors: Yes

6. Click **Add Project**

### Via API

```bash
curl -X POST "https://aigateway.example.com/api/admin/gitlab/integrations/<integration-id>/projects" \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "gitlab_project_id": 123,
    "analysis_model_id": "model-uuid",
    "auto_review": true,
    "settings": {
      "include_patterns": ["*.go", "*.ts"],
      "exclude_patterns": ["vendor/**"],
      "max_files_per_mr": 50,
      "skip_draft_mrs": true
    }
  }'
```

## Step 3: Configure Webhook in GitLab

### Option A: Automatic Setup (Recommended)

1. In AIGateway, go to the project detail
2. Click **Setup Webhook**
3. AIGateway will automatically create the webhook in GitLab

### Option B: Manual Setup

If automatic setup fails (e.g., network restrictions), configure manually:

1. Go to GitLab → Your Project → Settings → Webhooks
2. Add a new webhook:

   | Field | Value |
   |-------|-------|
   | URL | `https://aigateway.example.com/webhook/gitlab` |
   | Secret token | Same as configured in AIGateway integration |
   | Trigger | ✅ Merge request events |
   | SSL verification | ✅ Enable SSL verification |

3. Click **Add webhook**
4. Click **Test** → **Merge request events** to verify

## Webhook URL Format

```
https://<aigateway-host>/webhook/gitlab
```

Examples:
- `https://aigateway.example.com/webhook/gitlab`
- `http://localhost:8080/webhook/gitlab` (development)

## Supported Events

| Event | Action | Result |
|-------|--------|--------|
| Merge Request Hook | `open` | Creates review job |
| Merge Request Hook | `update` | Updates review if code changed |
| Merge Request Hook | `reopen` | Creates new review |
| Merge Request Hook | `merge` | No action |
| Merge Request Hook | `close` | No action |

## Webhook Security

### Secret Token Verification

AIGateway verifies the `X-Gitlab-Token` header against the configured webhook secret:

```
X-Gitlab-Token: your-configured-secret
```

### IP Whitelisting (Optional)

For additional security, whitelist GitLab's outbound IPs in your firewall:

**GitLab.com IPs:**
- See [GitLab IP Ranges](https://docs.gitlab.com/ee/user/gitlab_com/index.html#ip-range)

**Self-hosted GitLab:**
- Whitelist your GitLab server IP

### HTTPS

Always use HTTPS in production:
- Prevents token interception
- Required for GitLab.com webhooks

## Troubleshooting

### Webhook Not Triggering

1. **Check webhook status** in GitLab:
   - Go to Settings → Webhooks
   - Click Edit on the webhook
   - Check "Recent Deliveries"

2. **Common issues:**
   - SSL certificate errors → Ensure valid certificate
   - Network unreachable → Check firewall rules
   - Timeout → AIGateway must respond within 10s

### Reviews Not Created

1. **Check project settings:**
   - Auto Review enabled?
   - Project status is `active`?

2. **Check filters:**
   - Include patterns match files?
   - Exclude patterns too broad?
   - Skip draft MRs enabled for WIP MR?

3. **Check logs:**
   ```bash
   # AIGateway logs
   tail -f /var/log/aigateway/server.log | grep gitlab
   ```

### Duplicate Reviews

1. **Webhook deduplication** - AIGateway deduplicates webhooks within 5 minutes
2. **Check for multiple webhooks** in GitLab project settings
3. **Check X-Gitlab-Event-UUID** header for duplicate events

### Authentication Errors

1. **Token expired** - Rotate the Personal Access Token
2. **Insufficient scopes** - Ensure `api` scope is granted
3. **Token revoked** - Create a new token

### Rate Limiting

GitLab API rate limits:
- **GitLab.com**: 2000 requests/minute
- **Self-hosted**: Configurable

AIGateway implements rate limiting (30 req/sec default) to stay within limits.

If you see `429 Too Many Requests`:
1. Reduce concurrent projects
2. Increase rate limit interval in config
3. Contact GitLab admin to increase limits

## Example: Complete Setup Flow

```bash
# 1. Create integration
INTEGRATION_ID=$(curl -s -X POST "https://aigateway.example.com/api/admin/gitlab/integrations" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production GitLab",
    "gitlab_url": "https://gitlab.company.com",
    "access_token": "glpat-xxx",
    "webhook_secret": "secret123"
  }' | jq -r '.id')

# 2. Add project
PROJECT_ID=$(curl -s -X POST "https://aigateway.example.com/api/admin/gitlab/integrations/$INTEGRATION_ID/projects" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "gitlab_project_id": 42,
    "analysis_model_id": "gpt-4-uuid",
    "auto_review": true,
    "settings": {
      "include_patterns": ["*.go"],
      "skip_draft_mrs": true
    }
  }' | jq -r '.id')

# 3. Setup webhook
curl -X POST "https://aigateway.example.com/api/admin/gitlab/projects/$PROJECT_ID/webhook" \
  -H "Authorization: Bearer $JWT_TOKEN"

# 4. Verify
curl -s "https://aigateway.example.com/api/admin/gitlab/integrations/$INTEGRATION_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

## Next Steps

- [API Documentation](./gitlab-api.md) - Full API reference
- [GitLab Integration README](./gitlab-readme.md) - Overview and examples

