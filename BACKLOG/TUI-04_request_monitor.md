# TUI-04: Request Monitor в TUI

**Приоритет:** HIGH  
**Версия:** 1.2.0  
**Оценка:** 6-8 часов  
**Статус:** ⚠️ Partially Implemented (заглушка существует)

---

## Цель

Создать полнофункциональный Request Monitor с live таблицей активных и завершенных запросов.

---

## Current State

✅ Базовая страница "Requests" создана  
❌ Live таблица запросов (TODO)  
❌ WebSocket integration (TODO)  
❌ Детальный просмотр (TODO)

---

## Requirements

### 1. Live Request Table (3 часа)

**Data Structure:**
```go
type RequestInfo struct {
    ID          string
    Timestamp   time.Time
    Method      string
    Endpoint    string
    Status      string    // pending, success, error
    Duration    time.Duration
    Model       string
    APIKey      string    // masked
    StatusCode  int
    ErrorMsg    string    // if error
}
```

**Table Columns:**
```
ID | Time | Method | Endpoint | Model | Status | Duration | Key
```

**Sorting:**
- Default: by timestamp (newest first)
- Press `s` to cycle: time/duration/status/endpoint

**Filtering:**
- Press `f` → filter menu
- By status (all/pending/success/error)
- By endpoint
- By model

### 2. WebSocket Integration (2 часа)

**Backend:**
```go
// Broadcast request events
eventBroadcaster.BroadcastRequestStart(req)
eventBroadcaster.BroadcastRequestComplete(req, resp, duration)
eventBroadcaster.BroadcastRequestError(req, err)
```

**TUI:**
```go
// Subscribe to WebSocket updates
ws.Subscribe("request_*", func(event Event) {
    updateRequestTable(event.Data)
})
```

### 3. Detailed View (1 час)

Press `Enter` on request → show details:
```
┌─────────────────────────────────────┐
│ Request Details                     │
├─────────────────────────────────────┤
│ ID: req_abc123                      │
│ Time: 2025-10-05 10:30:45          │
│ Duration: 234ms                     │
│                                     │
│ Request:                            │
│   POST /v1/chat/completions         │
│   Model: gpt-3.5-turbo             │
│   Messages: 5                       │
│   Tools: 3                          │
│                                     │
│ Response:                           │
│   Status: 200 OK                    │
│   Tokens: 150                       │
│   Finish: stop                      │
│                                     │
│ [Press Esc to close]               │
└─────────────────────────────────────┘
```

### 4. Pagination & Limits (1 час)

- Keep last 1000 requests in memory
- Pagination (50 per page)
- `n`/`p` for next/previous page
- Auto-scroll to new requests

### 5. Export (1 час)

- Press `e` → export to CSV/JSON
- Include current filters
- Save to `logs/requests-{timestamp}.csv`

---

## Implementation

### Backend Changes:

**1. Request Tracking Middleware:**
```go
// internal/api/middleware/request_tracker.go
func RequestTracker() gin.HandlerFunc {
    return func(c *gin.Context) {
        req := &RequestInfo{
            ID: generateID(),
            Timestamp: time.Now(),
            // ...
        }
        
        // Broadcast start
        eventBroadcaster.BroadcastRequestStart(req)
        
        c.Next()
        
        // Broadcast complete
        req.Duration = time.Since(req.Timestamp)
        eventBroadcaster.BroadcastRequestComplete(req)
    }
}
```

**2. WebSocket Events:**
```go
// internal/websocket/events.go
const (
    EventTypeRequestStart    = "request_start"
    EventTypeRequestComplete = "request_complete"
    EventTypeRequestError    = "request_error"
)
```

### TUI Changes:

**Update `cmd/tui/main.go`:**
```go
// Replace placeholder Requests view
func renderRequestsScreen() string {
    table := createRequestTable(requestBuffer)
    stats := fmt.Sprintf("Total: %d | Active: %d | Errors: %d",
        totalRequests, activeRequests, errorCount)
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        titleStyle.Render("📊 Request Monitor"),
        stats,
        table,
        helpStyle.Render("↑↓: scroll | Enter: details | f: filter | s: sort | e: export | q: quit"),
    )
}
```

---

## Testing

- [ ] Table updates в real-time
- [ ] Sorting работает
- [ ] Filtering работает
- [ ] Detailed view показывает все поля
- [ ] Export создает корректный файл
- [ ] No memory leaks при длительной работе
- [ ] Pagination корректная

---

**Dependencies:**  
- Phase 12.2 (WebSocket) ✅
- Request tracking middleware (new)

**Priority in 1.2.0:** #1 (most requested feature)
