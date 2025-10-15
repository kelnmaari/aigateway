# 📚 Documentation Index - Ollama-OpenAI Proxy v1.9.3

Полная документация проекта

---

## 🚀 Getting Started

### For New Users

1. **[Quick Start Guide](QUICK_START.md)** ⭐
   - Установка за 5 минут
   - Первый запуск
   - Bootstrap admin user
   - Первый chat
   - Создание API key

### For Administrators

2. **[WebUI Guide](WEBUI_GUIDE.md)**
   - Полный мануал по WebUI
   - Chat interface
   - Multi-tenancy & Teams
   - Admin panel
   - Performance monitoring

3. **[Configuration Guide](CONFIGURATION.md)**
   - Все параметры конфигурации
   - SQLite vs PostgreSQL
   - Production setup
   - Docker deployment
   - Environment variables

---

## 📖 Reference Documentation

### API & Integration

4. **[API Documentation](API_DOCUMENTATION.md)**
   - OpenAI-compatible endpoints
   - REST API reference
   - Authentication
   - Request/Response examples
   - Error codes

5. **[Authentication API](AUTH_API.md)**
   - JWT authentication
   - User management
   - Tenant management
   - API keys
   - RBAC

### Architecture & Design

6. **[Architecture](ARCHITECTURE.md)**
   - System overview
   - Component design
   - Data flow
   - Technology stack
   - Design patterns

7. **[Performance Guide](PERFORMANCE.md)**
   - Performance tuning
   - GPU optimization
   - Connection pooling
   - Monitoring & metrics
   - Benchmarks

---

## 🔧 Troubleshooting & Maintenance

8. **[Troubleshooting Guide](TROUBLESHOOTING.md)** ⭐
   - Common issues & solutions
   - Server problems
   - Database issues
   - Authentication errors
   - Build & deployment

---

## 📊 Feature Documentation

### Version 1.9.3 (Current)

**Key Features:**
- ✅ ChatGPT-like WebUI
- ✅ Multi-Tenancy with RBAC
- ✅ Dynamic Model Parameters
- ✅ Context Tracking & Auto-Summarization
- ✅ MoniGo Performance Dashboard
- ✅ NVIDIA GPU Monitoring (Linux/macOS)
- ✅ OpenTelemetry Distributed Tracing

**Documentation:**
- [WebUI Guide](WEBUI_GUIDE.md) - Sections: Chat, Parameters, Context
- [Configuration](CONFIGURATION.md) - Sections: Performance, GPU Monitoring
- [Quick Start](QUICK_START.md) - Section: First Chat

### Upcoming Features

**v1.10.0 - Smart Chat & Content**
- Vision OCR (Multimodal Chat)
- Web Content Fetcher
- File Upload (PDF, DOCX, TXT)
- Conversation Export/Import
- WebSocket Real-time Updates

See [Roadmap.MD](../Roadmap.MD) for full plan.

---

## 🎯 Quick Reference

### Common Tasks

| Task | Documentation | Section |
|------|---------------|---------|
| Install & Run | [Quick Start](QUICK_START.md) | Installation |
| Create User | [Quick Start](QUICK_START.md) | Bootstrap Admin |
| Start Chatting | [WebUI Guide](WEBUI_GUIDE.md) | Chat Interface |
| Create API Key | [WebUI Guide](WEBUI_GUIDE.md) | API Keys |
| Add Team Member | [WebUI Guide](WEBUI_GUIDE.md) | Tenants & Teams |
| Configure Server | [Configuration](CONFIGURATION.md) | Server Settings |
| Setup Production | [Configuration](CONFIGURATION.md) | Production Config |
| Monitor Performance | [WebUI Guide](WEBUI_GUIDE.md) | Admin Panel |
| Fix Issues | [Troubleshooting](TROUBLESHOOTING.md) | All Sections |

### API Quick Links

| Endpoint | Documentation |
|----------|---------------|
| `POST /v1/chat/completions` | [API Docs](API_DOCUMENTATION.md#post-v1chatcompletions) |
| `GET /v1/models` | [API Docs](API_DOCUMENTATION.md#get-v1models) |
| `POST /api/auth/register` | [Auth API](AUTH_API.md#post-apiauthregister) |
| `POST /api/auth/login` | [Auth API](AUTH_API.md#post-apiauthlogin) |
| `GET /api/gpu/metrics` | [WebUI Guide](WEBUI_GUIDE.md#gpu-monitoring) |

---

## 📝 Documentation Standards

### File Naming

- `*.md` - Markdown format
- UPPERCASE for main docs (e.g., `README.md`)
- lowercase_snake_case для technical docs

### Structure

All documentation follows this structure:
```markdown
# 📄 Title - Version

Brief description

---

## 📋 Table of Contents

---

## Sections...

---

## Related Documentation

---

**Version:** X.Y.Z
**Last Updated:** YYYY-MM-DD
```

### Update Policy

- ✅ Update docs при каждом major/minor release
- ✅ Version badge в header каждого файла
- ✅ Last Updated date в footer
- ✅ Cross-references между документами
- ✅ Code examples актуальны и протестированы

---

## 🔗 External Resources

### Official Links

- **GitHub Repository:** https://github.com/yourusername/ollama-openai-proxy
- **Issue Tracker:** https://github.com/yourusername/ollama-openai-proxy/issues
- **Discussions:** https://github.com/yourusername/ollama-openai-proxy/discussions

### Related Projects

- **Ollama:** https://ollama.ai/ - Local LLM platform
- **OpenAI API:** https://platform.openai.com/docs - API standard
- **MoniGo:** https://github.com/iyashjayesh/monigo - Performance monitoring

### Community

- **Discord:** (coming soon)
- **Reddit:** (coming soon)
- **Stack Overflow:** Tag `ollama-openai-proxy`

---

## 📞 Support

### Getting Help

1. **Check Documentation** - Start here! 📚
2. **Search Issues** - Maybe already answered
3. **Ask in Discussions** - Community support
4. **Open Issue** - For bugs/features

### Reporting Bugs

When reporting issues, include:
- Version: `cat VERSION`
- OS: `uname -a` or `ver`
- Config snippet (без секретов!)
- Error logs
- Steps to reproduce

### Feature Requests

Use GitHub Discussions:
- Describe use case
- Explain benefits
- Suggest implementation (optional)

---

## 🤝 Contributing

Want to improve documentation?

1. Fork repository
2. Edit `docs/*.md`
3. Follow [Documentation Standards](#documentation-standards)
4. Submit Pull Request

**Documentation PRs welcome!** 📝

---

## 📜 License

Documentation is part of Ollama-OpenAI Proxy project:
- **License:** MIT
- **See:** [LICENSE](../LICENSE)

---

**Documentation Version:** 1.9.3  
**Last Updated:** 2025-10-14  
**Maintained by:** Ollama-OpenAI Proxy Team

---

<div align="center">

**[⬆ Back to Top](#-documentation-index---ollama-openai-proxy-v193)**

Made with ❤️ for the Open Source Community

</div>

