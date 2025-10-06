# 📊 Test Coverage Tracking

> **MVP Commitment**: 100% test coverage for all critical components

**Last Updated**: 2025-10-05  
**Target**: v1.2.1 - 100% Test Coverage Release

---

## 📈 Current Coverage Status

| Package | Current | Target | Status | Priority |
|---------|---------|--------|--------|----------|
| **internal/auth** | n/a | n/a | ✅ | - |
| **internal/auth/apikey** | 42.5% | 95%+ | 🔴 | HIGH |
| **internal/auth/middleware** | 45.8% | 95%+ | 🔴 | HIGH |
| **internal/converter** | 10.1% | 95%+ | 🔴 | CRITICAL |
| **internal/metrics** | 45.7% | 95%+ | 🔴 | HIGH |
| **internal/api/handlers** | 0.0% | 95%+ | 🔴 | CRITICAL |
| **internal/request** | 0.0% | 90%+ | 🔴 | MEDIUM |
| **internal/websocket** | 0.0% | 90%+ | 🔴 | MEDIUM |
| **internal/storage** | ~60%* | 95%+ | 🟡 | HIGH |
| **internal/optimizer** | 0.0% | 80%+ | 🟡 | LOW |

\* Estimated based on JSON storage tests

---

## 🎯 Coverage Goals by Component

### PRIORITY 1: CRITICAL (Week 1)

#### ✅ 1. Mock Storage Fix

**Status**: COMPLETED ✅  
**Coverage**: N/A (infrastructure)  
**Files**: `internal/storage/memory_storage.go`

- [x] Add `EnableAPIKey()` method to MemoryStorage
- [x] Add `EnableAPIKey()` method to ErrorStorage
- [x] Update `RevokeAPIKey()` signature with reason parameter
- [x] All tests compile and pass

---

#### 🔨 2. API Handlers Tests

**Status**: IN PROGRESS 🔨  
**Current Coverage**: 0.0%  
**Target Coverage**: 95%+  
**Files**: `internal/api/handlers/*.go`

##### Chat Handler (`chat.go`) ✅ COMPLETED

- [x] Test: Valid chat completion request → success
- [x] Test: Invalid JSON → validation error  
- [x] Test: Ollama server down → 503 error
- [x] Test: Tools/function calling → correct conversion
- [x] Test: Streaming enabled → handled

**Coverage**: ~60%+ (5/5 tests passing)  
**File**: `internal/api/handlers/chat_test.go`

##### Models Handler (`models.go`) ✅ COMPLETED

- [x] Test: List models → success with correct format
- [x] Test: Ollama models → OpenAI format conversion
- [x] Test: Ollama unavailable → error handling

**Coverage**: ~70%+ (3/3 tests passing)  
**File**: `internal/api/handlers/models_test.go`

##### Embeddings Handler (`embeddings.go`) ✅ COMPLETED [NEW]

- [x] Test: Single text embedding → success
- [x] Test: Batch embeddings → correct format
- [x] Test: Invalid JSON → validation error
- [x] Test: Error handling for Ollama failures

**Coverage**: ~70%+ (4/4 tests passing)  
**File**: `internal/api/handlers/embeddings_test.go`

##### Completions Handler (`completions.go`) ✅ COMPLETED [NEW]

- [x] Test: Text completion request → converted to chat
- [x] Test: String prompt → success
- [x] Test: Array prompt → success
- [x] Test: Invalid JSON → validation error
- [x] Test: Ollama error → handled correctly

**Coverage**: ~70%+ (4/4 tests passing)  
**File**: `internal/api/handlers/completions_test.go`

##### Admin Handler (`admin.go`) ✅ COMPLETED [AUTH-04]

