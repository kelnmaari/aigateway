# 📡 API Documentation

Complete API reference for AIGateway Platform.

## 📑 Table of Contents

- [Overview](#overview)
- [Authentication](#authentication)
- [Base URL](#base-url)
- [OpenAI-Compatible Endpoints](#openai-compatible-endpoints)
  - [Chat Completions](#post-v1chatcompletions)
  - [List Models](#get-v1models)
  - [Embeddings](#post-v1embeddings)
  - [Legacy Completions](#post-v1completions)
- [Proxy Endpoints](#proxy-endpoints)
  - [Health Check](#get-health)
  - [Server Statistics](#get-apistats)
  - [Server Configuration](#get-apiconfig)
  - [Prometheus Metrics](#get-metrics)
- [Error Responses](#error-responses)
- [Rate Limiting](#rate-limiting)
- [Code Examples](#code-examples)

---

## Overview

AIGateway Platform provides an **OpenAI-compatible API** for local Ollama models. All endpoints follow OpenAI API standards with additional features like function calling and embeddings.

### API Version

- **Current Version**: v1
- **OpenAI Compatibility**: OpenAI API v1
- **Base Path**: `/v1` for OpenAI endpoints

### Content Type

All requests must use:
```
Content-Type: application/json
```

### Response Format

All responses are in JSON format with appropriate HTTP status codes.

---

## Authentication

### API Key Authentication

Include your API key in the `Authorization` header:

```http
Authorization: Bearer sk-your-api-key-here
```

### Getting an API Key

1. Launch TUI: `./bin/tui.exe`
2. Press `4` (API Keys)
3. Press `n` (New key)
4. Follow the creation wizard

Or use the admin API key from `configs/dev.yaml`:
```yaml
auth:
  admin_key: "sk-admin-dev-key-12345"
```

⚠️ **Change the admin key in production!**

### Public Endpoints

The following endpoints **do not require** authentication:
- `GET /health`
- `GET /healthz`
- `GET /ready`

---

## Base URL

### Development
```
http://localhost:8080
```

### Production
```
https://your-domain.com
```

---

## OpenAI-Compatible Endpoints

### POST /v1/chat/completions

Create a chat completion with optional streaming and function calling.

#### Request

**Headers:**
```http
Content-Type: application/json
Authorization: Bearer sk-your-api-key
```

**Body:**
```json
{
  "model": "string (required)",
  "messages": [
    {
      "role": "system|user|assistant|tool",
      "content": "string",
      "name": "string (optional)",
      "tool_calls": [] // for assistant messages
    }
  ],
  "temperature": 0.7,
  "top_p": 1.0,
  "max_tokens": null,
  "stop": ["string"],
  "stream": false,
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "string",
        "description": "string",
        "parameters": {}
      }
    }
  ],
  "tool_choice": "auto|required|none|{\"type\": \"function\", \"function\": {\"name\": \"string\"}}"
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | ✅ Yes | Ollama model name (e.g., `qwen2.5-coder:7b`, `llama3.1`) |
| `messages` | array | ✅ Yes | Conversation messages |
| `temperature` | float | No | Sampling temperature (0-2), default: 0.7 |
| `top_p` | float | No | Nucleus sampling, default: 1.0 |
| `max_tokens` | int | No | Maximum tokens to generate |
| `stop` | array | No | Stop sequences |
| `stream` | boolean | No | Enable streaming (SSE), default: false |
| `tools` | array | No | Available functions for the model |
| `tool_choice` | string/object | No | How to invoke tools: `auto`, `required`, `none` |

#### Response (Non-Streaming)

**Status:** `200 OK`

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1699999999,
  "model": "qwen2.5-coder:7b",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?",
        "tool_calls": null
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

**Finish Reasons:**
- `stop` - Natural completion
- `length` - Max tokens reached
- `tool_calls` - Model invoked a function
- `content_filter` - Content filtered (not implemented)

#### Response (Streaming)

**Status:** `200 OK`
**Content-Type:** `text/event-stream`

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1699999999,"model":"qwen2.5-coder:7b","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1699999999,"model":"qwen2.5-coder:7b","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1699999999,"model":"qwen2.5-coder:7b","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

#### Function Calling Example

**Request:**
```json
{
  "model": "llama3.1",
  "messages": [
    {
      "role": "user",
      "content": "What's the weather in Tokyo?"
    }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get current weather for a location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "City name"
            },
            "unit": {
              "type": "string",
              "enum": ["celsius", "fahrenheit"],
              "description": "Temperature unit"
            }
          },
          "required": ["location"]
        }
      }
    }
  ],
  "tool_choice": "auto"
}
```

**Response:**
```json
{
  "id": "chatcmpl-456",
  "object": "chat.completion",
  "created": 1699999999,
  "model": "llama3.1",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": null,
        "tool_calls": [
          {
            "id": "call_abc123",
            "type": "function",
            "function": {
              "name": "get_weather",
              "arguments": "{\"location\":\"Tokyo\",\"unit\":\"celsius\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ],
  "usage": {
    "prompt_tokens": 85,
    "completion_tokens": 20,
    "total_tokens": 105
  }
}
```

#### Examples

**cURL:**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "qwen2.5-coder:7b",
    "messages": [
      {"role": "user", "content": "Write a hello world in Go"}
    ],
    "temperature": 0.7
  }'
```

**cURL (Streaming):**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "qwen2.5-coder:7b",
    "messages": [{"role": "user", "content": "Count to 5"}],
    "stream": true
  }' \
  --no-buffer
```

**Python:**
```python
import openai

client = openai.OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="sk-your-api-key"
)

response = client.chat.completions.create(
    model="qwen2.5-coder:7b",
    messages=[
        {"role": "user", "content": "Explain async/await"}
    ],
    temperature=0.7
)

print(response.choices[0].message.content)
```

**JavaScript:**
```javascript
const OpenAI = require('openai');

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'sk-your-api-key'
});

async function main() {
  const response = await client.chat.completions.create({
    model: 'qwen2.5-coder:7b',
    messages: [
      { role: 'user', content: 'Write a React component' }
    ]
  });
  
  console.log(response.choices[0].message.content);
}

main();
```

---

### GET /v1/models

List all available Ollama models.

#### Request

**Headers:**
```http
Authorization: Bearer sk-your-api-key
```

**Query Parameters:** None

#### Response

**Status:** `200 OK`

```json
{
  "object": "list",
  "data": [
    {
      "id": "qwen2.5-coder:7b",
      "object": "model",
      "created": 1699999999,
      "owned_by": "ollama",
      "permission": [],
      "root": "qwen2.5-coder:7b",
      "parent": null
    },
    {
      "id": "llama3.1:latest",
      "object": "model",
      "created": 1699999999,
      "owned_by": "ollama",
      "permission": [],
      "root": "llama3.1:latest",
      "parent": null
    }
  ]
}
```

#### Examples

**cURL:**
```bash
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-your-api-key"
```

**Python:**
```python
import openai

client = openai.OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="sk-your-api-key"
)

models = client.models.list()
for model in models.data:
    print(model.id)
```

**JavaScript:**
```javascript
const OpenAI = require('openai');

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'sk-your-api-key'
});

async function listModels() {
  const models = await client.models.list();
  models.data.forEach(model => {
    console.log(model.id);
  });
}

listModels();
```

---

### POST /v1/embeddings

Generate embeddings (vector representations) for input text.

#### Request

**Headers:**
```http
Content-Type: application/json
Authorization: Bearer sk-your-api-key
```

**Body:**
```json
{
  "model": "string (required)",
  "input": "string or array of strings (required)",
  "encoding_format": "float|base64",
  "dimensions": null,
  "user": "string (optional)"
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | ✅ Yes | Embedding model (e.g., `nomic-embed-text`) |
| `input` | string/array | ✅ Yes | Text(s) to embed |
| `encoding_format` | string | No | Return format: `float` (default) or `base64` |
| `dimensions` | int | No | Number of dimensions (not supported) |
| `user` | string | No | User identifier for tracking |

#### Response

**Status:** `200 OK`

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.123, -0.456, 0.789, ...],
      "index": 0
    }
  ],
  "model": "nomic-embed-text",
  "usage": {
    "prompt_tokens": 5,
    "total_tokens": 5
  }
}
```

#### Examples

**cURL:**
```bash
curl -X POST http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "nomic-embed-text",
    "input": ["Hello world", "Goodbye world"]
  }'
```

**Python:**
```python
import openai

client = openai.OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="sk-your-api-key"
)

response = client.embeddings.create(
    model="nomic-embed-text",
    input=["Hello world", "Goodbye world"]
)

for embedding in response.data:
    print(f"Embedding {embedding.index}: {len(embedding.embedding)} dimensions")
```

**JavaScript:**
```javascript
const OpenAI = require('openai');

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'sk-your-api-key'
});

