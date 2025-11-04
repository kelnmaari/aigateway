
UPDATE changelogs 
SET content = '## [2.1.0] - 2025-10-27

### 🏷️ Major Rebranding

**Project renamed from "Ollama-OpenAI Proxy" to "AIGateway Platform"**

---

### Changed

#### **Project Identity**

- Repository name: ollama-openai-proxy → aigateway
- Module path: ollama-openai-proxy → aigateway  
- Container name: ollama-openai-proxy → aigateway
- Database file: proxy.db → aigateway.db
- Log files: proxy.log → aigateway.log

#### **Environment Variables** (Breaking Change ⚠️)

- All PROXY_* variables → AIGATEWAY_*
- Example: PROXY_SERVER_PORT → AIGATEWAY_SERVER_PORT
- See [Migration Guide](docs/MIGRATION_GUIDE_v2.1.0.md) for complete variable mapping

#### **Documentation**

- ✅ README.md - complete rewrite для AIGateway Platform
- ✅ Architecture.MD - updated diagrams with RAG System, Model Registry  
- ✅ All docs/*.md files - branding updates (20+ files)
- ✅ All BACKLOG/*.md files - task descriptions updated

#### **Code**

- ✅ go.mod module path updated
- ✅ All import statements across entire codebase
- ✅ Docker Compose configuration
- ✅ Dockerfile с новым VERSION=2.1.0

#### **WebUI**

- ✅ All HTML page titles: "AIGateway Platform"
- ✅ Navigation labels и headers
- ✅ About System page
- ✅ Footer copyright

---

### Why Rebranding?

**Reasons for transition to "AIGateway":**

1. **Expanded Scope**: No longer just an Ollama proxy - now supports multiple model providers (vLLM, future: OpenAI, Anthropic)

2. **RAG System**: Built-in RAG capabilities make it more than a proxy

3. **Enterprise Positioning**: "Gateway" better represents the platform''s role as AI infrastructure

4. **Scalability**: Name allows for future expansion to cloud providers and custom models

---

### 🔄 Migration Required

**This is a BREAKING release.** Existing deployments need migration.

See [MIGRATION_GUIDE_v2.1.0.md](docs/MIGRATION_GUIDE_v2.1.0.md) for detailed migration steps.

**Quick Migration Checklist:**

- Update environment variables: PROXY_* → AIGATEWAY_*
- Update Docker image names
- Rename database file (optional): proxy.db → aigateway.db
- Update any scripts/configs referencing old names
- Pull new Docker images: aigateway:2.1.0

---

### Technical

- **Module path**: aigateway (was ollama-openai-proxy)
- **Import paths**: Updated throughout codebase
- **Docker volumes**: aigateway_data, aigateway_logs
- **Functional changes**: Zero - pure rebranding release
- **Compilation**: Verified ✅'
WHERE version = '2.1.0';
    