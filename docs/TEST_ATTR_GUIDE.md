# testing.T.Attr() Guide - Go 1.25

**Feature**: Test attributes via `T.Attr()` method  
**Since**: Go 1.25.0  
**Status**: ✅ Stable (not experimental)

## Overview

Go 1.25 introduces `testing.T.Attr()` and `testing.B.Attr()` - methods to attach arbitrary metadata to tests and benchmarks. This enables better test organization, filtering, and reporting.

## Quick Start

### Basic Usage

```go
func TestMyFunction(t *testing.T) {
    t.Attr("category", "unit")
    t.Attr("type", "fast")
    t.Attr("owner", "backend-team")
    
    // Test code...
}

func BenchmarkPerformance(b *testing.B) {
    b.Attr("category", "performance")
    b.Attr("critical", "true")
    
    // Benchmark code...
}
```

### Test Output

When running tests with `-v`, attributes are displayed:

```bash
=== RUN   TestMyFunction
=== ATTR  TestMyFunction category unit
=== ATTR  TestMyFunction type fast
=== ATTR  TestMyFunction owner backend-team
--- PASS: TestMyFunction (0.00s)
```

---

## Why Use Test Attributes?

### 1. **Test Organization**
Group tests by category, feature, or team ownership:

```go
func TestAPIHandlers(t *testing.T) {
    t.Attr("category", "api")
    t.Attr("feature", "chat-completion")
    t.Attr("team", "backend")
}
```

### 2. **Test Filtering** (Future)
Future `go test` versions may support filtering by attributes:

```bash
# Hypothetical future syntax
go test -attr="category=unit"
go test -attr="category!=integration"
go test -attr="critical=true"
```

### 3. **CI/CD Integration**
Parse test output to:
- Generate test reports grouped by category
- Track ownership and coverage per team
- Identify critical vs non-critical test failures

### 4. **Documentation**
Self-documenting tests with metadata:

```go
func TestUserAuthentication(t *testing.T) {
    t.Attr("category", "security")
    t.Attr("compliance", "GDPR")
    t.Attr("go_version", "1.25")
    t.Attr("reviewed_by", "security-team")
}
```

---

## Common Attribute Patterns

### 1. Category Classification

```go
// Unit tests
func TestPureFunction(t *testing.T) {
    t.Attr("category", "unit")
    t.Attr("type", "fast")
}

// Integration tests
func TestDatabaseIntegration(t *testing.T) {
    t.Attr("category", "integration")
    t.Attr("type", "slow")
    t.Attr("requires", "database")
}

// End-to-end tests
func TestFullWorkflow(t *testing.T) {
    t.Attr("category", "e2e")
    t.Attr("type", "slow")
    t.Attr("requires", "network,database")
}
```

### 2. Go Version Tracking

```go
func TestGenerics(t *testing.T) {
    t.Attr("go_version", "1.25")
    t.Attr("feature", "generics")
}

func TestCustomIterators(t *testing.T) {
    t.Attr("go_version", "1.23")
    t.Attr("feature", "range-over-func")
}
```

### 3. Performance Tracking

```go
func BenchmarkCriticalPath(b *testing.B) {
    b.Attr("category", "benchmarks")
    b.Attr("type", "performance")
    b.Attr("critical", "true")
    b.Attr("slo", "<100ms") // Service Level Objective
}
```

### 4. Team Ownership

```go
func TestAuthenticationLogic(t *testing.T) {
    t.Attr("category", "auth")
    t.Attr("team", "security")
    t.Attr("on_call", "security-team")
}
```

### 5. Compliance & Security

```go
func TestDataEncryption(t *testing.T) {
    t.Attr("category", "security")
    t.Attr("compliance", "SOC2,GDPR")
    t.Attr("severity", "critical")
}
```

---

## Real-World Examples

### Example 1: API Handler Tests

