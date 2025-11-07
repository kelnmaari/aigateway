# RPC Tools Architecture Implementation (v2.5.4)

## 📋 Overview

Реализация **RPC Tools Pattern** для выполнения agent tools на стороне Desktop Client вместо сервера. Это правильная архитектура, аналогичная LSP (Language Server Protocol) и Zed IDE.

## ✅ Implemented Components

### Phase 1: Protocol & Models ✅
**Server:**
- `internal/models/tool_rpc.go` - Complete RPC protocol
  - `ToolExecutionRequest`/`Response`
  - `ToolApprovalRequest`/`Response`
  - Tool-specific params/results (FileRead, FileList, FileWrite, FileDelete, TerminalExecute)
  - WebSocket message type constants

### Phase 2: Server RPC Client ✅
**Server:**
- `internal/services/agent/tool_rpc_client.go` - RPC client for sending tool requests
  - `ExecuteTool()` - Send request to client, wait for response with timeout
  - `HandleResponse()` - Process responses from client
  - Pending request tracking with timeout
  - Cancellation support

- `internal/services/agent/conversational_agent.go` - Integration
  - `SetRPCClient()` method
  - `act()` updated to use RPC client if available
  - Fallback to local tools if RPC client not set

### Phase 3: Client Tool Executor ✅
**Desktop Client:**
- `aigateway-desktop/internal/tools/executor.go` - Tool execution engine
  - `ToolExecutor` struct with baseDir security
  - `file.read` - Read files with line range support
  - `file.list` - List directory contents
  - `file.write` - Write/append files
  - `file.delete` - Delete files/directories
  - `terminal.execute` - Execute shell commands with timeout
  - Security: Path validation against baseDir

### Phase 4: Client WebSocket Integration ✅
**Desktop Client:**
- `aigateway-desktop/internal/tools/tool_handler.go` - RPC request handler
  - `ToolRPCHandler` - Process incoming tool requests
  - Async execution with approval flow
  - Risk assessment (low/medium/high)
  - Response generation

- `aigateway-desktop/app.go` - Wails integration
  - `HandleToolExecutionRequest()` - Exposed to JavaScript
  - `showToolApprovalDialog()` - User approval dialog
  - Tool executor initialization with baseDir
  - Runtime event emission for responses

- `aigateway-desktop/frontend/src/lib/api/websocket.js` - WebSocket client
  - `_handleToolExecutionRequest()` - Handle incoming requests
  - `_setupToolRPCListener()` - Listen for Go → JS events
  - Forward requests to Go backend
  - Send responses back to server

### Phase 5: Approval System ✅
**Desktop Client:**
- Wails `MessageDialog` for user approval
- Risk levels: LOW (read/list), MEDIUM (write), HIGH (delete/terminal)
- Risk warnings with specific details
- Allow/Deny buttons with safe defaults

## ✅ Server WebSocket Integration - COMPLETED!

### Implementation Details

**File:** `internal/websocket/chat_handler.go`

**Implemented:**
1. ✅ Added `toolRPCClients map[string]*agent.ToolRPCClient` to ChatHandler
2. ✅ Created `getOrCreateToolRPCClient(client)` method
3. ✅ Integrated in `processAgentRequest()`:
   ```go
   rpcClient := h.getOrCreateToolRPCClient(client)
   customAgent.SetRPCClient(rpcClient)
   ```

4. ✅ Added `HandleToolExecutionResponse(client, message)` method
5. ✅ Added `CleanupToolRPCClient(clientID)` for disconnect cleanup

**File:** `internal/websocket/handler.go`

**Implemented:**
1. ✅ Updated `ChatHandlerInterface` with RPC methods:
   ```go
   type ChatHandlerInterface interface {
       HandleChatRequest(client *Client, message []byte)
       HandleToolExecutionResponse(client *Client, message []byte)
       CleanupToolRPCClient(clientID string)
   }
   ```

2. ✅ Added message routing in `handleClientMessage()`:
   ```go
   case models.WSMessageTypeToolExecutionResponse:
       if h.chatHandler != nil {
           h.chatHandler.HandleToolExecutionResponse(client, message)
       }
   ```

3. ✅ Added cleanup in `readPump()` defer:
   ```go
   defer func() {
       if h.chatHandler != nil {
           h.chatHandler.CleanupToolRPCClient(client.ID)
       }
       h.hub.unregister <- client
       client.Conn.Close()
   }()
   ```

## 🧪 Testing Plan

### Unit Tests
- [ ] `ToolExecutor` - All file tools
- [ ] `ToolRPCClient` - Request/response handling
- [ ] `ToolRPCHandler` - Async execution
- [ ] Path security validation

### Integration Tests  
- [ ] End-to-end: Server → Client → Server
- [ ] Agent task: "List files in project root"
- [ ] Agent task: "Read README.md file"
- [ ] Approval flow: "Write to test.txt"
- [ ] Error handling: Invalid paths, timeouts
- [ ] Concurrent tool requests

### Manual Testing
1. Start server with agent support
2. Start desktop client
3. Enable agent mode in chat
4. Send: "Покажи список файлов в корне проекта"
5. Verify:
   - Tool request reaches client
   - Client executes `file.list`
   - Response returns to server
   - Agent continues ReAct loop
   - Result displayed in UI