async function getEmbeddings() {
  const response = await client.embeddings.create({
    model: 'nomic-embed-text',
    input: ['Hello world', 'Goodbye world']
  });
  
  response.data.forEach(emb => {
    console.log(`Embedding ${emb.index}: ${emb.embedding.length} dimensions`);
  });
}

getEmbeddings();
```

---

### POST /v1/completions

Legacy text completion endpoint (non-chat).

#### Request

**Headers:**
```http
Content-Type: application/json
Authorization: Bearer sk-your-api-key
```

**Body:**
```json
{
  "model": "string (required)",
  "prompt": "string or array (required)",
  "max_tokens": 16,
  "temperature": 0.7,
  "top_p": 1.0,
  "n": 1,
  "stream": false,
  "logprobs": null,
  "echo": false,
  "stop": ["string"],
  "presence_penalty": 0.0,
  "frequency_penalty": 0.0,
  "best_of": 1,
  "user": "string (optional)"
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | ✅ Yes | Ollama model name |
| `prompt` | string/array | ✅ Yes | Text prompt(s) |
| `max_tokens` | int | No | Maximum tokens to generate, default: 16 |
| `temperature` | float | No | Sampling temperature (0-2), default: 0.7 |
| `top_p` | float | No | Nucleus sampling, default: 1.0 |
| `n` | int | No | Number of completions, default: 1 |
| `stream` | boolean | No | Enable streaming, default: false |
| `stop` | array | No | Stop sequences |

#### Response (Non-Streaming)

**Status:** `200 OK`

```json
{
  "id": "cmpl-123",
  "object": "text_completion",
  "created": 1699999999,
  "model": "llama3.1",
  "choices": [
    {
      "text": " Once upon a time, there was...",
      "index": 0,
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 3,
    "completion_tokens": 50,
    "total_tokens": 53
  }
}
```

#### Examples

**cURL:**
```bash
curl -X POST http://localhost:8080/v1/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "llama3.1",
    "prompt": "Once upon a time",
    "max_tokens": 50
  }'
```

**Python:**
```python
import openai

client = openai.OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="sk-your-api-key"
)

response = client.completions.create(
    model="llama3.1",
    prompt="Once upon a time",
    max_tokens=50
)

print(response.choices[0].text)
```

---

## Proxy Endpoints

### GET /health

Health check endpoint for monitoring.

#### Request

**Headers:** None required

**Query Parameters:** None

#### Response

**Status:** `200 OK`

```json
{
  "status": "ok"
}
```

#### Examples

**cURL:**
```bash
curl http://localhost:8080/health
```

**Response:**
```json
{"status":"ok"}
```

---

### GET /api/stats

Get server statistics and metrics.

#### Request

**Headers:** None required (but server must be accessible)

**Query Parameters:** None

#### Response

**Status:** `200 OK`

```json
{
  "server": {
    "status": "running",
    "host": "localhost",
    "port": 8080,
    "url": "http://localhost:8080"
  },
  "ollama": {
    "connected": true,
    "url": "http://localhost:11434",
    "models_count": 5,
    "models": ["llama3.1:latest", "qwen2.5-coder:7b", ...]
  },
  "stats": {
    "uptime_seconds": 7200.5,
    "uptime": "2h0m0s",
    "total_requests": 1234,
    "active_requests": 3,
    "success_requests": 1180,
    "error_requests": 54,
    "average_duration": "1.2s"
  },
  "api_keys": {
    "enabled": true,
    "count": 5,
    "keys": [
      {
        "id": "ak_1759404858_c620cc75",
        "name": "Admin Key",
        "enabled": true,
        "created_at": "2025-10-01T10:00:00Z",
        "last_used_at": "2025-10-04T12:30:00Z",
        "permissions": ["*"],
        "rate_limit": {
          "requests_per_minute": 0,
          "requests_per_hour": 0
        }
      }
    ]
  }
}
```

#### Examples

**cURL:**
```bash
curl http://localhost:8080/api/stats
```

**Python:**
```python
import requests

response = requests.get('http://localhost:8080/api/stats')
stats = response.json()

print(f"Server: {stats['server']['status']}")
print(f"Total Requests: {stats['stats']['total_requests']}")
print(f"Active Keys: {stats['api_keys']['count']}")
```

---

### GET /api/config

Get server configuration (sensitive data excluded).

#### Request

**Headers:** None required

**Query Parameters:** None

#### Response

**Status:** `200 OK`

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout": "30s",
    "write_timeout": "180s",
    "idle_timeout": "120s",
    "max_header_bytes": 1048576
  },
  "ollama": {
    "url": "http://localhost:11434",
    "timeout": "180s",
    "retry_attempts": 3,
    "retry_delay": "2s",
    "connection_pool_size": 100,
    "keep_alive": true
  },
  "auth": {
    "enabled": true,
    "storage_type": "json",
    "storage_path": "data/api_keys.json",
    "admin_key": "",
    "jwt": {
      "secret": "",
      "expiration": "24h"
    },
    "rate_limiting": {
      "enabled": true,
      "default_requests_per_minute": 30,
      "default_requests_per_hour": 500
    }
  },
  "logging": {
    "level": "debug",
    "format": "text",
    "output": "both",
    "file_path": "logs/proxy-dev.log",
    "max_size": 100,
    "max_backups": 5,
    "max_age": 30,
    "compress": true
  },
  "models": {
    "mapping": {},
    "aliases": {},
    "hidden": [],
    "cache": {
      "enabled": true,
      "ttl": "5m0s",
      "refresh_interval": "1m0s"
    }
  },
  "tools": {
    "force_usage": false,
    "default_choice": "auto",
    "fallback_model": "llama3.1",
    "optimizer": {
      "enabled": true,
      "simplify_system_message": true,
      "smart_tool_filtering": false,
      "max_tools_per_request": 0,
      "preserve_instructions": []
    }
  },
  "metrics": {
    "enabled": true,
    "prometheus_path": "/metrics"
  },
  "tui": {
    "enabled": true,
    "refresh_rate": "1s",
    "theme": "default"
  },
  "development": {
    "hot_reload": true,
    "debug_mode": true,
    "profile_enabled": false,
    "pprof_enabled": false,
    "race_detection": true
  }
}
```

**Note:** Sensitive fields (`admin_key`, `jwt.secret`) are always empty for security.

#### Examples

**cURL:**
```bash
curl http://localhost:8080/api/config
```

---

### GET /metrics

Prometheus metrics endpoint.

#### Request

**Headers:** None required

**Query Parameters:** None

#### Response

**Status:** `200 OK`
**Content-Type:** `text/plain; version=0.0.4`

```
# HELP ollama_proxy_http_requests_total Total number of HTTP requests.
# TYPE ollama_proxy_http_requests_total counter
ollama_proxy_http_requests_total{endpoint="/v1/chat/completions",method="POST",status_code="200"} 1234

