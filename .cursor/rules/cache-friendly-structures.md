# Cache-Friendly Data Structures Rule

## Цель

Автоматически применять CPU cache optimization patterns при создании новых структур данных с concurrent access.

## Когда применять

Эта rule активируется при:
- Создании новых `struct` типов с atomic fields
- Добавлении counters/metrics structures
- Рефакторинге существующих структур с concurrent access

## False Sharing Detection

### Признаки false sharing:

1. **Multiple atomic fields в одной структуре**
```go
type Stats struct {
    Counter1 int64  // ⚠️ Potential false sharing
    Counter2 int64  // ⚠️ If updated concurrently
    Counter3 int64
}
```

2. **Структура используется глобально** (`var Global = &Struct{}`)

3. **Методы используют `atomic.AddInt64` / `atomic.LoadInt64`**

4. **Benchmark показывает деградацию при `-cpu=16`**

## Автоматические исправления

### Pattern 1: Padded Atomic Counters

**Trigger:** Структура содержит 2+ `int64` fields с atomic operations

**Before:**
```go
type Metrics struct {
    RequestCount int64
    ErrorCount   int64
}

func (m *Metrics) IncrementRequests() {
    atomic.AddInt64(&m.RequestCount, 1)
}
```

**After (auto-suggest):**
```go
type Metrics struct {
    RequestCount int64
    _pad1        [56]byte  // Cache line padding
    
    ErrorCount int64
    _pad2      [56]byte
}

func (m *Metrics) IncrementRequests() {
    atomic.AddInt64(&m.RequestCount, 1)
}
```

### Pattern 2: Hot/Cold Data Split

**Trigger:** Структура содержит >5 fields, некоторые используются часто, другие редко

**Before:**
```go
type User struct {
    ID          string    // HOT
    Status      string    // HOT
    Name        string    // COLD
    Email       string    // COLD
    CreatedAt   time.Time // COLD
    LoginCount  int64     // HOT
}
```

**After (auto-suggest):**
```go
type UserHot struct {
    ID         string
    Status     string
    LoginCount int64
    _pad       [40]byte  // Padding to cache line if needed
    
    Cold *UserCold
}

type UserCold struct {
    Name      string
    Email     string
    CreatedAt time.Time
}
```

### Pattern 3: Atomic Operations Enforcement

**Trigger:** `int64` field обновляется в concurrent context без atomic

**Before (❌ ПЛОХО):**
```go
type Counter struct {
    value int64
}

func (c *Counter) Increment() {
    c.value++  // ⚠️ Race condition!
}
```

**After (auto-suggest):**
```go
type Counter struct {
    value int64
    _pad  [56]byte  // If part of larger struct with other atomics
}

func (c *Counter) Increment() {
    atomic.AddInt64(&c.value, 1)  // ✅ Thread-safe
}

func (c *Counter) Get() int64 {
    return atomic.LoadInt64(&c.value)
}
```

## Decision Tree

```
Новая структура данных?
│
├─ Содержит atomic operations?
│  ├─ YES → Нужно ли padding?
│  │  ├─ Global singleton? → YES, добавь padding
│  │  ├─ High update rate (>100/s)? → YES, добавь padding
│  │  ├─ Multiple atomic fields? → YES, добавь padding
│  │  └─ Single atomic field? → NO padding needed
│  │
│  └─ NO → Проверь на race conditions
│
├─ Содержит hot + cold data?
│  ├─ YES → Рассмотри hot/cold split
│  │  ├─ Структура часто читается? → YES, split
│  │  ├─ >1000 instances? → Взвесь memory cost
│  │  └─ <100 instances? → Split acceptable
│  │
│  └─ NO → OK as is
│
└─ Используется конкурентно?
   ├─ YES → Обязательно `go test -race`
   └─ NO → Standard struct OK
```

## Code Templates

### Template 1: Padded Global Stats

```go
// Auto-generated from template: padded-global-stats
type [Name]Stats struct {
    [Counter1Name] int64
    _pad1          [56]byte
    
    [Counter2Name] int64
    _pad2          [56]byte
    
    // Add more counters with padding...
}

func (s *[Name]Stats) Increment[Counter1]() {
    atomic.AddInt64(&s.[Counter1Name], 1)
}

func (s *[Name]Stats) Snapshot() [Name]StatsSnapshot {
    return [Name]StatsSnapshot{
        [Counter1]: atomic.LoadInt64(&s.[Counter1Name]),
        [Counter2]: atomic.LoadInt64(&s.[Counter2Name]),
    }
}

type [Name]StatsSnapshot struct {
    [Counter1] int64 `json:"[counter1_json]"`
    [Counter2] int64 `json:"[counter2_json]"`
}
```

### Template 2: Hot/Cold Split

```go
// Auto-generated from template: hot-cold-split
type [Name]Hot struct {
    // Hot fields (accessed frequently)
    [HotField1] [Type1]
    [HotField2] [Type2]
    
    _pad [CalculatedPadding]byte  // Align to cache line
    
    Cold *[Name]Cold
}

type [Name]Cold struct {
    // Cold fields (accessed rarely)
    [ColdField1] [Type1]
    [ColdField2] [Type2]
}

func Convert[Name]ToHotCold(obj *[Name]) (*[Name]Hot, *[Name]Cold) {
    cold := &[Name]Cold{
        [ColdField1]: obj.[ColdField1],
        [ColdField2]: obj.[ColdField2],
    }
    
    hot := &[Name]Hot{
        [HotField1]: obj.[HotField1],
        [HotField2]: obj.[HotField2],
        Cold:        cold,
    }
    
    return hot, cold
}
```