```go
func TestChatCompletionHandler(t *testing.T) {
    t.Attr("category", "api")
    t.Attr("type", "integration")
    t.Attr("feature", "chat-completion")
    t.Attr("go_version", "1.25")
    
    t.Run("successful request", func(t *testing.T) {
        // Test code...
    })
    
    t.Run("invalid model", func(t *testing.T) {
        // Test code...
    })
}
```

**Output:**
```
=== RUN   TestChatCompletionHandler
=== ATTR  TestChatCompletionHandler category api
=== ATTR  TestChatCompletionHandler type integration
=== ATTR  TestChatCompletionHandler feature chat-completion
=== ATTR  TestChatCompletionHandler go_version 1.25
```

---

### Example 2: Benchmark with Metadata

```go
func BenchmarkVectorSearch(b *testing.B) {
    b.Attr("category", "benchmarks")
    b.Attr("type", "performance")
    b.Attr("operation", "vector-similarity")
    b.Attr("dataset_size", "1M")
    b.Attr("critical", "true")
    
    // Setup
    vectors := generateTestVectors(1000000)
    query := generateQueryVector()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = findSimilarVectors(vectors, query, 10)
    }
}
```

---

### Example 3: Pointer Helper Tests (From Project)

```go
func TestPtr(t *testing.T) {
    t.Attr("category", "utils")
    t.Attr("type", "unit")
    t.Attr("go_version", "1.25")
    
    t.Run("string", func(t *testing.T) {
        s := "hello"
        ptr := Ptr(s)
        if ptr == nil {
            t.Fatal("expected non-nil pointer")
        }
        if *ptr != s {
            t.Errorf("expected %q, got %q", s, *ptr)
        }
    })
    
    // More subtests...
}
```

---

## Best Practices

### 1. **Consistent Naming**
Use standardized attribute names across your project:

```go
// ✅ Good - consistent naming
t.Attr("category", "unit")
t.Attr("category", "integration")

// ❌ Bad - inconsistent
t.Attr("category", "unit")
t.Attr("test_type", "integration") // Mixed naming
```

### 2. **Common Categories**
Establish a project-wide vocabulary:

```yaml
categories:
  - unit          # Isolated, fast tests
  - integration   # Database, external services
  - e2e           # Full workflows
  - regression    # Bug reproduction tests
  - security      # Security-focused tests
  - performance   # Performance/load tests
```

### 3. **Multiple Attributes**
Use multiple attributes for rich metadata:

```go
func TestRateLimiting(t *testing.T) {
    t.Attr("category", "security")
    t.Attr("subcategory", "rate-limiting")
    t.Attr("priority", "high")
    t.Attr("team", "backend")
    t.Attr("requires", "redis")
}
```

### 4. **Avoid Over-Attribution**
Don't add attributes that provide no value:

```go
// ❌ Too many attributes
t.Attr("written_on", "2025-11-04")
t.Attr("author_email", "user@example.com")
t.Attr("file_location", "internal/utils/ptr_test.go")

// ✅ Relevant attributes only
t.Attr("category", "utils")
t.Attr("type", "unit")
```

---

## CI/CD Integration

### Parsing Test Output

Extract attributes from `go test -json` output:

```go
// Example: Parse JSON test output
type TestEvent struct {
    Time    time.Time
    Action  string
    Package string
    Test    string
    Output  string
}

func parseTestAttributes(jsonOutput []byte) map[string]map[string]string {
    attributes := make(map[string]map[string]string)
    
    scanner := bufio.NewScanner(bytes.NewReader(jsonOutput))
    for scanner.Scan() {
        var event TestEvent
        if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
            continue
        }
        
        // Parse "=== ATTR  TestName key value" lines
        if strings.HasPrefix(event.Output, "=== ATTR") {
            // Extract test name, key, value
            // Store in attributes map
        }
    }
    
    return attributes
}
```

### GitHub Actions Example

