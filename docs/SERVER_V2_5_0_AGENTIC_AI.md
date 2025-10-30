# Server Version 2.5.0 - Agentic AI & MCP Integration

## 🎯 Overview

**Version 2.5.0** добавляет server-side инфраструктуру для поддержки **Autonomous AI Agent** в Desktop Client. AI получает возможность планировать задачи, использовать tools (file operations, terminal commands, MCP servers) и требовать approval для опасных операций.

**Status:** 📋 Planned (0/5 tasks)  
**Estimated Time:** 28-35 hours  
**Priority:** HIGH (Required for Desktop v0.8.0)  
**Dependencies:**

- Server v2.4.3+ (WebSocket infrastructure)
- Desktop v0.5.0+ (Terminal integration)
- Desktop v0.8.0 (Agentic Mode UI)

---

## 🧠 Vision: Autonomous AI Agent

```
User: "Refactor this module and add tests"
         ↓
AI Agent (server-side):
  1. Анализирует код
  2. Планирует задачи (5 steps)
  3. Использует tools:
     - file.read() → читает файлы
     - file.write() → пишет новый код
     - terminal.execute() → запускает тесты
     - mcp.github → создает PR (опционально)
  4. Требует approval для file.write(), terminal.execute()
  5. Применяет изменения после approval
         ↓
User: Reviews diff → Approves → Changes applied
```

---

## 📋 Tasks

### AGENT-01: Agent Planning & Orchestration API (v2.5.1)

**Time:** 6-8 hours

**Endpoints:**

```
POST /api/agent/plan
Request:  {task: "Refactor module", context: {...}}
Response: {plan: [{step: 1, action: "file.read", ...}], estimated_tools: 5}
```

**Features:**

- Agent session management (sessionID → AgentState)
- Task decomposition logic
- Task tree structure (parent → subtasks)
- Integration with Ollama for planning

**Models:**

- `AgentSession`, `AgentPlan`, `AgentTask`, `AgentStep`

---

### AGENT-02: Tool Execution Framework (v2.5.2)

**Time:** 8-10 hours

**Endpoints:**

```
POST /api/agent/tools/execute
Request:  {tool: "file.read", parameters: {path: "..."}}
Response: {success: true, result: {...}, execution_time_ms: 50}
```

**Built-in Tools:**

- `file.read(path)` - read file
- `file.write(path, content)` - write/overwrite
- `file.create(path, content)` - create new
- `file.delete(path)` - delete file
- `file.list(directory, recursive)` - list files
- `file.search(pattern, directory)` - search files

**Security:**

- Path validation (prevent `../../../etc/passwd`)
- File size limits (max 10 MB for read)
- Rate limiting (100 calls/min)

**Models:**

- `Tool`, `ToolExecution`, `ToolResult`, `ToolError`

---

### AGENT-03: Terminal Tool Integration (v2.5.3)

**Time:** 6-8 hours

**Features:**

- `terminal.execute(command, cwd, timeout)`
- Real-time output streaming (WebSocket)
- Process management (kill on timeout/cancel)
- Exit code capture

**Security:**

- **Dangerous command detection:**
  - `rm -rf`, `sudo`, `format`, `dd`, `mkfs`
  - `curl | bash`, `wget | sh`
  - `git push`, `git commit`
  - → Require explicit approval
- **Safe commands (auto-approve):**
  - `ls`, `cat`, `echo`, `pwd`, `which`
- **Sandbox (optional):**
  - Restrict to project directory

**Shell Support:**

- bash, powershell, zsh
- Auto-detect based on OS

**Models:**

- `TerminalExecution`, `ProcessInfo`, `CommandOutput`

---

### AGENT-04: MCP Server Integration & Tool Catalog (v2.5.4)

**Time:** 6-8 hours

**Endpoints:**

```
GET  /api/mcp/catalog
POST /api/mcp/invoke
```

**Features:**

- List available MCP servers
- Query MCP tools and schemas
- Invoke MCP tools
- Enable/Disable MCP servers per user/tenant
- Rate limiting per MCP server (100 calls/min)

**Security:**