### Template 3: Concurrent Counter Collection

```go
// Auto-generated from template: concurrent-counters
type [Name]Counters struct {
    [Counter1Name] int64
    _pad1          [56]byte
    
    [Counter2Name] int64
    _pad2          [56]byte
    
    [Counter3Name] int64
    _pad3          [56]byte
}

func New[Name]Counters() *[Name]Counters {
    return &[Name]Counters{}
}

func (c *[Name]Counters) Increment[Counter1]() {
    atomic.AddInt64(&c.[Counter1Name], 1)
}

func (c *[Name]Counters) Add[Counter1](delta int64) {
    atomic.AddInt64(&c.[Counter1Name], delta)
}

func (c *[Name]Counters) Get[Counter1]() int64 {
    return atomic.LoadInt64(&c.[Counter1Name])
}

func (c *[Name]Counters) Snapshot() [Name]CountersSnapshot {
    return [Name]CountersSnapshot{
        [Counter1]: atomic.LoadInt64(&c.[Counter1Name]),
        [Counter2]: atomic.LoadInt64(&c.[Counter2Name]),
        [Counter3]: atomic.LoadInt64(&c.[Counter3Name]),
    }
}
```

## Auto-generated Tests

При создании optimized structure автоматически генерируй:

### Race Detection Test

```go
func Test[Name]_RaceCondition(t *testing.T) {
    s := New[Name]()
    
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 1000; j++ {
                s.IncrementCounter1()
                s.IncrementCounter2()
            }
        }()
    }
    
    wg.Wait()
    
    snapshot := s.Snapshot()
    expected := int64(100 * 1000)
    
    if snapshot.Counter1 != expected {
        t.Errorf("Counter1 = %d, want %d", snapshot.Counter1, expected)
    }
}
```

### Benchmark Test

```go
func Benchmark[Name]_Sequential(b *testing.B) {
    s := New[Name]()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        s.IncrementCounter1()
    }
}

func Benchmark[Name]_Parallel(b *testing.B) {
    s := New[Name]()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            s.IncrementCounter1()
        }
    })
}

// Run with: go test -bench=Benchmark[Name] -cpu=1,2,4,8,16 -benchtime=10s
```

## Linting Rules

Автоматически проверяй:

1. **Atomic field без padding** в struct с 2+ atomic fields
   ```
   WARNING: Field 'Counter1' should have cache line padding
   Suggestion: Add '_pad1 [56]byte' after this field
   ```

2. **Non-atomic concurrent update**
   ```
   ERROR: Field 'value' updated without atomic operation
   Suggestion: Use atomic.AddInt64(&s.value, 1)
   ```

3. **Mixed hot/cold data** (>10 fields, некоторые часто используются)
   ```
   INFO: Consider hot/cold data split for better cache locality
   Hot fields: ID, Status, Counter
   Cold fields: Description, CreatedAt, Metadata
   ```

4. **Missing race test**
   ```
   WARNING: Concurrent structure missing race detection test
   Suggestion: Add Test[Name]_RaceCondition(t *testing.T)
   ```

## Memory Budget Guidelines

При применении padding проверяй memory impact:

| Scenario | Instances | Overhead | Acceptable? |
|----------|-----------|----------|-------------|
| Global singleton | 1 | 256 bytes | ✅ Always |
| Per-user counters | <1,000 | <256 KB | ✅ Yes |
| Per-user counters | 1,000-10,000 | 256 KB - 2.5 MB | ⚠️ Consider |
| Per-user counters | >10,000 | >2.5 MB | ❌ Reconsider |
| Per-request temp | N/A | N/A | ❌ No padding |

## Quick Reference

### Cache Line Sizes
- x86_64: 64 bytes
- ARM64: 64 bytes (обычно)
- RISC-V: 64 bytes (обычно)

### Padding Calculation
```go
// For int64 (8 bytes) to fill 64-byte cache line:
_pad [56]byte  // 64 - 8 = 56

// For struct already 32 bytes:
_pad [32]byte  // 64 - 32 = 32

// Always align to 64 bytes
```

### Performance Expectations

| Optimization | Typical Speedup | Use Case |
|--------------|-----------------|----------|
| Cache line padding | 5-10x | High contention global counters |
| Hot/Cold split | 3-5x | Frequent validation/lookup |
| Atomic operations | N/A (correctness) | Any concurrent access |

## Examples in Project

Refer to:
- `internal/api/handlers/stats_optimized.go` - Padded global stats
- `internal/models/apikey_optimized.go` - Hot/cold split + counters
- `docs/CACHE_OPTIMIZATION.md` - Full documentation

## Further Reading

- [CPU Cache-Friendly Go](https://skoredin.pro/blog/golang/cpu-cache-friendly-go)
- [False Sharing](https://mechanical-sympathy.blogspot.com/2011/07/false-sharing.html)
- [Go Memory Model](https://go.dev/ref/mem)

