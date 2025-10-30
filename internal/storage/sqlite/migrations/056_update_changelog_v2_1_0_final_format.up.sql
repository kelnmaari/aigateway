
UPDATE changelogs SET content = '## [2.1.0] - 2025-10-27

### 🏷️ Major Rebranding

**Project renamed from "Ollama-OpenAI Proxy" to "AIGateway Platform"**

### Changed

- **Repository name**: ollama-openai-proxy → aigateway
- **Module path**: ollama-openai-proxy → aigateway
- **Container name**: ollama-openai-proxy → aigateway
- **Database file**: proxy.db → aigateway.db (optional rename)
- **Log files**: proxy.log → aigateway.log

- **Environment Variables (⚠️ Breaking Change)**: All PROXY_* → AIGATEWAY_*
  - Example: PROXY_SERVER_PORT → AIGATEWAY_SERVER_PORT
  - PROXY_OLLAMA_URL → AIGATEWAY_OLLAMA_URL
  - PROXY_DATABASE_TYPE → AIGATEWAY_DATABASE_TYPE
  - See [Migration Guide](docs/MIGRATION_GUIDE_v2.1.0.md) for complete mapping

- **Documentation Updates**: 40+ files rebranded
  - README.md - complete rewrite для AIGateway Platform
  - Architecture.MD - updated diagrams with RAG System, Model Registry
  - All docs/*.md files (20+ files)
  - All BACKLOG/*.md files

- **Code Changes**: Zero functional changes - pure rebranding
  - go.mod module path updated
  - All import statements across entire codebase
  - Docker Compose configuration
  - Dockerfile VERSION=2.1.0

- **WebUI Branding**: Complete frontend rebranding
  - All HTML page titles: "AIGateway Platform"
  - Navigation labels и headers
  - About System page
  - Footer copyright

### Why Rebranding?

**Reasons for transition to "AIGateway":**

1. **Expanded Scope**: No longer just an Ollama proxy
  - Multi-provider support: vLLM (v2.2.0), future: OpenAI, Anthropic
  - Model registry для unified API access

2. **RAG System**: Built-in RAG capabilities
  - Vector search, embeddings, document processing
  - Enterprise-ready data integration

3. **Enterprise Positioning**: "Gateway" better represents platform role
  - Central AI infrastructure component
  - Unified API для multiple backends

4. **Scalability**: Name allows future expansion
  - Cloud provider integration
  - Custom model support

### 🔄 Migration Required

**This is a BREAKING release.** Existing deployments need migration.

See [MIGRATION_GUIDE_v2.1.0.md](docs/MIGRATION_GUIDE_v2.1.0.md) for detailed steps.

**Quick Migration Checklist:**
  - Update environment variables: PROXY_* → AIGATEWAY_*
  - Update Docker image names
  - Rename database file (optional): proxy.db → aigateway.db
  - Update scripts/configs referencing old names
  - Pull new Docker images: aigateway:2.1.0

### Technical

- **Module path**: aigateway (was ollama-openai-proxy)
- **Import paths**: Updated throughout codebase (~150+ Go files)
- **Docker Compose**: Service name aigateway, volumes aigateway_data/aigateway_logs
- **Functional changes**: Zero - pure rebranding release
- **Compilation**: Verified ✅
- **Database schema**: Unchanged (backward compatible)'
WHERE version = '2.1.0';
    