```yaml
- name: Run tests with attributes
  run: |
    go test -v -json ./... > test-output.json
    
- name: Generate test report
  run: |
    python scripts/parse_test_attrs.py test-output.json > test-report.html
    
- name: Upload test report
  uses: actions/upload-artifact@v3
  with:
    name: test-report
    path: test-report.html
```

---

## Migration Strategy

### Step 1: Identify Test Categories

Audit existing tests and categorize:

```bash
# Find all test functions
grep -r "func Test" . | wc -l

# Categorize manually or with script
# - Unit tests (no external dependencies)
# - Integration tests (database, API calls)
# - End-to-end tests
```

### Step 2: Add Attributes Incrementally

Start with high-value tests:

1. **Critical tests first** (auth, security, payment)
2. **Performance benchmarks** (SLA monitoring)
3. **Integration tests** (for filtering in CI/CD)

### Step 3: Create Standard Attributes

Document project-wide attribute schema:

```markdown
# Test Attributes Standard

## Required Attributes
- `category`: unit|integration|e2e|regression

## Optional Attributes
- `type`: fast|slow
- `team`: team-name
- `feature`: feature-name
- `requires`: comma-separated dependencies
- `go_version`: minimum Go version
- `priority`: low|medium|high|critical
```

### Step 4: Automate Validation

Add linter to enforce attribute presence:

```go
// scripts/test_lint.go
func validateTestAttributes(file string) error {
    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, file, nil, 0)
    if err != nil {
        return err
    }
    
    for _, decl := range node.Decls {
        if fn, ok := decl.(*ast.FuncDecl); ok {
            if strings.HasPrefix(fn.Name.Name, "Test") {
                // Check if function calls t.Attr()
                hasAttr := inspectForAttrCall(fn)
                if !hasAttr {
                    return fmt.Errorf("%s missing t.Attr() calls", fn.Name.Name)
                }
            }
        }
    }
    
    return nil
}
```

---

## Limitations & Future

### Current Limitations (Go 1.25)

1. **No Built-in Filtering**: Can't filter tests by attribute yet
   ```bash
   # This doesn't work in Go 1.25
   go test -attr="category=unit"
   ```

2. **No Query API**: Can't programmatically query test attributes

3. **Text Output Only**: Attributes appear in `-v` output, but not in JSON

### Expected in Future Go Versions

- **Test Filtering**: `go test -attr="key=value"`
- **JSON Output**: Attributes in `go test -json` output
- **Tooling**: IDE support for attribute-based test navigation

---

## Project Status: aigateway

### Files Updated with T.Attr()

**internal/utils/ptr_test.go**
- ✅ All test functions
- ✅ All benchmark functions
- Attributes: `category=utils`, `type=unit`, `go_version=1.25`

**internal/api/response_test.go**
- ✅ TestNewSuccess, TestNewSuccessWithMessage
- ✅ BenchmarkNewSuccess
- Attributes: `category=api`, `type=unit`, `go_version=1.25`

### Attribute Schema Used

```yaml
category:
  - utils
  - api
  - benchmarks

type:
  - unit
  - integration
  - performance

go_version:
  - "1.25"  # Indicates Go 1.25+ specific features
```

### Next Steps

1. Add attributes to remaining test files (100+ files)
2. Create automation script to add attributes to new tests
3. Build CI/CD parser for attribute-based reporting
4. Update contributing guide with attribute requirements

---

## Summary

**What is T.Attr()?**
- Attach metadata to tests and benchmarks
- Self-documenting test characteristics
- Future-proof for test filtering

**Why Use It?**
- Better test organization
- CI/CD integration opportunities
- Team ownership tracking
- Compliance documentation

**How to Adopt?**
1. Define attribute schema
2. Add to critical tests first
3. Automate validation
4. Build CI/CD reporting

**Impact:**
- ✅ Minimal code change (`t.Attr("key", "value")`)
- ✅ No performance overhead
- ✅ Better test visibility
- ✅ Enables future tooling

---

**Last Updated**: 2025-11-04  
**Go Version**: 1.25.3  
**Project**: aigateway v2.4.9

