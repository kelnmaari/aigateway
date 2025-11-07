INSERT INTO changelogs (version, release_date, content) VALUES
('2.5.0', '2025-11-04', '## [2.5.0] - 2025-11-04

### Added

- **🤖 Agentic AI System** (AGENT-01 through AGENT-05, CLIENT-028 through CLIENT-033):
  - **Agent Task Planning** (Iteration 1):
    - `internal/models/agent.go` - AgentSession, AgentPlan, AgentStep, AgentEvent models
    - `internal/services/agent/service.go` - Agent service with task decomposition
    - `internal/api/handlers/agent.go` - REST API for agent operations
    - API Endpoints: `POST /api/admin/agent/plan`, `GET /api/admin/agent/sessions/:id`, `POST /api/admin/agent/sessions/:id/cancel`
    - Desktop: `AgentPanel.svelte` - Real-time progress tracking UI
  - **Tool Execution Framework** (Iteration 2):
    - `internal/services/agent/tools/registry.go` - Tool registry with discovery and validation
    - `internal/services/agent/tools/file_tools.go` - 4 file operation tools (read, write, list, delete)
    - `internal/services/agent/tools/terminal_tool.go` - Shell command execution tool
    - API: `POST /api/admin/agent/sessions/:id/execute/:step` - Execute individual steps
    - Total: 5 production-ready tools
  - **Approval Flow & Safety** (Iteration 3):
    - `internal/services/agent/approval.go` - ApprovalManager for dangerous operations
    - Dangerous tool detection with user confirmation workflow
    - Approval timeout (5 minutes), approval status tracking
    - API: `POST /api/admin/agent/approvals/:id` - Approve/reject pending operations
    - Desktop: Approval Dialog UI with safety indicators, dangerous operation badges
  - **MCP Tools Integration** (Iteration 4):
    - `internal/services/agent/tools/mcp_tool.go` - MCP client stub implementation
    - Tool discovery from MCP servers (foundation for future protocol implementation)
    - Auto-registration helper for MCP tools
    - MCP Catalog already available via existing API
  - **WebSocket Real-Time Events** (Iteration 5):
    - `internal/websocket/events.go` - 13 Agent event types added
    - EventBroadcaster methods for agent lifecycle events
    - Integration: Agent Service → WebSocket → Web clients
    - Event types: session_created, planning_started, step_started, approval_needed, progress_update, etc.
    - Desktop polling via REST API (WebSocket optional for simplicity)

- **Desktop Client Enhancements** (aigateway-desktop):
  - `internal/http/agent.go` - Agent API client methods
  - `frontend/src/lib/api/agent.js` - Frontend API wrapper
  - `frontend/src/lib/components/AgentPanel.svelte` - Agent monitoring UI:
    - Task planning display with step-by-step breakdown
    - Real-time progress tracking with visual indicators
    - Tool execution results display
    - Approval dialog for dangerous operations
    - Safety badges and warning indicators
    - Session cancellation support

### Technical

- **Architecture**:
  ```
  [Agent Service] → [Tool Registry]
       ↓                  ↓
  [Planning]       [File Tools] [Terminal] [MCP]
       ↓                  ↓
  [Approval]         [Execution]
       ↓                  ↓
  [EventBroadcaster] → [WebSocket] → [Clients]
  ```
- **Statistics**:
  - Server files created: 14 (models, services, tools, handlers)
  - Desktop files modified: 5 (API, UI components)
  - Total LOC: ~4,000
  - Tools implemented: 6 (5 file/terminal + MCP foundation)
  - WebSocket events: 13 agent-specific event types
  - Test coverage: Unit tests for all core services
- **Dependencies**:
  - No new external dependencies (uses existing stack)
  - MCP integration uses stub implementation (ready for protocol extension)
- **Performance**:
  - Agent planning: ~1-5 seconds (depends on task complexity)
  - Tool execution: varies by tool (file ops ~10ms, terminal ~100-1000ms)
  - Approval timeout: 5 minutes configurable
  - WebSocket latency: ~10-50ms for event broadcasting
- **Security**:
  - Dangerous tool detection and approval workflow
  - Tool execution sandboxed to configured base directory
  - Terminal commands require explicit approval
  - All agent operations require admin role
  - Session timeout and cancellation support

### Changed

- `internal/api/router/router.go`:
  - Added Agent Service and handler initialization
  - Added WebSocket event callback for agent events
  - Added `mapAgentEventToWebSocketType` helper function
  - Integrated agent routes under `/api/admin/agent` group
- `internal/models/agent.go`:
  - Extended AgentEventType with 13 new event constants
  - Added ApprovalRequest, ApprovalStatus, ApprovalDecision models
  - Added session lifecycle and step event types
- `internal/websocket/events.go`:
  - Added Agent event type constants (13 types)
  - Added BroadcastAgent* methods (11 specialized broadcasters)
  - Integrated agent events into existing WebSocket infrastructure')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