## 📊 Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ Desktop Client (E:\golang\aigateway-desktop)                   │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Tool Executor (LOCAL FS)                                 │  │
│  │  - file.read() → E:\golang\aigateway-desktop\README.md  │  │
│  │  - file.list() → Lists client files                     │  │
│  │  - file.write() → Writes to client FS                   │  │
│  │  - file.delete() → Deletes from client FS               │  │
│  │  - terminal.execute() → Runs on client machine          │  │
│  └──────────────────────────────────────────────────────────┘  │
│               ↕ ToolRPCHandler (approval flow)                  │
│               ↕ Wails app.go (HandleToolExecutionRequest)       │
│               ↕ WebSocket Client (tool_rpc:send events)         │
└─────────────────────────────────────────────────────────────────┘
                             ↕ WebSocket (JSON RPC)
┌─────────────────────────────────────────────────────────────────┐
│ Server (E:\golang\my_projects)                                 │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Agent Brain (ReAct Loop)                                 │  │
│  │  1. Think (LLM) → "need to list files"                  │  │
│  │  2. Act → rpcClient.ExecuteTool("file.list", {...})     │  │
│  │  3. Wait for RPC response ← {success: true, files: [...]}│  │
│  │  4. Observe → Analyze result                            │  │
│  │  5. Loop → Next action or complete                      │  │
│  └──────────────────────────────────────────────────────────┘  │
│               ↕ ToolRPCClient (request tracking)                │
│               ↕ WebSocket Handler (message routing)             │
└─────────────────────────────────────────────────────────────────┘
```

## 🔒 Security Features

1. **Path Validation**: All file operations validated against `baseDir`
2. **Approval Required**: Dangerous tools (write/delete/terminal) require explicit user approval
3. **Risk Assessment**: Tools categorized by risk level
4. **Timeout Protection**: All tool executions have timeout (default 60s)
5. **Error Handling**: Graceful degradation on failures
6. **Audit Logging**: All tool executions logged on both sides

## 📈 Performance Considerations

- **Async Execution**: Tool requests don't block WebSocket receiver
- **Request Tracking**: Efficient map-based pending request tracking
- **Timeout Management**: Automatic cleanup of timed-out requests
- **Connection Pooling**: WebSocket reused for multiple tool calls
- **Streaming Support**: Future: Stream large file reads

## 🚀 Future Enhancements (v2.6+)

1. **Streaming Tools**: Stream large file reads chunk-by-chunk
2. **Tool Progress**: Report progress for long-running tools
3. **Tool Cancellation**: UI button to cancel running tools
4. **Tool History**: Track all tool executions per conversation
5. **Custom Tools**: Allow users to define custom tools
6. **MCP Integration**: Support Model Context Protocol tools
7. **Multi-Client**: Support multiple desktop clients
8. **Tool Permissions**: Per-tool permission settings

## 📝 Notes

- **baseDir Update**: When user opens project, update toolExecutor.baseDir
- **Working Directory**: Automatically set from ProjectManager
- **Approval Caching**: Consider caching approval decisions for repeated tools
- **Error Recovery**: Agent continues on tool failure (doesn't crash)
- **Backwards Compatible**: Fallback to server-side tools if RPC client not available

## 🎓 References

- Zed IDE source code: `crates/agent/src/tools/`
- LSP Specification: https://microsoft.github.io/language-server-protocol/
- Best Practices: Keep tools simple, deterministic, and fast
- Security: Never trust client input, validate everything

---

## ✅ IMPLEMENTATION COMPLETE!

**Status**: 🎉 **100% Complete** - Full RPC Tools Architecture implemented and compiling successfully!

**Components:**
- ✅ Server Protocol & Models (`internal/models/tool_rpc.go`)
- ✅ Server RPC Client (`internal/services/agent/tool_rpc_client.go`)
- ✅ Server WebSocket Integration (`internal/websocket/chat_handler.go`, `internal/websocket/handler.go`)
- ✅ Desktop Tool Executor (`aigateway-desktop/internal/tools/executor.go`)
- ✅ Desktop RPC Handler (`aigateway-desktop/internal/tools/tool_handler.go`)
- ✅ Desktop Wails Integration (`aigateway-desktop/app.go`)
- ✅ Desktop WebSocket Client (`aigateway-desktop/frontend/src/lib/api/websocket.js`)
- ✅ Approval System (Wails MessageDialog with risk assessment)
- ✅ Both server and desktop client compile without errors

**Next Step**: End-to-end testing with real agent tasks  
**Testing Scenarios**:
1. "Покажи список файлов в корне проекта" - file.list
2. "Прочитай README.md" - file.read
3. "Создай файл test.txt" - file.write (requires approval)
4. "Удали файл test.txt" - file.delete (requires approval)
5. "Выполни команду 'ls'" - terminal.execute (requires approval)

**Key Features Delivered**:
- 🔐 **Client-side tool execution** - Tools run on desktop client's file system
- 🔒 **Security** - Path validation against baseDir, approval for dangerous tools
- ⚡ **Performance** - Async execution, timeout protection, efficient request tracking
- 🔄 **Reliability** - Graceful error handling, automatic cleanup on disconnect
- 📊 **Observability** - Comprehensive logging on both server and client sides

