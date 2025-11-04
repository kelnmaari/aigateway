// Package benchmarks provides GC benchmarking for Go 1.25 GreenTea GC
//
// Run with standard GC:
//
//	go test -bench=. -benchmem -benchtime=10s ./benchmarks
//
// Run with GreenTea GC (experimental):
//
//	GOEXPERIMENT=greenteagc go test -bench=. -benchmem -benchtime=10s ./benchmarks
//
// Compare results:
//
//	go test -bench=. -benchmem -benchtime=10s ./benchmarks > standard.txt
//	GOEXPERIMENT=greenteagc go test -bench=. -benchmem -benchtime=10s ./benchmarks > greentea.txt
//	benchstat standard.txt greentea.txt
package benchmarks

import (
	"runtime"
	"runtime/debug"
	"testing"
)

// BenchmarkGC_AllocationsOnly measures pure allocation performance
func BenchmarkGC_AllocationsOnly(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "allocations")

	// Allocate 1KB per iteration
	size := 1024

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = make([]byte, size)
	}
}

// BenchmarkGC_AllocDealloc measures allocation + deallocation cycles
func BenchmarkGC_AllocDealloc(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "alloc-dealloc")

	size := 1024
	slices := make([][]byte, 1000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Allocate
		for j := range slices {
			slices[j] = make([]byte, size)
		}

		// Deallocate by clearing references
		for j := range slices {
			slices[j] = nil
		}
	}
}

// BenchmarkGC_LongLivedObjects measures GC with persistent objects
func BenchmarkGC_LongLivedObjects(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "long-lived")

	// Create long-lived objects (survive multiple GC cycles)
	longLived := make([][]byte, 10000)
	for i := range longLived {
		longLived[i] = make([]byte, 1024)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Allocate short-lived objects
		_ = make([]byte, 1024)

		// Keep long-lived objects alive
		runtime.KeepAlive(longLived)
	}
}

// BenchmarkGC_HighChurnRate measures GC under high allocation pressure
func BenchmarkGC_HighChurnRate(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "high-churn")

	// Simulate high churn workload (API gateway scenario)
	type Request struct {
		ID      string
		Payload []byte
		Headers map[string]string
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &Request{
			ID:      "req-12345",
			Payload: make([]byte, 4096),
			Headers: make(map[string]string, 10),
		}

		// Simulate processing
		req.Headers["Content-Type"] = "application/json"
		req.Headers["Authorization"] = "Bearer token"

		// Request goes out of scope (garbage)
		_ = req
	}
}

// BenchmarkGC_MixedObjectSizes measures GC with varied allocation sizes
func BenchmarkGC_MixedObjectSizes(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "mixed-sizes")

	sizes := []int{16, 128, 1024, 8192, 65536} // 16B to 64KB

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, size := range sizes {
			_ = make([]byte, size)
		}
	}
}

// BenchmarkGC_ConcurrentAllocations measures GC with concurrent goroutines
func BenchmarkGC_ConcurrentAllocations(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "concurrent")

	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Each goroutine allocates independently
			_ = make([]byte, 1024)
		}
	})
}

// BenchmarkGC_PointerHeavy measures GC with pointer-rich structures
func BenchmarkGC_PointerHeavy(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "pointer-heavy")

	type Node struct {
		Value int
		Next  *Node
		Prev  *Node
		Data  []byte
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create linked list (many pointers)
		head := &Node{Value: 1, Data: make([]byte, 64)}
		current := head

		for j := 0; j < 100; j++ {
			node := &Node{Value: j, Data: make([]byte, 64)}
			current.Next = node
			node.Prev = current
			current = node
		}

		// Let GC collect the list
		_ = head
	}
}

// BenchmarkGC_MapOperations measures GC impact on map-heavy workloads
func BenchmarkGC_MapOperations(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "maps")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m := make(map[string]interface{}, 100)

		for j := 0; j < 100; j++ {
			m[string(rune('a'+j))] = make([]byte, 128)
		}

		// Clear map
		for k := range m {
			delete(m, k)
		}
	}
}

// BenchmarkGC_StringConcatenation measures GC with string operations
func BenchmarkGC_StringConcatenation(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "strings")

	base := "Hello, World! "

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := base
		for j := 0; j < 10; j++ {
			s += base // Creates new string each time
		}
		_ = s
	}
}

// BenchmarkGC_ChannelCommunication measures GC with channel operations
func BenchmarkGC_ChannelCommunication(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "channels")

	ch := make(chan []byte, 100)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			select {
			case ch <- make([]byte, 1024):
			case <-ch:
			default:
			}
		}
	})
}

// BenchmarkGC_RealisticAPIGateway simulates aigateway workload
func BenchmarkGC_RealisticAPIGateway(b *testing.B) {
	b.Attr("category", "gc")
	b.Attr("type", "performance")
	b.Attr("gc_test", "realistic-workload")

	type Message struct {
		Role    string
		Content string
	}

	type ChatRequest struct {
		Model       string
		Messages    []Message
		Temperature float64
		MaxTokens   int
		Metadata    map[string]string
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate incoming request
		req := &ChatRequest{
			Model:       "deepseek-r1",
			Temperature: 0.7,
			MaxTokens:   2048,
			Messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "What is Go GreenTea GC?"},
			},
			Metadata: map[string]string{
				"user_id": "user-123",
				"api_key": "sk-xxx",
			},
		}

		// Simulate response
		response := make([]byte, 4096) // ~4KB response
		copy(response, "Response data...")

		// Request and response go out of scope
		runtime.KeepAlive(req)
		runtime.KeepAlive(response)
	}
}

// GCStats captures GC statistics before and after benchmark
type GCStats struct {
	NumGC        uint32
	PauseTotal   uint64 // nanoseconds
	PauseAvg     uint64 // nanoseconds
	HeapAlloc    uint64
	HeapSys      uint64
	HeapObjects  uint64
}

// GetGCStats returns current GC statistics
func GetGCStats() GCStats {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var pauseAvg uint64
	if mem.NumGC > 0 {
		pauseAvg = mem.PauseTotalNs / uint64(mem.NumGC)
	}

	return GCStats{
		NumGC:       mem.NumGC,
		PauseTotal:  mem.PauseTotalNs,
		PauseAvg:    pauseAvg,
		HeapAlloc:   mem.HeapAlloc,
		HeapSys:     mem.HeapSys,
		HeapObjects: mem.HeapObjects,
	}
}

// TestGCStats_Baseline reports baseline GC stats (not a benchmark)
func TestGCStats_Baseline(t *testing.T) {
	t.Attr("category", "gc")
	t.Attr("type", "stats")

	// Force GC
	runtime.GC()

	stats := GetGCStats()

	t.Logf("=== GC Statistics ===")
	t.Logf("GC Implementation: %s", func() string {
		if debug.SetGCPercent(-1); debug.SetGCPercent(100) == -1 {
			return "Standard"
		}
		// GreenTea GC detection would need GOEXPERIMENT check
		return "Standard (or GreenTea if GOEXPERIMENT=greenteagc)"
	}())
	t.Logf("Number of GCs: %d", stats.NumGC)
	t.Logf("Total GC Pause: %.2f ms", float64(stats.PauseTotal)/1e6)
	t.Logf("Average GC Pause: %.2f µs", float64(stats.PauseAvg)/1e3)
	t.Logf("Heap Allocated: %.2f MB", float64(stats.HeapAlloc)/(1024*1024))
	t.Logf("Heap System: %.2f MB", float64(stats.HeapSys)/(1024*1024))
	t.Logf("Heap Objects: %d", stats.HeapObjects)
}

