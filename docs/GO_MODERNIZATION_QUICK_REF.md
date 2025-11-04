# Go 1.18-1.25 Modernization - Quick Reference

## 🎯 Цель v2.4.9
Модернизация кода с использованием современных возможностей Go:
- **-500 строк кода**
- **+High type safety**
- **+Very High readability**
- **+Go 1.25 new features**: WaitGroup.Go, FlightRecorder, CSRF protection

---

## 📋 Checklist

### Core Generics (Go 1.18-1.24)
- [ ] **Priority 1**: Generic Pointer Helpers (`internal/utils/ptr.go`)
- [ ] **Priority 2**: `slices` package integration (~30 файлов)
- [ ] **Priority 3**: `maps` package integration (~15 файлов)
- [ ] **Priority 4**: Generic API Response wrapper (~40 handlers)
- [ ] **Priority 5**: Custom Iterators (Go 1.23)
- [ ] **Priority 6**: Generic Type Aliases (Go 1.24)
- [ ] **Priority 7**: Generic Vector Operations (RAG)

### Go 1.25 Features
- [ ] **Priority 8**: `sync.WaitGroup.Go()` refactoring
- [ ] **Priority 9**: `runtime/trace.FlightRecorder` для production debugging
- [ ] **Priority 10**: `net/http.CrossOriginProtection` CSRF middleware
- [ ] **Priority 11**: `testing.T.Attr()` для test metadata
- [ ] **Priority 12**: GreenTea GC benchmarking

### Documentation
- [ ] **Priority 13**: Update all documentation

---

## 🔧 Quick Patterns

### Generic Pointer Helpers
```go
// Before:
temp := float64Ptr(0.7)
val := intValue(ptr, 0)

// After:
import "aigateway/internal/utils"
temp := utils.Ptr(0.7)
val := utils.Value(ptr, 0)
```

### slices Package
```go
// Before:
func contains(list []string, item string) bool {
    for _, v := range list { ... }
}

// After:
import "slices"
exists := slices.Contains(list, item)
filtered := slices.DeleteFunc(items, predicate)
idx := slices.IndexFunc(items, finder)
```

### maps Package
```go
// Before:
dst := make(map[string]string)
for k, v := range src { dst[k] = v }

// After:
import "maps"
dst := maps.Clone(src)
maps.Copy(dst, overrides)
```

### Generic API Response
```go
// Before:
c.JSON(200, gin.H{"data": models})

// After:
import "aigateway/internal/api/response"
c.JSON(200, response.Success(models))
```

### Custom Iterators (Go 1.23)
```go
// Before:
for offset := 0; ; offset += 100 {
    users, err := db.List(ctx, 100, offset)
    if len(users) == 0 { break }
    for _, u := range users { process(u) }
}

// After:
for user := range db.IterateUsers(ctx, 100) {
    process(user)
}
```

### Generic Type Aliases (Go 1.24)
```go
// Before:
type Config struct {
    StringMap map[string]string
    IntMap    map[string]int
}

// After:
import "aigateway/internal/types"
type Config struct {
    StringMap types.StringConfig
    IntMap    types.IntConfig
}
```

### sync.WaitGroup.Go() (Go 1.25)
```go
// Before:
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    process()
}()
wg.Wait()

// After:
var wg sync.WaitGroup
wg.Go(func() { process() })
wg.Wait()
```

### FlightRecorder (Go 1.25)
```go
import "runtime/trace"

recorder, _ := trace.NewFlightRecorder(trace.FlightRecorderConfig{
    Period: 10 * time.Second,
})

// При панике:
defer func() {
    if r := recover(); r != nil {
        f, _ := os.Create("panic-trace.out")
        recorder.WriteTo(f)
        panic(r)
    }
}()
```

### CSRF Protection (Go 1.25)
```go
import "net/http"

csrf := http.CrossOriginProtection{
    AllowOrigin: []string{"https://example.com"},
}
r.Use(func(c *gin.Context) {
    if err := csrf.Check(c.Request); err != nil {
        c.AbortWithStatus(403)
        return
    }
    c.Next()
})
```

---

## 📊 Impact Matrix

