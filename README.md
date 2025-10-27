# 🚀 AIGateway Platform

> **Universal AI Infrastructure Platform with Multi-Provider Support**  
> OpenAI-Compatible API • RAG System • Multi-Tenancy • Enterprise Security • GPU Acceleration

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-2.0.0-brightgreen.svg)](VERSION)

---

## 📖 What is AIGateway?

**AIGateway** is a production-ready, enterprise-grade AI infrastructure platform that provides a unified **OpenAI-compatible API** for multiple AI model providers. Built with Go 1.25+, it combines local models (Ollama, vLLM), RAG capabilities, and enterprise features in a single platform.

### 🎯 Why AIGateway?

- **🔀 Multi-Provider**: Unified API для Ollama, vLLM, HuggingFace models (future: OpenAI, Anthropic)
- **🧠 RAG Built-in**: Retrieval-Augmented Generation with vector search, embeddings, external data sources
- **🏢 Enterprise-Ready**: Multi-tenancy, RBAC, OIDC/LDAP, audit logging, quotas
- **⚡ GPU Optimized**: Multi-GPU support через vLLM для fast inference
- **🌐 ChatGPT-like UI**: Полнофункциональный WebUI с RAG integration
- **📊 Observability**: OpenTelemetry, Prometheus, GPU metrics, distributed tracing

---

## ✨ Key Features

### 🤖 AI Model Support

- ✅ **Ollama Integration** - Local models с GGUF format
- ✅ **vLLM Support** - HuggingFace models с GPU acceleration *(v2.2.0)*
- ✅ **OpenAI-Compatible API** - `/v1/chat/completions`, `/v1/models`, `/v1/embeddings`
- 🔥 **Real-time Streaming** - Server-Sent Events (SSE) для живых ответов
- 🛠️ **Function Calling** - Инструменты в стиле OpenAI
- 🎨 **Vision Support** - Multimodal models (LLaVA, BakLLaVA)

### 🧠 RAG System (v2.0.0)

- 📄 **Multi-Format Documents** - PDF, DOCX, CSV, TXT, Images с OCR
- 🔌 **External Data Sources** - REST APIs, PostgreSQL databases, Web scraping
- 🧬 **Semantic Chunking** - Intelligent text splitting с overlap
- 🔍 **Vector Search** - PgVector с HNSW indexing
- ⚡ **Async Processing** - Worker pool для document processing
- 🔐 **Credentials Encryption** - AES-256 для sensitive data
- 💬 **Chat Integration** - RAG toggle, source selector, Top-K controls в UI

### 🔐 Enterprise Security

- 🔒 **JWT Authentication** - Access & Refresh tokens с rotation
- 👥 **Multi-Tenancy** - Organizations с membership и RBAC
- 🔑 **API Key Management** - Personal & Tenant keys с model-level permissions
- ⏱️ **Advanced Rate Limiting** - Per-key, per-tenant, per-endpoint limits
- 🔍 **Audit Logging** - Comprehensive event tracking для compliance
- 🛡️ **OIDC/LDAP** - Enterprise SSO integration (Keycloak, Active Directory)

### 📊 Monitoring & Observability

- 📈 **Real-time Dashboard** - CPU, Memory, Goroutines, GPU metrics
- 🎮 **Multi-GPU Monitoring** - NVIDIA GPU metrics через `nvidia-smi` (БЕЗ CGO!)
- 📉 **Prometheus Integration** - Full metrics export для Grafana
- 🔍 **OpenTelemetry Tracing** - Distributed tracing (Jaeger/Zipkin)
- 📋 **Structured Logging** - Enhanced logs с SSE streaming
- ⚠️ **Separate Error Log** - Dedicated error/warning log file с rotation

### ⚡ Performance & UX