# HELP ollama_proxy_http_request_duration_seconds Duration of HTTP requests in seconds.
# TYPE ollama_proxy_http_request_duration_seconds histogram
ollama_proxy_http_request_duration_seconds_bucket{endpoint="/v1/chat/completions",method="POST",le="0.005"} 10
ollama_proxy_http_request_duration_seconds_bucket{endpoint="/v1/chat/completions",method="POST",le="0.01"} 50
ollama_proxy_http_request_duration_seconds_sum{endpoint="/v1/chat/completions",method="POST"} 1523.456
ollama_proxy_http_request_duration_seconds_count{endpoint="/v1/chat/completions",method="POST"} 1234

# HELP ollama_proxy_ollama_requests_total Total number of requests to Ollama.
# TYPE ollama_proxy_ollama_requests_total counter
ollama_proxy_ollama_requests_total{model="qwen2.5-coder:7b",operation="chat_completion",status="success"} 1100

# HELP ollama_proxy_api_keys_active_total Current number of active API keys.
# TYPE ollama_proxy_api_keys_active_total gauge
ollama_proxy_api_keys_active_total 4

# ... (more metrics)
```

#### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `ollama_proxy_http_requests_total` | Counter | Total HTTP requests |
| `ollama_proxy_http_request_duration_seconds` | Histogram | HTTP request duration |
| `ollama_proxy_http_requests_in_flight` | Gauge | Active requests |
| `ollama_proxy_http_response_size_bytes` | Histogram | Response sizes |
| `ollama_proxy_ollama_requests_total` | Counter | Ollama API calls |
| `ollama_proxy_ollama_request_duration_seconds` | Histogram | Ollama request duration |
| `ollama_proxy_ollama_errors_total` | Counter | Ollama errors |
| `ollama_proxy_api_key_requests_total` | Counter | Requests per API key |
| `ollama_proxy_api_key_rate_limit_exceeded_total` | Counter | Rate limit violations |
| `ollama_proxy_api_keys_active_total` | Gauge | Active API keys |
| `ollama_proxy_api_key_tokens_used_total` | Counter | Tokens used per key |

#### Examples

**cURL:**
```bash
curl http://localhost:8080/metrics
```

**Prometheus Configuration:**
```yaml
scrape_configs:
  - job_name: 'ollama-proxy'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