| Feature | LOC Δ | Type Safety | Readability | Time |
|---------|-------|-------------|-------------|------|
| **Core Generics** |
| Ptr Helpers | -150 | ⬆️⬆️ | ⬆️⬆️⬆️ | 2-3h |
| slices | -150 | ⬆️ | ⬆️⬆️ | 3-4h |
| maps | -80 | ⬆️⬆️ | ⬆️ | 2h |
| API Response | -40 | ⬆️⬆️⬆️ | ⬆️⬆️⬆️ | 4-5h |
| Iterators | +100 | ⬆️ | ⬆️⬆️⬆️ | 2-3h |
| Type Aliases | +30 | ⬆️⬆️ | ⬆️⬆️ | 1-2h |
| Vector Ops | +80 | ⬆️⬆️ | ⬆️⬆️ | 2h |
| **Go 1.25 Features** |
| WaitGroup.Go | -50 | ⬆️ | ⬆️⬆️ | 1-2h |
| FlightRecorder | +100 | - | ⬆️⬆️⬆️ | 2-3h |
| CSRF | +60 | ⬆️⬆️ | ⬆️ | 1-2h |
| Test Attrs | +200 | - | ⬆️⬆️ | 2-3h |
| GreenTea GC | - | - | - | 4h |
| **TOTAL** | **+100** | **High** | **Very High** | **~30h** |

---

## 🚀 Implementation Order

1. **Week 1**: Foundation
   - Generic Pointer Helpers
   - slices/maps integration
   
2. **Week 2**: Advanced
   - Generic API Response
   - Custom Iterators
   
3. **Week 3**: Generics & Go 1.25
   - Type Aliases + Vector Ops
   - WaitGroup.Go refactoring
   - FlightRecorder + CSRF
   - Test attributes

4. **Week 4**: Performance & Polish
   - GreenTea GC benchmarking
   - Code review & testing
   - Documentation

---

## ✅ Testing Commands

```bash
# Compile check
go build ./...

# Unit tests
go test ./...

# Race detector
go test -race ./...

# Benchmarks
go test -bench=. -benchmem ./...

# Lint
golangci-lint run
```

---

## 📚 References

- **Full Roadmap**: `docs/ROADMAP_2.4.9.md`
- **CHANGELOG**: `CHANGELOG.md` (v2.4.9 section)
- **TODOs**: 13 tasks in project TODO list
- **Go Release Notes**:
  - Go 1.18: https://go.dev/doc/go1.18
  - Go 1.21: https://go.dev/doc/go1.21
  - Go 1.23: https://go.dev/doc/go1.23
  - Go 1.24: https://go.dev/doc/go1.24
  - **Go 1.25**: https://tip.golang.org/doc/go1.25 ⭐ NEW
- **Go 1.25 Issues**:
  - [1.25.1](https://github.com/golang/go/issues?q=milestone%3AGo1.25.1)
  - [1.25.2](https://github.com/golang/go/issues?q=milestone%3AGo1.25.2)
  - [1.25.3](https://github.com/golang/go/issues?q=milestone%3AGo1.25.3)

---

## 🎓 Learning Resources

### Generics (Go 1.18)
```go
// Type parameter
func Max[T constraints.Ordered](a, b T) T { ... }

// Type constraint
type Number interface {
    ~int | ~int64 | ~float32 | ~float64
}

// Generic struct
type Stack[T any] struct {
    items []T
}
```

### Iterators (Go 1.23)
```go
// Iterator signature
func Iterator[T any]() func(func(T) bool)

// Usage with range
for item := range Iterator[string]() {
    process(item)
}
```

### Type Aliases (Go 1.24)
```go
// Generic type alias
type MyMap[K comparable, V any] = map[K]V

// Specialized alias
type StringMap[V any] = MyMap[string, V]
```

---

## ⚠️ Common Pitfalls

1. **Generic Constraints**:
   - Use `any` for unconstrained types
   - Use `comparable` for map keys
   - Use `~` for type approximation

2. **Iterator Early Exit**:
   - `yield` returns bool for early exit
   - Always check `!yield()` in loops

3. **Type Aliases**:
   - Only available in Go 1.24+
   - No runtime overhead

4. **Performance**:
   - Generics are zero-cost (compile-time)
   - No reflection overhead
   - Same performance as hand-written code

---

## 🔥 Quick Start

```bash
# 1. Checkout feature branch
git checkout -b feature/go-modernization

# 2. Start with Priority 1
mkdir -p internal/utils
touch internal/utils/ptr.go
# Implement generic pointer helpers

# 3. Run tests
go test ./internal/utils/...

# 4. Refactor usage (use IDE refactoring)
# Replace all stringPtr/intPtr/etc with utils.Ptr

# 5. Commit incremental changes
git add internal/utils/ptr.go
git commit -m "feat: add generic pointer helpers"

# 6. Continue with Priority 2-7
# ...
```

---

## 💡 Pro Tips

- Use **IDE refactoring** tools for mass replacements
- **Commit incrementally** after each priority
- Run **tests after each change** to catch regressions
- Update **benchmarks** to prove no performance loss
- Add **examples** in code comments for future reference
- Consider **backward compatibility** for public APIs

---

## 📞 Support

- **Full docs**: See `docs/ROADMAP_2.4.9.md`
- **Questions**: Check TODOs for task breakdown
- **Issues**: File issues with `go-modernization` label

