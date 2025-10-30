# Desktop Roadmap - New Features Added (2025-10-29)

## 📋 Summary of Additions

Based on user feedback, the following features have been added to [Roadmap-desktop.md](../Roadmap-desktop.md):

---

## 🐛 1. Version 0.4.1 - Critical Bug Fix

### CLIENT-030: Models Loading Bug

**Issue:** После свежей установки desktop app нужно перезапустить приложение, чтобы модели загрузились.

**Root Cause:** Race condition между successful authentication и models loading.

**Solution:**
- Добавить explicit `loadModels()` после successful auth
- Retry механизм для models loading (3 attempts)
- Loading state в UI (spinner в model dropdown)
- Fallback на первую available модель если `selectedModel` empty

**Priority:** 🔴 CRITICAL  
**Time:** ~3 days  
**Status:** IN PROGRESS

**Testing:**
```bash
# 1. Удалить config
rm ~/.aigateway/config.json

# 2. Запустить app
./build/bin/aigateway-desktop.exe

# 3. Login
# 4. ✅ Models должны загрузиться СРАЗУ (no restart needed)
```

---

## 🔍 2. Version 0.5.0 - LSP/Linter Integration

### CLIENT-029: Language Server Protocol

**Feature:** Real-time code diagnostics и quick fixes через LSP.

**Supported Languages:**
- ✅ **Go**: `gopls`
- ✅ **Python**: `pyright` / `pylance`
- ✅ **TypeScript/JavaScript**: Built-in Monaco support
- ✅ **Rust**: `rust-analyzer` (optional)

**Features:**
- 🔍 **Real-time Diagnostics**: Errors/Warnings в editor gutter
- 💡 **Quick Fixes**: Lightbulb menu с auto-import, fix typos, etc.
- 📝 **Code Actions**: Refactoring suggestions
- ⚙️ **Configurable**: Enable/Disable per language в Settings

**Dependencies:**
- Requires external LSP binaries (gopls, pyright, etc.)
- Auto-detect installed LSPs или prompt user to install

**Note:** Это опциональная feature - editor работает без LSP.

---

## 🤖 3. Version 0.8.0 - Agentic Mode (BIGGEST FEATURE)

### Overview

AI становится автономным агентом, способным:
- 🎯 **Task Planning** - разбивает задачи на подзадачи
- 🔧 **Tool Use** - file operations, terminal commands, MCP servers
- ✅ **Approval Flow** - пользователь подтверждает опасные действия
- 📊 **Progress Tracking** - показывает что делает агент
- ↩️ **Rollback** - откат изменений

### Architecture

```
User Request → AI Agent (server) → Task Planning
                ↓
        Tool Selection & Execution
        ↓        ↓        ↓
   File Ops  Terminal  MCP Tools
        ↓        ↓        ↓
    Execution Results
        ↓
User Approval (if needed)
        ↓
Apply Changes + Rollback
```

### CLIENT-028: Agentic Mode Framework

**28.1. Agent Communication Protocol:**
- WebSocket/SSE для agent streaming
- Message types:
  - `agent_thinking` - planning phase
  - `agent_tool_use` - запрос на tool
  - `agent_tool_result` - результат
  - `agent_approval_needed` - требуется confirm
  - `agent_completed` - task done

**28.2. Task Planning UI:**
- Agent thinking indicator (animated 🧠)
- Task decomposition tree view
- Progress tracking (N/M tasks completed)

**28.3. Tool Execution UI:**
```
┌────────────────────────────────────┐
│ 🔧 File Operation                  │
│ Action: Write file                 │
│ Path: src/main.go                  │
│ Changes: +45 lines, -12 lines      │
│ [ View Diff ] [ Approve ] [ Deny ] │
└────────────────────────────────────┘
```

**28.4. Approval Flow:**

**Require approval:**
- ❌ File deletion/overwrite
- ❌ Terminal: `sudo`, `rm -rf`, `format`, `dd`
- ❌ Network: `curl`, `wget`
- ❌ Git: `commit`, `push`

**Auto-approve:**
- ✅ File read operations
- ✅ Non-destructive commands: `ls`, `cat`, `echo`
- ✅ MCP tool queries (if whitelisted)

**28.5. Rollback Mechanism:**
- Snapshot files before modification (git-like)
- Undo last N operations (stack-based)
- "Restore Previous Version" button
- Diff viewer для preview