- Whitelist only MCP servers from our catalog
- Validate tool parameters against schema
- Dangerous tool warnings

**Models:**

- `MCPServer`, `MCPTool`, `MCPToolInvocation`, `MCPToolResult`

---

### AGENT-05: Approval Flow & Rollback System (v2.5.5)

**Time:** 8-10 hours

**Endpoints:**

```
POST /api/agent/approval/request
POST /api/agent/approval/:id/approve
POST /api/agent/approval/:id/deny
POST /api/agent/rollback
GET  /api/agent/approval/history
```

**Approval Rules:**

| Level | Operations | Behavior |
|-------|-----------|----------|
| 🔴 **RED** | File: delete, overwrite<br>Terminal: sudo, rm -rf<br>Git: push, commit | **Require approval** |
| 🟡 **YELLOW** | File: create, read<br>Terminal: safe commands | **Auto-approve + log** |
| 🟢 **GREEN** | File: read<br>Terminal: ls, pwd | **Always allowed** |

**Rollback System:**

- File snapshots before modification (git-like)
- Store in `/tmp/agent-snapshots/{session_id}/{file_hash}`
- TTL 24 hours
- Undo stack (max 50 operations per session)

**Models:**

- `ApprovalRequest`, `ApprovalRule`, `FileSnapshot`, `RollbackOperation`

---

## 🏗️ Architecture

### Backend Structure

```go
internal/
  agent/
    planner.go          // Task planning & decomposition
    orchestrator.go     // Agent session management
    tools/
      registry.go       // Tool registry
      filesystem.go     // File operations
      terminal.go       // Terminal execution
      mcp.go           // MCP integration
    approval/
      manager.go        // Approval flow
      rules.go         // Approval rules
    rollback/
      snapshots.go      // File snapshots
      undo.go          // Undo operations
  models/
    agent.go           // Agent models
    tools.go           // Tool models
    approval.go        // Approval models
```

### API Endpoints Summary

```
POST   /api/agent/plan                    # Task planning
POST   /api/agent/tools/execute           # Execute tool
GET    /api/mcp/catalog                   # List MCP servers
POST   /api/mcp/invoke                    # Invoke MCP tool
POST   /api/agent/approval/request        # Request approval
POST   /api/agent/approval/:id/approve    # Approve action
POST   /api/agent/approval/:id/deny       # Deny action
POST   /api/agent/rollback                # Rollback changes
GET    /api/agent/approval/history        # Approval history
```

### WebSocket Messages

```json
// Agent → Client
{
  "type": "agent_thinking",
  "session_id": "...",
  "payload": {
    "current_step": 2,
    "total_steps": 5,
    "message": "Reading file src/main.go..."
  }
}

{
  "type": "agent_tool_use",
  "session_id": "...",
  "payload": {
    "tool": "file.read",
    "parameters": {"path": "src/main.go"}
  }
}

{
  "type": "agent_approval_needed",
  "session_id": "...",
  "payload": {
    "approval_id": "...",
    "tool": "file.write",
    "reason": "Overwriting existing file",
    "diff": "+45 lines, -12 lines",
    "timeout_at": "2025-10-29T12:00:30Z"
  }
}

{
  "type": "agent_completed",
  "session_id": "...",
  "payload": {
    "success": true,
    "tools_used": 8,
    "files_modified": 3,
    "summary": "Refactoring completed successfully"
  }
}
```

---

## 💾 Database Schema

### New Tables

```sql
-- Agent sessions
CREATE TABLE agent_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    task TEXT NOT NULL,
    plan JSONB,
    status VARCHAR(50) DEFAULT 'active',
    tools_used INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Tool executions
CREATE TABLE tool_executions (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES agent_sessions(id),
    tool_name VARCHAR(100) NOT NULL,
    parameters JSONB,
    result JSONB,
    success BOOLEAN,
    execution_time_ms INTEGER,
    executed_at TIMESTAMP DEFAULT NOW()
);

-- Approval requests
CREATE TABLE approval_requests (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES agent_sessions(id),
    tool_name VARCHAR(100) NOT NULL,
    parameters JSONB,
    reason TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    approved_by UUID REFERENCES users(id),
    timeout_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP
);

-- File snapshots (for rollback)
CREATE TABLE file_snapshots (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES agent_sessions(id),
    file_path TEXT NOT NULL,
    content_hash VARCHAR(64),
    snapshot_path TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP
);

-- MCP tool invocations
CREATE TABLE mcp_tool_invocations (
    id UUID PRIMARY KEY,
    session_id UUID REFERENCES agent_sessions(id),
    user_id UUID NOT NULL REFERENCES users(id),
    mcp_server VARCHAR(100) NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    parameters JSONB,
    result JSONB,
    success BOOLEAN,
    execution_time_ms INTEGER,
    invoked_at TIMESTAMP DEFAULT NOW()
);
```