- [x] Test: Create API key → success with plaintext
- [x] Test: List API keys → success without hashes
- [x] Test: Get API key by ID → success
- [x] Test: Update API key → success (AUTH-04)
- [x] Test: Revoke API key → status changed (AUTH-04)
- [x] Test: Enable API key → status active (AUTH-04)
- [x] Test: Delete API key → removed
- [x] Test: Extend expiration → date updated (AUTH-04)
- [x] Test: Update permissions → persisted (AUTH-04)
- [x] Test: Without key manager → error handling
- [x] Test: Invalid JSON → validation errors
- [x] Test: Not found → error handling

**Coverage**: ~80%+ (12/12 tests passing)  
**File**: `internal/api/handlers/admin_test.go`
- [ ] Test: Get usage stats → correct data
- [ ] Test: Unauthorized access → 401
- [ ] Test: Non-admin key → 403

##### Streaming Handler (`streaming.go`)

- [ ] Test: SSE chunk formatting → correct
- [ ] Test: Tool calls in stream → proper format
- [ ] Test: Stream error handling → graceful
- [ ] Test: Client disconnect → cleanup
- [ ] Test: Multiple chunks → ordered delivery

##### Health Handler (`health.go`) ✅ COMPLETED

- [x] Test: Health check → 200 OK
- [x] Test: Ready check with Ollama → success
- [x] Test: Ready check Ollama down → 503
- [x] Test: Liveness check → always OK
- [x] Test: Integration (state transitions)
- [x] Test: Context cancellation handling

**Coverage**: ~90%+ (6/6 tests passing)  
**File**: `internal/api/handlers/health_test.go`

**Estimated Time**: 4-6 hours

---

#### 🔨 3. Converter Tests

**Status**: PENDING 🔄  
**Current Coverage**: 10.1%  
**Target Coverage**: 95%+  
**Files**: `internal/converter/*.go`

##### Chat Converter (`chat.go`)

- [x] Basic model mapping (from simple_test.go)
- [ ] Test: Full chat request conversion
- [ ] Test: System message handling
- [ ] Test: Multi-turn conversation
- [ ] Test: Temperature, top_p, max_tokens → mapped
- [ ] Test: Stop sequences → converted
- [ ] Test: Unknown model → error
- [ ] Test: Tool-aware routing → model switch

##### Streaming Converter (`streaming.go`)

- [ ] Test: Chat chunk → SSE format
- [ ] Test: Tool calls in chunks → structured
- [ ] Test: Finish reason mapping → correct
- [ ] Test: Done message → [DONE]
- [ ] Test: Error in stream → propagated

##### Response Converter (`response.go`)

- [ ] Test: Ollama response → OpenAI format
- [ ] Test: Usage stats → mapped correctly
- [ ] Test: Tool calls → correct structure
- [ ] Test: Multiple choices → handled
- [ ] Test: Error responses → proper format

##### Tools Handling (across converters)

- [ ] Test: Tools in request → forwarded to Ollama
- [ ] Test: Tool calls in response → OpenAI format
- [ ] Test: Tool choice "required" → enforced
- [ ] Test: Tool choice "auto" → default behavior
- [ ] Test: Tool choice specific function → filtered
- [ ] Test: Force usage config → applied

**Estimated Time**: 3-4 hours

---

### PRIORITY 2: HIGH (Week 2)

#### 🔨 4. Auth/APIKey Enhancement Tests

**Status**: PENDING 🔄  
**Current Coverage**: 42.5%  
**Target Coverage**: 95%+  
**Files**: `internal/auth/apikey/manager.go`

##### AUTH-04 New Methods

- [x] Basic CRUD operations (existing tests)
- [ ] Test: EnableAPIKey → status changed, event logged
- [ ] Test: Enable non-revoked key → error
- [ ] Test: ExtendExpiration → date updated, validated
- [ ] Test: Extend past max → error
- [ ] Test: UpdatePermissions → persisted, validated
- [ ] Test: UpdatePermissions invalid → error
- [ ] Test: UpdateAPIKey → partial updates work
- [ ] Test: UpdateAPIKey rate limits → validated
- [ ] Test: GetAPIKeyUsage → correct stats