**28.6. Agent Settings:**
- Enable/Disable agentic mode toggle
- Auto-approve preferences (per tool type)
- Max tool use per task (limit 50)
- Timeout settings

---

### CLIENT-031: MCP Servers Support

**31.1. MCP Catalog Browser:**
- GET /api/mcp/catalog → список available MCP servers

```
┌────────────────────────────────────┐
│ 🔌 GitHub MCP Server               │
│ Description: GitHub API integration│
│ Tools: 12 available                │
│ Status: ● Enabled                  │
│ [ Configure ] [ Disable ]          │
└────────────────────────────────────┘
```

**31.2. MCP Tool Management:**
- Browse tools per MCP server
- Tool schema display (input/output)
- Enable/Disable tools individually
- Usage statistics

**31.3. MCP Tool Execution:**
```json
POST /api/agent/tools/execute
{
  "tool": "github.create_issue",
  "mcp_server": "github-mcp",
  "parameters": {
    "repo": "user/repo",
    "title": "Bug report",
    "body": "Description..."
  }
}
```

**31.4. Security:**
- ✅ Whitelist MCP servers (only from our catalog)
- ✅ Rate limiting per MCP server (max 100 calls/min)
- ✅ Tool execution logs (audit trail)
- ✅ Dangerous tool warnings

---

### CLIENT-032: Tool System

**File Operations:**
- `file.read(path)`
- `file.write(path, content)`
- `file.create(path, content)`
- `file.delete(path)` - requires approval
- `file.list(directory)`
- `file.search(pattern, directory)`

**Terminal Execution:**
- `terminal.execute(command, cwd)`
- Output streaming (real-time)
- Exit code capture
- Timeout handling (kill after 60s)
- Dangerous command detection

**Tool Registry:**
- Central registry на server
- Tool versioning (v1, v2)
- Dynamic tool loading

---

### CLIENT-033: Agent Progress & History

**Progress Visualization:**
- Agent task tree (expandable/collapsible)
- Real-time progress updates
- Time elapsed per task
- Success/Failure indicators (✅ ❌)

**Agent History:**
- Conversation with agent actions
- "Agent used 5 tools" summary
- Expandable tool use details
- Export agent session

**Agent Metrics:**
- Total tools used per session
- Success rate (tasks completed / tasks attempted)
- Average approval time
- Most used tools (charts)

---

## 📊 Updated Timeline

```
✅ v0.1.0 (MVP)               - 2 weeks  - COMPLETED
✅ v0.2.0 (File Integration)  - 2 weeks  - COMPLETED
✅ v0.3.0 (Monaco Editor)     - 3 weeks  - COMPLETED
✅ v0.4.0 (WebSocket)         - 2 weeks  - COMPLETED
⏳ v0.4.1 (Hotfixes)          - 3 days   - IN PROGRESS
📋 v0.5.0 (Terminal & LSP)    - 3 weeks  - PLANNED
📋 v0.6.0 (Offline Mode)      - 2 weeks  - PLANNED
📋 v0.7.0 (Packaging)         - 2 weeks  - PLANNED
🤖 v0.8.0 (Agentic Mode)      - 4-5 weeks - PLANNED (BIGGEST FEATURE)
🎯 v1.0.0 (Production)        - 6-8 months total
```

---

## ⚠️ Server Dependencies

**New Server Requirements for Agentic Mode (v2.5.0+):**

| Feature | Endpoint | Description |
|---------|----------|-------------|
| Agent Task Planning | POST /api/agent/plan | Task decomposition |
| Agent Tool Execution | POST /api/agent/tools/execute | Execute tool |
| MCP Catalog | GET /api/mcp/catalog | List MCP servers |
| MCP Tool Invoke | POST /api/mcp/invoke | Invoke MCP tool |
| File Operations | POST /api/agent/tools/file | File read/write/delete |
| Terminal Tool | POST /api/agent/tools/terminal | Execute commands |

**Note:** Desktop v0.8.0 REQUIRES server v2.5.0+

---

## 🛡️ Security Considerations

### Critical Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Agent executes dangerous commands** | **Critical** | **Approval flow + command whitelist** |
| **Agent infinite loop** | **High** | **Max 50 tools, timeout per task** |
| **MCP server compromised** | **High** | **Whitelist only our catalog** |
| **File operations corrupt data** | **Medium** | **Rollback mechanism, snapshots** |