- 🎨 **Dynamic Model Parameters** - Temperature, top_p, context window в UI
- 📊 **Context Tracking** - Real-time usage monitoring
- 🔄 **Auto-Summarization** - Context compression при overflow
- 💾 **Conversation Export/Import** - JSON, Markdown, Text formats
- 🌐 **Modern WebUI** - Consistent dark theme, responsive design
- 🖥️ **Terminal UI (TUI)** - Alternative management interface

---

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────┐
│ AIGateway Platform (Port 8080)                   │
│                                                  │
│ ┌──────────────────────────────────────────────┐ │
│ │ OpenAI-Compatible API Layer                  │ │
│ │ /v1/chat/completions, /v1/models, etc        │ │
│ └──────────────────────────────────────────────┘ │
│                                                  │
│ ┌──────────────────────────────────────────────┐ │
│ │ Model Router & Registry                      │ │
│ │  ├─ Ollama models → Ollama (11434)          │ │
│ │  ├─ HF models (GPU) → vLLM (8000)           │ │
│ │  └─ Cloud APIs → OpenAI/Anthropic (future)  │ │
│ └──────────────────────────────────────────────┘ │
│                                                  │
│ ┌──────────────────────────────────────────────┐ │
│ │ RAG System                                   │ │
│ │  ├─ Document Processing Pipeline            │ │
│ │  ├─ Vector Store (PgVector)                 │ │
│ │  ├─ Embeddings (Ollama)                     │ │
│ │  └─ Orchestrator & Reranking                │ │
│ └──────────────────────────────────────────────┘ │
│                                                  │
│ ┌──────────────────────────────────────────────┐ │
│ │ Enterprise Features                          │ │
│ │  ├─ Multi-Tenancy & RBAC                    │ │
│ │  ├─ Rate Limiting & Quotas                  │ │
│ │  ├─ Audit Logging                           │ │
│ │  └─ OIDC/LDAP Integration                   │ │
│ └──────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
```

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.25+** (для сборки из исходников)
- **Ollama** server ([Download Ollama](https://ollama.ai/))
- **PostgreSQL 13+** с pgvector extension (для RAG)
- At least one model: `ollama pull llama3.2`

### Installation

#### Option 1: Docker Compose (Recommended)

```bash
git clone https://github.com/yourusername/aigateway.git
cd aigateway

# Build & Start all services
docker compose --profile build up --build

# Server: http://localhost:8080
# WebUI: http://localhost:8080/login
```

#### Option 2: Build from Source

```bash
git clone https://github.com/yourusername/aigateway.git
cd aigateway

# Install dependencies
go mod tidy

# Build all binaries
./build.sh all

# Binaries в папке dist/
```

### First Run

1. **Start Ollama**:

```bash
ollama serve
ollama pull llama3.2  # Or any other model
```

2. **Configure AIGateway**:

```bash
cp configs/production.yaml.example configs/production.yaml
# Edit configs/production.yaml with your settings
```

3. **Run AIGateway**:

```bash
# From source
./dist/server -config configs/production.yaml

# Or with Docker
docker compose up
```

4. **Access WebUI**:

Open [http://localhost:8080/login](http://localhost:8080/login)

**Default credentials** (первый запуск):
- Email: `admin@localhost`
- Password: `admin`

⚠️ **Change default password immediately!**

---

## 📚 Documentation

### Getting Started
- [Quick Start Guide](docs/QUICKSTART.md)
- [Configuration Guide](docs/CONFIGURATION.md)
- [Docker Deployment](docs/DOCKER.md)

### Features
- [RAG System Guide](docs/RAG_DEPLOYMENT_GUIDE.md)
- [vLLM Installation](docs/VLLM_INSTALLATION.md)
- [Multi-Tenancy Setup](docs/MULTITENANCY.md)
- [OIDC Integration](docs/OIDC_SETUP.md)

### Development
- [Architecture Overview](Architecture.MD)
- [Development Guide](docs/DEVELOPMENT.md)
- [API Reference](docs/API.md)
- [Contributing Guide](CONTRIBUTING.md)

---

## 🎯 Use Cases

### 1. **Private ChatGPT Alternative**
Run your own ChatGPT-like service с local models и full data control.

### 2. **RAG-Powered Knowledge Base**
Upload documents, connect databases, enable semantic search через RAG system.

### 3. **Multi-Team AI Platform**
Multi-tenancy с isolated workspaces, RBAC, usage quotas.

### 4. **GPU-Accelerated Inference**
Use vLLM для fast inference HuggingFace models на multi-GPU setups.

### 5. **Enterprise AI Gateway**
Unified API для multiple LLM providers с governance, audit, compliance.

---

## 🔧 Configuration

### Basic Configuration

```yaml
# configs/production.yaml
server:
  host: "0.0.0.0"
  port: 8080

