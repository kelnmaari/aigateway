# 🏗️ Architecture Overview — AIGateway Platform v4.0

Complete architecture documentation for AIGateway Platform.

---

## 📑 Table of Contents

- [System Overview](#system-overview)
- [High-Level Architecture](#high-level-architecture)
- [Inference Layer](#inference-layer)
- [API Layer](#api-layer)
- [Integrations](#integrations)
- [Data Flow](#data-flow)
- [Technology Stack](#technology-stack)
- [Security Architecture](#security-architecture)

---

## System Overview

**AIGateway Platform v4.0** is a production-ready Go application providing:

- **OpenAI-Compatible API** for local LLM inference
- **Multi-Provider Docker-Based Inference** (vLLM, SGLang, TGI, llama.cpp)
- **GitLab Integration** for AI-powered code reviews
- **Web Search** via Tavily API
- **RAG System** with vector search

### Key Principles

1. **OpenAI Compatibility** — Exact API format matching
2. **Multi-Provider** — Support for vLLM, SGLang, TGI, llama.cpp
3. **Docker-Native** — All inference runs in containers
4. **Security First** — JWT, API Keys, RBAC, Multi-tenancy
5. **Observability** — Prometheus metrics, container logs, health checks

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           CLIENT APPLICATIONS                            │
│    (IDE Plugins, Continue.dev, Custom Apps, GitLab Webhooks)            │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │ OpenAI API Format
                                 │ Authorization: Bearer sk-xxx
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      AIGateway Platform (Port 8080)                      │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  HTTP Server (Gin Framework)                                       │ │
│  │  /v1/chat/completions  /v1/models  /v1/embeddings                  │ │
│  │  /api/chat/completions (with tools)  /api/system/*                 │ │
│  └────────────────────────┬───────────────────────────────────────────┘ │
│                           │                                              │
│  ┌────────────────────────┴───────────────────────────────────────────┐ │
│  │  Middleware Pipeline                                                │ │
│  │  [Auth] → [Rate Limiting] → [Stats] → [Prometheus] → [Logging]    │ │
│  └────────────────────────┬───────────────────────────────────────────┘ │
│                           │                                              │
│  ┌────────────────────────┴───────────────────────────────────────────┐ │
│  │  Inference Router                                                   │ │
│  │  • Model Registry (specs, status, health)                          │ │
│  │  • Provider Selection (vLLM/SGLang/TGI/llama.cpp)                  │ │
│  │  • Request Routing (alias → container endpoint)                    │ │
│  └────────────────────────┬───────────────────────────────────────────┘ │
│                           │                                              │
│  ┌────────────────────────┴───────────────────────────────────────────┐ │
│  │  Tools Registry (v4.0+)                                             │ │
│  │  • Web Search (Tavily API)                                          │ │
│  │  • Tool Calling Handler                                             │ │
│  │  • Tool Events Streaming                                            │ │
│  └────────────────────────┬───────────────────────────────────────────┘ │
│                           │                                              │
│  ┌────────────────────────┴───────────────────────────────────────────┐ │
│  │  Docker Container Manager                                           │ │
│  │  • Container Lifecycle (create, start, stop, remove)               │ │
│  │  • Port Allocation                                                  │ │
│  │  • GPU Device Mapping                                               │ │
│  │  • Log Streaming                                                    │ │
│  └────────────────────────┬───────────────────────────────────────────┘ │
│                           │                                              │
└───────────────────────────┼──────────────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────────┐
            │               │                   │
            ▼               ▼                   ▼
    ┌───────────┐   ┌───────────┐       ┌───────────┐
    │   vLLM    │   │  SGLang   │  ...  │llama.cpp  │
    │ Container │   │ Container │       │ Container │
    │ :port1    │   │ :port2    │       │ :portN    │
    └───────────┘   └───────────┘       └───────────┘
```

---

## Inference Layer

### Providers

| Provider | Image | GPU | Formats | Use Case |
|----------|-------|-----|---------|----------|
| **vLLM** | `vllm/vllm-openai:latest` | ✅ | HF, AWQ, GPTQ | Production |
| **SGLang** | `lmsysorg/sglang:latest` | ✅ | HF | Research |
| **TGI** | `ghcr.io/huggingface/text-generation-inference:latest` | ✅ | HF, GPTQ | HF Native |
| **llama.cpp** | `ghcr.io/ggml-org/llama.cpp:server-cuda` | ✅/CPU | GGUF | Low VRAM |
| **TEI** | `ghcr.io/huggingface/text-embeddings-inference:cpu-1.7` | CPU | — | Embeddings |

### Model Lifecycle

```
┌─────────┐   ┌──────────┐   ┌─────────┐   ┌─────────┐
│ Pending │ → │ Preparing│ → │Starting │ → │ Running │
└─────────┘   └──────────┘   └─────────┘   └─────────┘
                  │                             │
                  │ (download HF model)         │ (health check)
                  ▼                             ▼
             ┌──────────┐                  ┌─────────┐
             │  Error   │                  │ Stopped │
             └──────────┘                  └─────────┘
```

### Container Management

```go
// Container naming: aigw-{alias}
// Example: aigw-qwen-7b

// GPU mapping
--gpus '"device=0,1"'

// Port allocation
// Dynamic port from 40000-50000 range
```

---

## API Layer

### OpenAI-Compatible Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | Chat completion (streaming) |
| `/v1/completions` | POST | Text completion |
| `/v1/embeddings` | POST | Text embeddings |
| `/v1/models` | GET | List available models |

### Internal API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/chat/completions` | POST | Chat with tools (web search) |
| `/api/system/inference/load` | POST | Load model |
| `/api/system/inference/stop` | POST | Stop model |
| `/api/system/inference/logs` | GET | Container logs |
| `/api/system/inference/models` | GET | Running models |

### Request Flow

```
1. Request → Auth Middleware (validate API key/JWT)
2. → Inference Router (resolve model alias → container)
3. → Tools Handler (check use_tools, inject tool definitions)
4. → Proxy to Container (forward to vLLM/SGLang/etc)
5. → Stream Response (SSE with tool events)
```

---

## Integrations

### GitLab MR Reviews

```
GitLab Webhook → AIGateway → Per-File Analysis → Comment on MR
                     │
                     ▼
              ┌──────────────┐
              │   Qdrant     │ (code embeddings)
              │ Vector Store │
              └──────────────┘
```

### Web Search (Tavily)

```
Chat Request (use_tools: true)
     │
     ▼
┌──────────────┐     ┌─────────────┐
│   LLM Call   │ ──▶ │  Tool Call  │
│  (streaming) │     │ web_search  │
└──────────────┘     └──────┬──────┘
                            │
                            ▼
                     ┌─────────────┐
                     │ Tavily API  │
                     └──────┬──────┘
                            │
                            ▼
                  ┌──────────────────┐
                  │ Inject Results   │
                  │ Continue LLM     │
                  └──────────────────┘
```

### RAG System

```
Documents → Chunking → Embeddings → PgVector/Qdrant
                                         │
Chat + RAG Toggle ──────────────────────▶│
                                         ▼
                                  Vector Search
                                         │
                                         ▼
                              Context Injection → LLM
```

---

## Data Flow

### Chat Completion

```
Client                  AIGateway                    Container
  │                         │                            │
  │──POST /v1/chat/completions────────────────────────▶ │
  │                         │                            │
  │                         │──Resolve alias─────────────│
  │                         │                            │
  │                         │──POST /v1/chat/completions─│
  │                         │                            │
  │◀─────────────────────── │◀───SSE Stream──────────────│
  │      SSE Stream         │                            │
```

### Chat with Tools

```
Client                  AIGateway            Tavily          Container
  │                         │                   │                │
  │──POST (use_tools)───────│                   │                │
  │                         │──Add tool defs────│                │
  │                         │                   │                │
  │                         │───────────────────────POST─────────│
  │                         │                   │                │
  │                         │◀──────────────────────tool_call────│
  │◀──ToolEvent(start)──────│                   │                │
  │                         │                   │                │
  │                         │───Search──────────▶                │
  │                         │◀──Results─────────│                │
  │                         │                   │                │
  │◀──ToolEvent(end)────────│                   │                │
  │                         │                   │                │
  │                         │──Inject results + continue─────────▶
  │                         │                   │                │
  │◀────────────────────────│◀──Final response───────────────────│
```

---

## Technology Stack

### Backend

| Component | Technology |
|-----------|------------|
| Language | Go 1.25+ |
| HTTP Framework | Gin |
| Database | SQLite / PostgreSQL |
| Vector Store | PgVector / Qdrant |
| Container Runtime | Docker API |
| Metrics | Prometheus |

### Frontend (WebUI)

| Component | Technology |
|-----------|------------|
| Framework | SvelteKit 2.0 |
| UI | Tailwind CSS, Shadcn/ui |
| i18n | Paraglide.js |
| Build | Vite, embedded in Go binary |

### Inference Providers

| Provider | Technology |
|----------|------------|
| vLLM | Python, PagedAttention |
| SGLang | Python, RadixAttention |
| TGI | Rust, Continuous Batching |
| llama.cpp | C++, GGUF format |

---

## Security Architecture

### Authentication

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   JWT       │     │  API Key    │     │   OIDC      │
│ (WebUI)     │     │ (API calls) │     │ (SSO)       │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                           ▼
                   ┌───────────────┐
                   │ Auth Middleware│
                   └───────────────┘
```

### API Key Scopes

- **Models** — Which models can be accessed
- **Permissions** — chat, embeddings, admin
- **Rate Limits** — Requests per minute/hour

### Multi-Tenancy

```
User → Tenant (Organization) → API Keys → Model Access
                │
                └── Members (with roles: admin, member, viewer)
```

---

## Directory Structure

```
aigateway/
├── cmd/
│   └── server/main.go          # Entry point
├── internal/
│   ├── api/
│   │   ├── handlers/           # HTTP handlers
│   │   └── router/             # Route setup
│   ├── inference/
│   │   ├── orchestrator.go     # Container management
│   │   ├── providers/          # vLLM, SGLang, TGI, llama.cpp
│   │   └── router.go           # Model routing
│   ├── tools/
│   │   ├── tavily.go           # Web search
│   │   └── tools.go            # Tool registry
│   ├── gitlab/                 # GitLab integration
│   ├── rag/                    # RAG system
│   └── storage/                # Database layer
├── web-svelte/                 # Frontend (embedded)
├── configs/                    # Configuration files
└── data/                       # Runtime data (models, db)
```

---

**Version:** 4.0.2  
**Last Updated:** 2025-12-29