---

## Error Responses

All errors follow OpenAI API error format.

### Error Structure

```json
{
  "error": {
    "message": "Error description",
    "type": "error_type",
    "param": null,
    "code": "error_code"
  }
}
```

### HTTP Status Codes

| Code | Name | Description |
|------|------|-------------|
| `200` | OK | Request successful |
| `400` | Bad Request | Invalid request parameters |
| `401` | Unauthorized | Missing or invalid API key |
| `403` | Forbidden | Insufficient permissions |
| `404` | Not Found | Resource not found |
| `429` | Too Many Requests | Rate limit exceeded |
| `500` | Internal Server Error | Server error |
| `502` | Bad Gateway | Ollama connection error |
| `503` | Service Unavailable | Server temporarily unavailable |
| `504` | Gateway Timeout | Request to Ollama timed out |

### Example Error Responses

#### 400 Bad Request

```json
{
  "error": {
    "message": "Invalid request: model is required",
    "type": "invalid_request_error",
    "param": "model",
    "code": "missing_required_parameter"
  }
}
```

#### 401 Unauthorized

```json
{
  "error": {
    "message": "Invalid API key provided",
    "type": "invalid_request_error",
    "param": null,
    "code": "invalid_api_key"
  }
}
```

#### 403 Forbidden

```json
{
  "error": {
    "message": "API key does not have permission to access model 'llama3.1'",
    "type": "insufficient_quota",
    "param": "model",
    "code": "model_not_allowed"
  }
}
```

