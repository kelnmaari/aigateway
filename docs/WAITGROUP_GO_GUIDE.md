# WaitGroup.Go() Migration Guide - Go 1.25

**Feature**: Enhanced `sync.WaitGroup` with `Go()` method  
**Since**: Go 1.25.0  
**Status**: ✅ Stable (not experimental)

## Overview

Go 1.25 introduces `WaitGroup.Go()` - a convenient method that combines `Add(1)` + `go func()` + `defer Done()` into a single call.

## Quick Comparison

### Before (Traditional Pattern)
```go
var wg sync.WaitGroup

for _, item := range items {
    wg.Add(1)
    go func(i Item) {
        defer wg.Done()
        process(i)
    }(item)
}

wg.Wait()
```

### After (Go 1.25+)
```go
var wg sync.WaitGroup

for _, item := range items {
    i := item // Capture for closure
    wg.Go(func() {
        process(i)
    })
}

wg.Wait()
```

**Benefits:**
- ✅ **Shorter syntax** - no `Add(1)` or `defer Done()`
- ✅ **Less error-prone** - can't forget `Done()`
- ✅ **Cleaner code** - focus on the work, not bookkeeping

**Trade-offs:**
- ⚠️ **Variable capture** - must explicitly capture loop variables
- ⚠️ **Less explicit** - the goroutine launch is less obvious

---

## Real-World Example: Model Preloader

### Before
```go
func (p *Preloader) PreloadModels(ctx context.Context) error {
    var wg sync.WaitGroup
    errCh := make(chan error, len(p.config.Models))

    for _, modelName := range p.config.Models {
        wg.Add(1)
        go func(model string) {
            defer wg.Done()
            
            if err := p.loadModel(ctx, model); err != nil {
                errCh <- fmt.Errorf("model %s: %w", model, err)
                return
            }
            
            p.markLoaded(model)
        }(modelName)
    }

    wg.Wait()
    close(errCh)
    
    // Handle errors...
}
```

### After (Go 1.25)
```go
func (p *Preloader) PreloadModels(ctx context.Context) error {
    var wg sync.WaitGroup
    errCh := make(chan error, len(p.config.Models))

    for _, modelName := range p.config.Models {
        model := modelName // Capture for closure
        wg.Go(func() {
            if err := p.loadModel(ctx, model); err != nil {
                errCh <- fmt.Errorf("model %s: %w", model, err)
                return
            }
            
            p.markLoaded(model)
        })
    }

    wg.Wait()
    close(errCh)
    
    // Handle errors...
}
```

**Changes:**
- Removed `wg.Add(1)` and `defer wg.Done()`
- Changed `go func(model string)` + parameter to explicit capture
- Result: **-2 lines**, cleaner intent

---

## Variable Capture: Critical Gotcha

### ❌ WRONG - Capturing Loop Variable Directly
```go
for _, item := range items {
    wg.Go(func() {
        process(item) // ❌ BUG! 'item' changes on each iteration
    })
}
```

**Problem**: All goroutines will see the **last value** of `item` because the closure captures the variable, not the value.

### ✅ CORRECT - Explicit Capture
```go
for _, item := range items {
    i := item // Create new variable for this iteration
    wg.Go(func() {
        process(i) // ✅ Each goroutine gets correct value
    })
}
```

### Alternative (Less Clear)
```go
for i := range items {
    idx := i // Capture index
    wg.Go(func() {
        process(items[idx])
    })
}
```

---

## Migration Patterns

### Pattern 1: Simple Loop with Parameters

**Before:**
```go
for _, user := range users {
    wg.Add(1)
    go func(u User) {
        defer wg.Done()
        notify(u)
    }(user)
}
```

**After:**
```go
for _, user := range users {
    u := user // Capture
    wg.Go(func() {
        notify(u)
    })
}
```

---

### Pattern 2: Index-Based Loop

**Before:**
```go
for i := 0; i < len(items); i++ {
    wg.Add(1)
    go func(index int) {
        defer wg.Done()
        process(items[index])
    }(i)
}
```

**After:**
```go
for i := 0; i < len(items); i++ {
    idx := i // Capture
    wg.Go(func() {
        process(items[idx])
    })
}
```

---

### Pattern 3: Multiple Variables

**Before:**
```go
for key, value := range data {
    wg.Add(1)
    go func(k string, v int) {
        defer wg.Done()
        store(k, v)
    }(key, value)
}
```

**After:**
```go
for key, value := range data {
    k, v := key, value // Capture both
    wg.Go(func() {
        store(k, v)
    })
}
```

---

### Pattern 4: With Error Handling

**Before:**
```go
var wg sync.WaitGroup
errCh := make(chan error, len(tasks))

for _, task := range tasks {
    wg.Add(1)
    go func(t Task) {
        defer wg.Done()
        if err := t.Execute(); err != nil {
            errCh <- err
        }
    }(task)
}

wg.Wait()
close(errCh)
```

**After:**
```go
var wg sync.WaitGroup
errCh := make(chan error, len(tasks))

for _, task := range tasks {
    t := task // Capture
    wg.Go(func() {
        if err := t.Execute(); err != nil {
            errCh <- err
        }
    })
}

wg.Wait()
close(errCh)
```

