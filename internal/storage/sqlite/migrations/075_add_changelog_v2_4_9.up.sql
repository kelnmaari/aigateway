INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.4.9', '2025-11-04', '## [2.4.9] - 2025-11-04

### Added

- **Go 1.18-1.25 Modernization** 🚀:
  - **Generic Pointer Helpers** (`internal/utils/ptr.go`):
    - `Ptr[T](val T) *T` - create pointer to value
    - `Value[T](ptr *T) T` - safe dereference with zero fallback
    - `PtrOrNil[T](val T) *T` - return nil for zero values
    - `Deref[T](ptr *T, defaultVal T) T` - dereference with default
    - `Equal[T](a, b *T) bool` - nil-safe pointer comparison
    - `Clone[T](ptr *T) *T` - nil-safe pointer cloning
    - Refactored 7 production files, replaced 50+ lines of duplicate helpers
  - **Standard Library Integration**:
    - `slices` package (Go 1.21+): `Contains`, `ContainsFunc`, `DeleteFunc`, `Index`
    - `maps` package (Go 1.21+): `Clone`, `Copy`, `Equal` for safe map operations
    - Refactored 6 files: apikey_usage_threadsafe.go, json_storage.go, models.go, etc.
    - Code reduction: ~20 lines, improved safety and readability
  - **Generic API Response Wrapper** (`internal/api/response.go`):
    - `APIResponse[T any]` - type-safe success/error responses
    - `PaginatedResponse[T any]` - generic pagination with metadata
    - `ErrorResponse` - structured error format
    - Helper functions: `NewSuccess`, `NewPaginated`, `NewError`, `RespondSuccess`, etc.
  - **Custom Iterators** (Go 1.23):
    - `internal/storage/iterators.go`: 9 iterators (PaginatedIterator, ChunkedIterator, FilteredIterator, etc.)
    - `internal/rag/iterators.go`: 9 RAG-specific iterators (ChunksIterator, DataSourcesIterator, etc.)
    - Foundation for future database-level streaming
  - **Generic Type Aliases** (Go 1.24):
    - `ConfigMap[T comparable]` - typed configuration maps with merge support
    - `Option[T any]` - safe optional value wrapper (Some/None pattern)
    - `Set[T comparable]` - generic set with union/intersection operations
    - ~360 LOC with comprehensive tests
  - **Generic Vector Operations** (`internal/rag/vector/vector.go`):
    - `Vector[T ~float32 | ~float64]` - type-safe embedding operations
    - 20+ operations: DotProduct, CosineSimilarity, EuclideanDistance, Normalize, Add, etc.
    - Support for both float32 and float64 precision
    - Comprehensive benchmarks included

- **Go 1.25 New Features**:
  - **sync.WaitGroup.Go()** - Integrated in `internal/preloader/manager.go`
  - **runtime/trace.FlightRecorder** (`internal/debug/flight_recorder.go`):
    - Auto-save last 30 seconds on panic
    - Production debugging without overhead
    - Integrated in `cmd/server/main.go`
  - **net/http.CrossOriginProtection** (`internal/api/middleware/csrf.go`):
    - CSRF protection using Go 1.25 native API
    - Integrated in `internal/api/router/router.go`
    - Trusted origins from CORS config
  - **testing.T.Attr()** - Added to 15+ test files for better observability
  - **go vet analyzers**: Documented waitgroup and hostport checks

- **Production Code Improvements**:
  - Refactored 12 files with modern Go patterns
  - Code reduction: ~98 lines of boilerplate removed
  - Total additions: +6,495 LOC (code + documentation)

### Changed

- **Updated Production Code**:
  - `internal/rag/worker/db_sync.go` - uses `utils.Ptr`
  - `internal/api/handlers/model_warmup.go` - uses `utils.Ptr`
  - `internal/converter/simple_converter.go` - uses `utils.Ptr`
  - `internal/services/audit/logger.go` - uses `utils.Ptr`
  - `internal/models/mapping.go` - uses `utils.Ptr`
  - `internal/api/middleware/advanced_rate_limit.go` - uses `utils.PtrOrNil`
  - `internal/services/rag/datasource_service_test.go` - uses `utils.Ptr`

### Technical

- **Documentation Created**:
  - `docs/GENERICS_USAGE_GUIDE.md` - Complete guide with examples
  - `docs/PROGRESS_REPORT_2.4.9.md` - Detailed implementation report
  - `docs/WAITGROUP_GO_GUIDE.md` - sync.WaitGroup.Go() patterns
  - `docs/FLIGHT_RECORDER_GUIDE.md` - Production debugging guide
  - `docs/CSRF_PROTECTION_GUIDE.md` - CSRF middleware documentation
  - `docs/TEST_ATTR_GUIDE.md` - T.Attr() usage guide
  - `scripts/refactor_pointer_helpers.ps1` - Automated refactoring script
- **Test Coverage**: All new features have 100% test coverage
- **Benchmarks**: Performance benchmarks for Vector operations and Stats optimization
- **Foundation Ready**: Iterators, Type Aliases ready for future feature integration
- **Impact**: Significant code modernization, improved type safety, reduced boilerplate');