#### 429 Rate Limit Exceeded

```json
{
  "error": {
    "message": "Rate limit exceeded: 30 requests per minute",
    "type": "rate_limit_exceeded",
    "param": null,
    "code": "rate_limit_exceeded"
  }
}
```

**Response Headers:**
```http
X-RateLimit-Limit: 30
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1699999999
Retry-After: 60
```

#### 502 Bad Gateway

```json
{
  "error": {
    "message": "Failed to connect to Ollama server: connection refused",
    "type": "api_error",
    "param": null,
    "code": "ollama_connection_error"
  }
}
```

#### 504 Gateway Timeout

```json
{
  "error": {
    "message": "Request to Ollama timed out after 180s",
    "type": "api_error",
    "param": null,
    "code": "ollama_timeout"
  }
}
```

---

## Rate Limiting

### Per-Key Rate Limits

Rate limits are enforced per API key and can be customized when creating keys.

**Default Limits:**
- **30 requests per minute**
- **500 requests per hour**

### Rate Limit Headers

All API responses include rate limit headers:

```http
X-RateLimit-Limit: 30
X-RateLimit-Remaining: 25
X-RateLimit-Reset: 1699999999
```

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests allowed in window |
| `X-RateLimit-Remaining` | Requests remaining in current window |
| `X-RateLimit-Reset` | Unix timestamp when limit resets |

### Rate Limit Response

When limit is exceeded:

**Status:** `429 Too Many Requests`

**Headers:**
```http
X-RateLimit-Limit: 30
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1699999999
Retry-After: 60
```

**Body:**
```json
{
  "error": {
    "message": "Rate limit exceeded: 30 requests per minute. Try again in 60 seconds.",
    "type": "rate_limit_exceeded",
    "param": null,
    "code": "rate_limit_exceeded"
  }
}
```

### Best Practices

1. **Check headers** - Monitor `X-RateLimit-Remaining`
2. **Implement backoff** - Use exponential backoff on 429
3. **Respect Retry-After** - Wait specified seconds
4. **Request higher limits** - Create keys with custom limits
5. **Use admin key carefully** - Admin keys have unlimited rate limits

### Bypass Rate Limits

Admin API keys (with `*` permissions) bypass rate limiting:

```yaml
# configs/dev.yaml
auth:
  admin_key: "sk-admin-dev-key-12345"
```

---

## Code Examples

### Complete Python Client