##### Edge Cases

- [ ] Test: Concurrent key operations → safe
- [ ] Test: Bootstrap admin key → idempotent
- [ ] Test: Expired key operations → handled
- [ ] Test: Rate limit bypass for admin → works

**Estimated Time**: 2-3 hours

---

#### 🔨 5. Auth/Middleware Enhancement Tests

**Status**: PENDING 🔄  
**Current Coverage**: 45.8%  
**Target Coverage**: 95%+  
**Files**: `internal/auth/middleware/auth_keys.go`

##### Additional Scenarios

- [x] Basic authentication (existing tests)
- [ ] Test: Permission middleware → enforces correctly
- [ ] Test: Multiple permissions → OR logic
- [ ] Test: Model authorization → checks allowed models
- [ ] Test: Wildcard models → allows all
- [ ] Test: Record usage → increments stats
- [ ] Test: Expired key → rejected
- [ ] Test: Disabled key → rejected
- [ ] Test: Missing auth header → 401
- [ ] Test: Malformed auth header → 401
- [ ] Test: Rate limit exceeded → 429
- [ ] Test: Rate limit admin bypass → allowed

**Estimated Time**: 2 hours

---

#### 🔨 6. Metrics Tests

**Status**: PENDING 🔄  
**Current Coverage**: 45.7%  
**Target Coverage**: 95%+  
**Files**: `internal/metrics/*.go`

##### Metrics Storage (`storage.go`)

- [x] Aggregator tests (existing)
- [x] Ring buffer tests (existing)
- [ ] Test: RecordMetric → stored correctly
- [ ] Test: GetMetrics by type → filtered
- [ ] Test: GetStats → aggregated correctly
- [ ] Test: GetHistory → time range filtering
- [ ] Test: GetRecent → last N entries
- [ ] Test: Cleanup old data → retention works
- [ ] Test: Concurrent writes → thread-safe

##### Broadcaster (`broadcaster.go`)

- [ ] Test: Start/Stop → goroutines managed
- [ ] Test: Broadcast interval → fires regularly
- [ ] Test: Event creation → correct format
- [ ] Test: No clients → no panic

##### WebSocket Hub (`hub.go`)

- [ ] Test: Client connect → registered
- [ ] Test: Client disconnect → cleaned up
- [ ] Test: Broadcast message → all clients receive
- [ ] Test: Client-specific message → only target receives
- [ ] Test: Error handling → logged, no crash

**Estimated Time**: 3 hours

---

### PRIORITY 3: MEDIUM (Week 3)

#### 🔨 7. Request Storage Tests

**Status**: PENDING 🔄  
**Current Coverage**: 0.0%  
**Target Coverage**: 90%+  
**Files**: `internal/request/storage.go`, `internal/request/types.go`

- [ ] Test: StoreRequest → added to ring buffer
- [ ] Test: GetRequests → pagination works
- [ ] Test: FilterRequests by status → correct results
- [ ] Test: FilterRequests by method → correct results
- [ ] Test: SortRequests by timestamp → ordered
- [ ] Test: SortRequests by duration → ordered
- [ ] Test: Cleanup old requests → retention works
- [ ] Test: Concurrent access → thread-safe
- [ ] Test: Export to CSV → valid format
- [ ] Test: Export to JSON → valid format

**Estimated Time**: 2 hours

---

#### 🔨 8. WebSocket Tests

**Status**: PENDING 🔄  
**Current Coverage**: 0.0%  
**Target Coverage**: 90%+  
**Files**: `internal/websocket/*.go`

- [ ] Test: Hub client management → lifecycle
- [ ] Test: Event system → all event types
- [ ] Test: Broadcast → delivery guaranteed
- [ ] Test: Handler upgrade → WebSocket established
- [ ] Test: Handler errors → proper HTTP codes
- [ ] Test: Message parsing → JSON validation
- [ ] Test: Connection errors → cleanup

**Estimated Time**: 2 hours

---