ollama:
  url: "http://localhost:11434"
  timeout: 120s

# Enable RAG System
rag:
  enabled: true
  vector_store:
    type: "pgvector"
    connection_string: "postgresql://user:pass@localhost:5432/aigateway"
  embeddings:
    provider: "ollama"
    model: "nomic-embed-text"

# Enable vLLM Provider (v2.2.0+)
providers:
  ollama:
    enabled: true
    base_url: "http://localhost:11434"
  vllm:
    enabled: true
    base_url: "http://localhost:8000"
    tensor_parallel_size: 2  # For 2x GPU
```

See [Configuration Guide](docs/CONFIGURATION.md) для detailed options.

---

## 🧪 Testing

### Run Tests

```bash
# Unit tests
go test ./...

# With race detector
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### E2E Tests (Playwright)

```bash
cd tests/playwright
npm install
npm test
```

---

## 📊 Roadmap

See [Roadmap.MD](Roadmap.MD) для detailed development plan.

### ✅ Completed
- **v1.x** - Ollama proxy, Multi-tenancy, RBAC, OIDC, Performance monitoring
- **v2.0.0** - RAG System, Chat Export/Import UI

### 🚧 In Progress
- **v2.1.0** - Rebranding to AIGateway
- **v2.2.0** - Model Registry, vLLM Integration, Multi-Provider support

### 🔮 Planned
- **v2.3.0** - HuggingFace Hub integration, Model hot-swap
- **v2.4.0** - OpenAI/Anthropic proxy support
- **v2.5.0** - Advanced RAG features, Query analytics

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) для guidelines.

### Development Setup

```bash
# Clone repository
git clone https://github.com/yourusername/aigateway.git
cd aigateway

# Install dependencies
go mod tidy

# Run in dev mode
go run cmd/server/main.go -config configs/dev.yaml

# Run tests
go test ./...
```

---

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file для details.

---

## 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=yourusername/aigateway&type=Date)](https://star-history.com/#yourusername/aigateway&Date)

---

## 📧 Contact & Support

- 📖 **Documentation**: [docs/](docs/)
- 🐛 **Issues**: [GitHub Issues](https://github.com/yourusername/aigateway/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/yourusername/aigateway/discussions)
- 📧 **Email**: your-email@example.com

---

## 🙏 Acknowledgments

- [Ollama](https://ollama.ai/) - Awesome local LLM runtime
- [vLLM](https://vllm.ai/) - Fast inference engine
- [Gin](https://gin-gonic.com/) - HTTP framework
- [pgvector](https://github.com/pgvector/pgvector) - Vector similarity search

---

**Built with ❤️ for the open-source AI community**

---

## 📈 Project Stats

![GitHub](https://img.shields.io/github/license/yourusername/aigateway)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/yourusername/aigateway)
![GitHub last commit](https://img.shields.io/github/last-commit/yourusername/aigateway)
![GitHub issues](https://img.shields.io/github/issues/yourusername/aigateway)
![GitHub pull requests](https://img.shields.io/github/issues-pr/yourusername/aigateway)