### Indexes

```sql
CREATE INDEX idx_agent_sessions_user ON agent_sessions(user_id);
CREATE INDEX idx_agent_sessions_status ON agent_sessions(status);
CREATE INDEX idx_tool_executions_session ON tool_executions(session_id);
CREATE INDEX idx_approval_requests_session ON approval_requests(session_id);
CREATE INDEX idx_approval_requests_status ON approval_requests(status);
CREATE INDEX idx_file_snapshots_session ON file_snapshots(session_id);
CREATE INDEX idx_mcp_tool_invocations_session ON mcp_tool_invocations(session_id);
CREATE INDEX idx_mcp_tool_invocations_user ON mcp_tool_invocations(user_id);
```

---

## 📈 Use Cases

### 1. Code Refactoring

```
User: "Refactor UserService to use dependency injection"

Agent:
  1. file.read(UserService.go)           → Reads code
  2. Analyzes dependencies                → Plans refactoring
  3. Generates new code with DI           → Creates diff
  4. file.write(UserService.go) [APPROVE] → User reviews diff
  5. User approves                        → Applies changes
  6. terminal.execute(go test)            → Runs tests
  7. Success! ✅
```

### 2. Bug Fix Automation

```
User: "Fix panic in handleRequest function"

Agent:
  1. file.read(handler.go)                → Reads code
  2. Identifies panic location            → Proposes fix
  3. file.write(handler.go) [APPROVE]     → User approves
  4. file.create(handler_test.go)         → Adds test
  5. terminal.execute(go test ./...)      → Runs tests
  6. git.commit [APPROVE]                 → User approves
  7. Success! ✅
```

### 3. Project Setup

```
User: "Initialize new Go project with tests"

Agent:
  1. file.create(main.go)                 → Creates entry point
  2. file.create(go.mod)                  → Creates module
  3. file.create(main_test.go)            → Adds tests
  4. terminal.execute(go mod tidy)        → Installs deps
  5. terminal.execute(go test ./...)      → Runs tests
  6. Success! ✅
```

### 4. MCP Integration

```
User: "Create GitHub issue for this bug"

Agent:
  1. Analyzes bug details                 → Extracts info
  2. mcp.invoke(github.create_issue)      → Creates issue
  3. Returns issue URL                    → https://github.com/...
  4. Success! ✅
```

---

## 🛡️ Security

### Critical Security Measures

| Measure | Implementation |
|---------|---------------|
| **Path Validation** | Prevent `../../../etc/passwd` |
| **Command Whitelist** | Block `rm -rf`, `sudo`, dangerous ops |
| **Approval Flow** | Require user confirmation for destructive ops |
| **Rate Limiting** | Max 100 tools/min per session |
| **MCP Whitelist** | Only trusted MCP servers from catalog |
| **Audit Trail** | Log all tool executions and approvals |
| **Rollback** | Snapshots before file modifications |
| **Timeout** | 30s approval timeout → auto-deny |

### Dangerous Command Patterns

```bash
# BLOCKED (require approval):
rm -rf
sudo 
format
dd if=
mkfs
> /dev/sda
curl | bash
wget | sh
git push
git commit

# SAFE (auto-approve):
ls
cat
echo
pwd
which
grep
find
```

---

## 📊 Performance Requirements