#### 🔨 9. Prometheus Integration Tests

**Status**: PENDING 🔄  
**Current Coverage**: N/A (middleware)  
**Target Coverage**: 90%+  
**Files**: `internal/api/middleware/metrics_collector.go`

- [ ] Test: Metrics collected on request → counters updated
- [ ] Test: Latency recorded → histogram updated
- [ ] Test: Error tracking → error counter incremented
- [ ] Test: Active requests → gauge updated
- [ ] Test: Prometheus exposition → valid format
- [ ] Test: Label consistency → no cardinality explosion

**Estimated Time**: 1-2 hours

---

### PRIORITY 4: LOW (Optional)

#### 📝 10. Optimizer Tests

**Status**: PENDING 🔄  
**Current Coverage**: 0.0%  
**Target Coverage**: 80%+  
**Files**: `internal/optimizer/*.go`

- [ ] Test: System message simplification → applied
- [ ] Test: Tool filtering → max tools respected
- [ ] Test: Temperature auto-fix → lowered for tools
- [ ] Test: Context window optimization → history trimmed
- [ ] Test: Smart filtering enabled → works
- [ ] Test: Smart filtering disabled → bypassed

**Estimated Time**: 1 hour

---

#### 📝 11. Integration Tests

**Status**: PENDING 🔄  
**Coverage**: N/A  
**Files**: `internal/integration/*_test.go` (to create)

- [ ] Test: Full request flow → end-to-end
- [ ] Test: WebUI proxy → correct forwarding
- [ ] Test: TUI HTTP client → stats fetching
- [ ] Test: Streaming flow → complete

**Estimated Time**: 2 hours

---

## 📊 Progress Summary

### Overall Progress

- **Completed**: 1/11 tasks (9%)
- **In Progress**: 1/11 tasks (9%)
- **Pending**: 9/11 tasks (82%)

### Coverage by Priority

- **Priority 1 (Critical)**: 1/3 completed (33%)
- **Priority 2 (High)**: 0/3 completed (0%)
- **Priority 3 (Medium)**: 0/3 completed (0%)
- **Priority 4 (Low)**: 0/2 completed (0%)

### Estimated Time Remaining

- **Priority 1**: ~7-10 hours
- **Priority 2**: ~7 hours
- **Priority 3**: ~5 hours
- **Priority 4**: ~3 hours
- **TOTAL**: ~22-25 hours

---

## 🎯 Milestones

- [ ] **Milestone 1**: Priority 1 complete (v1.2.1-alpha)
- [ ] **Milestone 2**: Priority 2 complete (v1.2.1-beta)
- [ ] **Milestone 3**: Priority 3 complete (v1.2.1-rc1)
- [ ] **Milestone 4**: Priority 4 complete (v1.2.1 release)
- [ ] **Final Goal**: 95%+ overall coverage ✅

---

## 📝 Notes

### Testing Philosophy

1. **Critical paths first**: Auth, API handlers, converters
2. **Edge cases matter**: Always test error scenarios
3. **Real-world scenarios**: Use actual request/response examples
4. **Concurrency safety**: Test with race detector
5. **Mock external deps**: Ollama client should be mockable

### Test Organization

- Unit tests: `*_test.go` in same package
- Integration tests: `internal/integration/`
- Benchmarks: `*_bench_test.go`
- Test helpers: `internal/test/helpers.go`

### CI/CD Integration

```bash
# Run all tests with coverage
make test-coverage

# Run only fast tests
make test-fast

# Check coverage threshold
make test-coverage-check
```

---

**Last Run**: 2025-10-05  
**Command**: `go test ./internal/... -cover -short`

```
✅ internal/auth              : [no statements]
⚠️  internal/auth/apikey       :  42.5%
⚠️  internal/auth/middleware   :  45.8%
⚠️  internal/converter         :  10.1%
⚠️  internal/metrics           :  45.7%
❌ internal/api/handlers      :   0.0% [no tests]
```
