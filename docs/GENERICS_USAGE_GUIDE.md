# Generics Usage Guide: v2.4.9 Modernization

This guide demonstrates how to use the new generic helpers introduced in v2.4.9.

## Table of Contents

1. [Generic Pointer Helpers](#generic-pointer-helpers)
2. [slices Package Usage](#slices-package-usage)
3. [maps Package Usage](#maps-package-usage)
4. [Generic API Response Wrapper](#generic-api-response-wrapper)
5. [Best Practices](#best-practices)
6. [Migration Guide](#migration-guide)

---

## Generic Pointer Helpers

**Package**: `internal/utils`

### utils.Ptr[T]()

Create a pointer from a value inline.

```go
import "aigateway/internal/utils"

// Before
temp := 0.7
config := &Config{
    Temperature: &temp,
}

// After ✅
config := &Config{
    Temperature: utils.Ptr(0.7),
}
```

**When to use**: Struct literals, function calls requiring pointer arguments.

---

### utils.Value[T]()

Extract value from pointer with default.

```go
import "aigateway/internal/utils"

// Before
func getTemp(config *Config) float64 {
    if config.Temperature != nil {
        return *config.Temperature
    }
    return 0.5
}

// After ✅
func getTemp(config *Config) float64 {
    return utils.Value(config.Temperature, 0.5)
}
```

**When to use**: Safe pointer dereferencing with fallback values.

---

### utils.PtrOrNil[T]()

Create pointer only for non-zero values.

```go
import "aigateway/internal/utils"

// Before
var namePtr *string
if name != "" {
    namePtr = &name
}
record := Record{Name: namePtr}

// After ✅
record := Record{
    Name: utils.PtrOrNil(name), // nil if name == ""
}
```

**When to use**: Optional fields where zero values should be omitted.

---

### utils.Deref[T]()

Safe dereference with zero value fallback.

```go
import "aigateway/internal/utils"

// Before
var count int
if countPtr != nil {
    count = *countPtr
}

// After ✅
count := utils.Deref(countPtr) // Returns 0 if nil
```

**When to use**: When zero value is acceptable default.

---

### utils.Equal[T]()

Compare pointers for equality (handles nil).

```go
import "aigateway/internal/utils"

// Before
func areEqual(a, b *int) bool {
    if a == nil && b == nil {
        return true
    }
    if a == nil || b == nil {
        return false
    }
    return *a == *b
}

// After ✅
func areEqual(a, b *int) bool {
    return utils.Equal(a, b)
}
```

**When to use**: Pointer comparison in validation, tests.

---

### utils.Clone[T]()

Create independent copy of pointed value.

```go
import "aigateway/internal/utils"

original := utils.Ptr(42)
clone := utils.Clone(original)
*clone = 99
// original still points to 42
```

**When to use**: Deep copying pointers, concurrent access patterns.

---

## slices Package Usage

**Package**: `slices` (Go 1.21+)

### slices.Contains()

Check if slice contains element.

```go
import "slices"

// Before ❌
func hasPermission(perms []string, target string) bool {
    for _, p := range perms {
        if p == target {
            return true
        }
    }
    return false
}

// After ✅
func hasPermission(perms []string, target string) bool {
    return slices.Contains(perms, target)
}
```

**Performance**: Same or better than manual loop (inlined by compiler).

---

### slices.ContainsFunc()

Check with custom predicate.

```go
import "slices"

// Before ❌
func hasLargeModel(models []string) bool {
    patterns := []string{"70b", "405b"}
    for _, model := range models {
        for _, pattern := range patterns {
            if strings.Contains(model, pattern) {
                return true
            }
        }
    }
    return false
}

// After ✅
func hasLargeModel(models []string) bool {
    patterns := []string{"70b", "405b"}
    return slices.ContainsFunc(models, func(model string) bool {
        return slices.ContainsFunc(patterns, func(p string) bool {
            return strings.Contains(model, p)
        })
    })
}
```

---

### slices.Index() / slices.IndexFunc()

Find element index.

```go
import "slices"

// Before ❌
func findUserIndex(users []User, id string) int {
    for i, u := range users {
        if u.ID == id {
            return i
        }
    }
    return -1
}

// After ✅
func findUserIndex(users []User, id string) int {
    return slices.IndexFunc(users, func(u User) bool {
        return u.ID == id
    })
}
```

---

### slices.DeleteFunc()

Remove elements matching predicate.

```go
import "slices"

// Before ❌
func removeExpired(keys []APIKey) []APIKey {
    result := make([]APIKey, 0, len(keys))
    for _, k := range keys {
        if !k.IsExpired() {
            result = append(result, k)
        }
    }
    return result
}

// After ✅
func removeExpired(keys []APIKey) []APIKey {
    return slices.DeleteFunc(keys, func(k APIKey) bool {
        return k.IsExpired()
    })
}
```

---

## maps Package Usage

**Package**: `maps` (Go 1.21+)

### maps.Clone()

Create shallow copy of map.

```go
import "maps"

// Before ❌
func cloneUsage(original map[string]int64) map[string]int64 {
    result := make(map[string]int64, len(original))
    for k, v := range original {
        result[k] = v
    }
    return result
}

// After ✅
func cloneUsage(original map[string]int64) map[string]int64 {
    return maps.Clone(original)
}
```

**Benefits**: 
- Atomic metadata copy
- Clearer intent
- Thread-safe for immutable values

---

### maps.Copy()

Merge maps (modifies destination).

```go
import "maps"

// Before ❌
func mergeConfig(base, override map[string]interface{}) {
    for k, v := range override {
        base[k] = v
    }
}

// After ✅
func mergeConfig(base, override map[string]interface{}) {
    maps.Copy(base, override)
}
```

---

### maps.Equal()

Compare maps for equality.

```go
import "maps"

// Before ❌
func mapsEqual(a, b map[string]int) bool {
    if len(a) != len(b) {
        return false
    }
    for k, v := range a {
        if b[k] != v {
            return false
        }
    }
    return true
}

// After ✅
func mapsEqual(a, b map[string]int) bool {
    return maps.Equal(a, b)
}
```

---

## Generic API Response Wrapper

**Package**: `internal/api`

### Success Responses

```go
import "aigateway/internal/api"

// Single object
func GetUser(c *gin.Context) {
    user, err := findUser(userID)
    if err != nil {
        api.RespondError(c, 404, "User not found", "USER_NOT_FOUND")
        return
    }
    
    api.RespondSuccess(c, 200, user) // APIResponse[User]
}
```

**Output**:
```json
{
  "success": true,
  "data": {
    "id": "123",
    "name": "John Doe"
  }
}
```

---

### Paginated Responses

```go
import "aigateway/internal/api"

func ListUsers(c *gin.Context) {
    page := getPage(c)
    pageSize := getPageSize(c)
    
    users, total, err := fetchUsers(page, pageSize)
    if err != nil {
        api.RespondError(c, 500, "Database error", "DB_ERROR")
        return
    }
    
    pagination := api.CalculatePagination(total, page, pageSize)
    api.RespondPaginated(c, 200, users, pagination)
}
```

**Output**:
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_items": 150,
    "total_pages": 8
  }
}
```

---

### Error Responses

```go
import "aigateway/internal/api"

func ValidateInput(c *gin.Context) {
    if err := validate(input); err != nil {
        details := map[string]interface{}{
            "field": "email",
            "reason": "invalid format",
        }
        api.RespondErrorWithDetails(c, 400, "Validation failed", "VALIDATION_ERROR", details)
        return
    }
    // ...
}
```

**Output**:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed"
  },
  "details": {
    "field": "email",
    "reason": "invalid format"
  }
}
```

---

## Best Practices

### 1. Generic Constraints

Use appropriate constraints for generic functions:

```go
// ✅ Good: Comparable constraint for equality checks
func FindUnique[T comparable](items []T) []T {
    seen := make(map[T]bool)
    result := []T{}
    for _, item := range items {
        if !seen[item] {
            seen[item] = true
            result = append(result, item)
        }
    }
    return result
}

// ❌ Bad: No constraint when needed
func FindUnique[T any](items []T) []T {
    // Won't compile - can't use T as map key
}
```

---

### 2. Avoid Over-Generification

```go
// ❌ Bad: Generic when not needed
func AddInt[T int](a, b T) T {
    return a + b
}

// ✅ Good: Use concrete types for simple operations
func AddInt(a, b int) int {
    return a + b
}
```

**Rule**: Use generics when you need type safety across multiple types, not for single-type operations.

---

### 3. Document Generic Parameters

```go
// ✅ Good: Clear documentation
// Cache[K, V] provides a generic key-value cache.
// K must be comparable for use as map keys.
// V can be any type but should be immutable for thread safety.
type Cache[K comparable, V any] struct {
    items map[K]V
    mu    sync.RWMutex
}
```

---

### 4. Prefer Standard Library

```go
// ❌ Bad: Reinventing the wheel
func myContains[T comparable](slice []T, value T) bool { ... }

// ✅ Good: Use standard library
import "slices"
found := slices.Contains(items, target)
```

---

## Migration Guide

### Step 1: Identify Patterns

Search your codebase for these patterns:

```bash
# Pointer helpers
grep -r "Ptr\(.*\) \*" .

# Manual loops
grep -r "for .*, .* := range" .

# Map copying
grep -r "for k, v := range.*map" .
```

---

### Step 2: Replace Incrementally

Don't replace everything at once. Start with:
1. Most frequently used patterns
2. Files with highest test coverage
3. Performance-critical paths

---

### Step 3: Test Thoroughly

```bash
# Run tests
go test ./...

# Race detection
go test -race ./...

# Benchmarks
go test -bench=. -benchmem
```

---

### Step 4: Update Imports

```go
import (
    "maps"      // Go 1.21+
    "slices"    // Go 1.21+
    
    "aigateway/internal/api"
    "aigateway/internal/utils"
)
```

---

## Common Pitfalls

### 1. Closure Variable Capture

```go
// ❌ Bad: Loop variable captured
for _, user := range users {
    wg.Add(1)
    go func() {
        defer wg.Done()
        process(user) // Captures last user!
    }()
}

// ✅ Good: Pass as parameter
for _, user := range users {
    wg.Add(1)
    go func(u User) {
        defer wg.Done()
        process(u)
    }(user)
}
```

---

### 2. Nil Slice vs Empty Slice

```go
// ❌ Bad: Returns nil for empty result
func getItems() []Item {
    return nil // JSON: null
}

// ✅ Good: Return empty slice
func getItems() []Item {
    return []Item{} // JSON: []
}
```

---

### 3. Generic Type Inference Limitations

```go
// ❌ Bad: Compiler can't infer
result := utils.Ptr() // Error: missing type argument

// ✅ Good: Explicit type or inferred from usage
result := utils.Ptr[int](42)
// OR
result := utils.Ptr(42) // Inferred as int
```

---

## Performance Notes

### Generics Performance

- **No runtime overhead**: Generics use monomorphization (like C++ templates)
- **Binary size**: May increase slightly due to code duplication per type
- **Compile time**: Slightly longer with heavy generic usage

### Standard Library Performance

- `slices.Contains`: ~10ns/op (same as manual loop)
- `maps.Clone`: ~100ns per 1000 elements
- `utils.Ptr`: Inlined (0 overhead)

---

## Further Reading

- [Go Generics Tutorial](https://go.dev/doc/tutorial/generics)
- [slices Package Documentation](https://pkg.go.dev/slices)
- [maps Package Documentation](https://pkg.go.dev/maps)
- [Project Roadmap](./ROADMAP_2.4.9.md)
- [Progress Report](./PROGRESS_REPORT_2.4.9.md)

---

**Last Updated**: 2025-11-04  
**Version**: 2.4.9-alpha  
**Maintainer**: AI Gateway Team