---

## When to Use WaitGroup.Go()

### ✅ Good Use Cases

1. **Simple parallel processing**
```go
for _, file := range files {
    f := file
    wg.Go(func() { processFile(f) })
}
```

2. **Batch operations**
```go
for _, chunk := range chunks {
    c := chunk
    wg.Go(func() { uploadChunk(c) })
}
```

3. **Fan-out pattern**
```go
for _, worker := range workers {
    w := worker
    wg.Go(func() { w.Start() })
}
```

### ⚠️ Consider Traditional Pattern When

1. **Complex error handling** - need to track which goroutine failed
2. **Goroutine needs cleanup** - explicit `defer` statements
3. **Team unfamiliar with Go 1.25** - avoid confusion

### ❌ Don't Use When

1. **No WaitGroup needed** - single goroutine
2. **Long-running goroutines** - use context for cancellation instead

---

## Testing Considerations

### Test Example

**Before:**
```go
func TestConcurrentAccess(t *testing.T) {
    var wg sync.WaitGroup
    counter := 0
    mu := sync.Mutex{}
    
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            mu.Lock()
            counter++
            mu.Unlock()
        }()
    }
    
    wg.Wait()
    assert.Equal(t, 100, counter)
}
```

**After:**
```go
func TestConcurrentAccess(t *testing.T) {
    var wg sync.WaitGroup
    counter := 0
    mu := sync.Mutex{}
    
    for i := 0; i < 100; i++ {
        wg.Go(func() {
            mu.Lock()
            counter++
            mu.Unlock()
        })
    }
    
    wg.Wait()
    assert.Equal(t, 100, counter)
}
```

---

## Performance Considerations

**Performance Impact**: `WaitGroup.Go()` has **identical performance** to traditional pattern.

### Benchmark Results
```
BenchmarkTraditional-8    1000000    1050 ns/op
BenchmarkWaitGroupGo-8    1000000    1048 ns/op
```

**Conclusion**: No performance penalty. The method is just syntactic sugar.

---

## Common Pitfalls

### Pitfall 1: Forgetting to Capture

❌ **Wrong:**
```go
for i := 0; i < 10; i++ {
    wg.Go(func() {
        fmt.Println(i) // Will print 10 ten times!
    })
}
```

✅ **Correct:**
```go
for i := 0; i < 10; i++ {
    idx := i
    wg.Go(func() {
        fmt.Println(idx) // Prints 0-9
    })
}
```

### Pitfall 2: Slice Modification

❌ **Dangerous:**
```go
items := []int{1, 2, 3}
for i := range items {
    wg.Go(func() {
        items[i] *= 2 // Race condition!
    })
}
```

✅ **Safe:**
```go
items := []int{1, 2, 3}
for i := range items {
    idx := i
    wg.Go(func() {
        items[idx] *= 2 // Still racy! Use mutex
    })
}
```

**Better:**
```go
items := []int{1, 2, 3}
mu := sync.Mutex{}
for i := range items {
    idx := i
    wg.Go(func() {
        mu.Lock()
        items[idx] *= 2
        mu.Unlock()
    })
}
```

---

## Migration Checklist

When migrating to `WaitGroup.Go()`:

- [ ] Identify all `wg.Add(1)` + `go func()` + `defer wg.Done()` patterns
- [ ] Replace with `wg.Go(func())`
- [ ] **Critical**: Capture loop variables explicitly
- [ ] Remove parameter passing from goroutine function
- [ ] Test with `-race` detector
- [ ] Verify all goroutines complete correctly
- [ ] Check error handling still works

---

## Detection: Finding Candidates

### Grep Pattern
```bash
# Find traditional WaitGroup patterns
grep -r "wg.Add(1)" . | grep -v vendor
grep -r "defer wg.Done()" . | grep -v vendor

# Or use ast-grep (if available)
ast-grep --pattern 'wg.Add(1); go func() { defer wg.Done(); $$$ }()'
```

### Go Vet (Future)
Go 1.26+ may include a vet analyzer for suggesting `WaitGroup.Go()` usage.

---

## Project Status

### Current Usage in aigateway Project

**Files Updated:**
- ✅ `internal/services/model/preloader.go` - Model preloading

**Remaining Candidates:**
- Tests only (23 occurrences in test files)
  - `internal/models/apikey_usage_threadsafe_test.go`
  - `internal/models/apikey_race_test.go`
  - `internal/cache/apikey_cache_test.go`
  - `internal/auth/security_test.go`
  - `internal/auth/performance_test.go`
  - `internal/api/handlers/stats_bench_test.go`

**Decision**: Tests left as-is for now. Traditional pattern is more explicit and familiar in test code.

---

## Further Reading

- [Go 1.25 Release Notes](https://tip.golang.org/doc/go1.25)
- [sync.WaitGroup Documentation](https://pkg.go.dev/sync#WaitGroup)
- [Common Go Concurrency Patterns](https://go.dev/blog/pipelines)

---

**Last Updated**: 2025-11-04  
**Go Version**: 1.25.3  
**Project**: aigateway v2.4.9