| Metric | Target |
|--------|--------|
| Agent planning time | < 2 seconds |
| Tool execution avg | < 5 seconds |
| File read (1MB) | < 100ms |
| Terminal command spawn | < 50ms |
| WebSocket message latency | < 50ms |
| Approval timeout | 30 seconds |
| Rollback operation | < 200ms |
| Database query avg | < 10ms |

---

## ✅ Acceptance Criteria

- ✅ Agent планирует tasks из 5+ steps
- ✅ File operations работают (read, write, delete)
- ✅ Terminal commands выполняются safely
- ✅ MCP tools доступны через catalog
- ✅ Approval flow требует confirmation для dangerous ops
- ✅ Rollback восстанавливает files correctly
- ✅ WebSocket messages в real-time
- ✅ Rate limiting работает (100 tools/min)
- ✅ Database migrations без ошибок
- ✅ All endpoints authenticated
- ✅ Path validation prevents directory traversal
- ✅ Dangerous commands detected and blocked

---

## 🔄 Integration with Desktop Client

**Desktop Client v0.8.0 will use:**

```javascript
// 1. Start agent session
POST /api/agent/plan
{
  "task": "Refactor UserService",
  "context": {
    "current_file": "internal/user/service.go",
    "project_path": "/home/user/myproject"
  }
}

// 2. Execute tools (via WebSocket)
WebSocket → agent_tool_use
{
  "type": "agent_tool_use",
  "tool": "file.read",
  "parameters": {"path": "internal/user/service.go"}
}

// 3. Handle approval requests
WebSocket → agent_approval_needed
{
  "type": "agent_approval_needed",
  "approval_id": "abc123",
  "tool": "file.write",
  "diff": "+45 lines, -12 lines"
}

// Desktop shows approval dialog
POST /api/agent/approval/abc123/approve

// 4. Receive completion
WebSocket → agent_completed
{
  "type": "agent_completed",
  "success": true,
  "summary": "Refactoring complete"
}
```

---

## ⚠️ Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Agent executes malicious command** | Critical | Approval flow + command whitelist |
| **Agent infinite loop** | High | Max 50 tools, timeout 5 min |
| **File corruption** | Medium | Snapshots + rollback |
| **MCP server compromise** | High | Whitelist only our catalog |
| **Approval timeout abuse** | Medium | 30s → auto-deny |
| **Resource exhaustion** | Medium | Rate limiting, max concurrent sessions |

---

## 🎯 Success Metrics

### Functionality

- ✅ Agent completes 95%+ of valid tasks
- ✅ Zero false positives for dangerous command detection
- ✅ Rollback успешен в 100% случаев
- ✅ MCP tool success rate > 98%

### Performance

- ✅ Agent planning < 2 seconds
- ✅ Tool execution avg < 5 seconds
- ✅ UI remains responsive

### User Experience

- ✅ Approval flow < 3 clicks
- ✅ Progress visible in real-time
- ✅ Agent "explains" actions
- ✅ Errors clearly communicated

---

## 📝 Implementation Notes

### Phase 1: Core Agent (AGENT-01, AGENT-02)

- Setup agent session management
- Implement file operations tools
- Basic task planning

### Phase 2: Terminal & MCP (AGENT-03, AGENT-04)

- Add terminal execution
- Integrate MCP catalog
- Security hardening

### Phase 3: Approval & Rollback (AGENT-05)

- Implement approval flow
- File snapshots system
- Rollback mechanism

### Phase 4: Testing & Polish

- Comprehensive testing
- Performance optimization
- Documentation

---

## 🔗 Related Documentation

- [Roadmap.MD](../Roadmap.MD) - Server roadmap v2.5.0
- [Roadmap-desktop.md](../Roadmap-desktop.md) - Desktop client v0.8.0 (Agentic Mode)
- [DESKTOP_ROADMAP_ADDITIONS.md](DESKTOP_ROADMAP_ADDITIONS.md) - Desktop agentic features

---

**Version:** 2.5.0  
**Status:** 📋 Planned  
**Target Start:** After v2.4.6  
**Estimated Completion:** 28-35 hours (4-5 weeks)  
**Priority:** HIGH  
**Required For:** Desktop v0.8.0 Agentic Mode

Let's build the most powerful AI agent system! 🤖🚀