### Approval Flow Design

**Philosophy:** "Trust but verify"

- 🔴 **Red**: Requires explicit approval (destructive operations)
- 🟡 **Yellow**: Auto-approve with logging (safe operations)
- 🟢 **Green**: Always allowed (read-only)

### Dangerous Command Detection

**Patterns to block:**
```bash
rm -rf
sudo 
format
dd
mkfs
> /dev/sda
wget | bash
curl | sh
```

**Require approval for:**
- Any command with `sudo`
- Any command with wildcards + delete (`rm *.*)
- Git operations (`git push`, `git commit`)
- Network downloads (`curl`, `wget`)

---

## 🎯 Success Metrics for v0.8.0

### Performance
- ✅ Agent response time < 2 seconds (planning phase)
- ✅ Tool execution < 5 seconds average
- ✅ UI remains responsive during agent work
- ✅ Memory < 250 MB with agent mode active

### Reliability
- ✅ Agent completes 95%+ of valid tasks
- ✅ Zero false positives for dangerous command detection
- ✅ Rollback успешен в 100% случаев
- ✅ MCP tool execution success rate > 98%

### User Experience
- ✅ Approval flow intuitive (< 3 clicks)
- ✅ Progress visible в real-time
- ✅ Agent "explains" what it's doing
- ✅ Errors clearly communicated

---

## 🚀 Implementation Priority

**Immediate (v0.4.1):**
1. ⏳ Fix models loading bug (CLIENT-030) - 3 days

**Short-term (v0.5.0):**
2. 🎯 Terminal integration (CLIENT-017) - 1 week
3. 🔍 LSP integration (CLIENT-029) - 2 weeks

**Medium-term (v0.8.0):**
4. 🤖 Agent framework (CLIENT-028) - 2 weeks
5. 🔌 MCP support (CLIENT-031) - 1 week
6. 🔧 Tool system (CLIENT-032) - 1 week
7. 📊 Progress UI (CLIENT-033) - 3-4 days

**Long-term (v1.0.0):**
8. 🎁 Packaging & release - 2 weeks

---

## 📝 Notes

### Why Agentic Mode is Important

Modern AI IDEs (Cursor, Windsurf, Codeium) все имеют agentic capabilities:

1. **Autonomous Coding** - AI может писать код без постоянного input
2. **Multi-step Tasks** - "Refactor this module" → agent выполняет 10+ операций
3. **Tool Use** - AI использует file system, terminal, external APIs
4. **Context Awareness** - Agent понимает project structure

**Our Advantage:**
- 🌐 **Server-side Agent** - более мощные модели (не limited by client resources)
- 🔌 **MCP Integration** - доступ к нашему каталогу инструментов
- 🔐 **Security First** - approval flow встроен изначально
- 💾 **Cloud Sync** - agent history синхронизируется между devices

### LSP Integration Benefits

- 📝 **Instant Feedback** - ошибки видны до запуска кода
- 💡 **Smart Suggestions** - auto-import, quick fixes
- 🚀 **Productivity** - меньше переключений в terminal для проверки
- 🎓 **Learning** - видишь best practices в real-time

**Compared to Competitors:**
- VS Code: Built-in LSP support ✅
- Cursor: Built-in LSP support ✅
- **Our App**: Will have LSP support! ✅

---

## 🎉 Conclusion

With these additions, AIGateway Desktop будет иметь:

✅ **All Core Features** (Chat, Files, Editor) - DONE  
✅ **Real-time Streaming** (WebSocket) - DONE  
🔧 **Terminal Integration** (v0.5.0) - PLANNED  
🔍 **LSP/Linter** (v0.5.0) - PLANNED  
🤖 **Full Agentic Mode** (v0.8.0) - PLANNED  
🔌 **MCP Integration** (v0.8.0) - PLANNED  

**We'll be competitive with:**
- Cursor (agentic mode ✅, LSP ✅)
- Windsurf (autonomous coding ✅)
- Codeium (tool use ✅, MCP ✅)

**Unique advantages:**
- Server-side agent (more powerful)
- MCP catalog integration
- Multi-device sync
- Security-first design

---

**Status:** 2025-10-29  
**Current Version:** v0.4.0 ✅  
**Next Release:** v0.4.1 (Hotfix) ⏳  
**Target:** v1.0.0 with Full Agent Capabilities 🎯

Let's build the best AI desktop client! 🚀🤖