```python
import openai
from openai import OpenAI

class OllamaProxyClient:
    def __init__(self, base_url="http://localhost:8080", api_key="sk-your-key"):
        self.client = OpenAI(base_url=f"{base_url}/v1", api_key=api_key)
    
    def chat(self, model, messages, stream=False, tools=None):
        """Send chat completion request."""
        return self.client.chat.completions.create(
            model=model,
            messages=messages,
            stream=stream,
            tools=tools
        )
    
    def chat_stream(self, model, messages):
        """Stream chat completion."""
        stream = self.client.chat.completions.create(
            model=model,
            messages=messages,
            stream=True
        )
        for chunk in stream:
            if chunk.choices[0].delta.content:
                yield chunk.choices[0].delta.content
    
    def list_models(self):
        """List available models."""
        return self.client.models.list()
    
    def embed(self, model, input_text):
        """Generate embeddings."""
        return self.client.embeddings.create(
            model=model,
            input=input_text
        )

# Usage
client = OllamaProxyClient(api_key="sk-your-api-key")

# Chat
response = client.chat(
    model="qwen2.5-coder:7b",
    messages=[{"role": "user", "content": "Hello!"}]
)
print(response.choices[0].message.content)

# Stream
for text in client.chat_stream(
    model="qwen2.5-coder:7b",
    messages=[{"role": "user", "content": "Count to 5"}]
):
    print(text, end="", flush=True)

# Models
models = client.list_models()
print([m.id for m in models.data])

# Embeddings
embeddings = client.embed(
    model="nomic-embed-text",
    input_text=["Hello", "World"]
)
print(f"Got {len(embeddings.data)} embeddings")
```

### Complete JavaScript Client

```javascript
const OpenAI = require('openai');

class OllamaProxyClient {
  constructor(baseURL = 'http://localhost:8080', apiKey = 'sk-your-key') {
    this.client = new OpenAI({
      baseURL: `${baseURL}/v1`,
      apiKey: apiKey
    });
  }

  async chat(model, messages, options = {}) {
    return await this.client.chat.completions.create({
      model,
      messages,
      ...options
    });
  }

  async *chatStream(model, messages) {
    const stream = await this.client.chat.completions.create({
      model,
      messages,
      stream: true
    });

    for await (const chunk of stream) {
      const content = chunk.choices[0]?.delta?.content;
      if (content) yield content;
    }
  }

  async listModels() {
    return await this.client.models.list();
  }

  async embed(model, input) {
    return await this.client.embeddings.create({
      model,
      input
    });
  }
}

// Usage
const client = new OllamaProxyClient('http://localhost:8080', 'sk-your-api-key');

// Chat
(async () => {
  const response = await client.chat('qwen2.5-coder:7b', [
    { role: 'user', content: 'Hello!' }
  ]);
  console.log(response.choices[0].message.content);

  // Stream
  for await (const text of client.chatStream('qwen2.5-coder:7b', [
    { role: 'user', content: 'Count to 5' }
  ])) {
    process.stdout.write(text);
  }

  // Models
  const models = await client.listModels();
  console.log(models.data.map(m => m.id));

  // Embeddings
  const embeddings = await client.embed('nomic-embed-text', ['Hello', 'World']);
  console.log(`Got ${embeddings.data.length} embeddings`);
})();
```

### cURL Cheat Sheet

```bash
# Chat (basic)
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-key" \
  -d '{"model":"qwen2.5-coder:7b","messages":[{"role":"user","content":"Hello"}]}'

# Chat (streaming)
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-key" \
  -d '{"model":"qwen2.5-coder:7b","messages":[{"role":"user","content":"Hello"}],"stream":true}' \
  --no-buffer

# List models
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-your-key"

# Embeddings
curl -X POST http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-key" \
  -d '{"model":"nomic-embed-text","input":["Hello","World"]}'

# Health check
curl http://localhost:8080/health

# Statistics
curl http://localhost:8080/api/stats

# Metrics
curl http://localhost:8080/metrics
```

---

## Additional Resources

- **[TUI Guide](TUI_GUIDE.md)** - Terminal UI documentation
- **[Configuration](CONFIGURATION.md)** - Configuration options
- **[Troubleshooting](TROUBLESHOOTING.md)** - Common issues
- **[Performance](PERFORMANCE.md)** - Performance tuning

---

**Questions or issues?** [Open an issue on GitHub](https://github.com/yourusername/aigateway/issues)


