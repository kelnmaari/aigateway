package storage

import (
	"context"
	"errors"
	"testing"
)

type TestItem struct {
	ID   int
	Name string
}

func TestPaginatedIterator(t *testing.T) {
	t.Run("successful pagination", func(t *testing.T) {
		items := []TestItem{
			{1, "a"}, {2, "b"}, {3, "c"},
			{4, "d"}, {5, "e"},
		}

		fetchFn := func(ctx context.Context, limit, offset int) ([]TestItem, error) {
			if offset >= len(items) {
				return []TestItem{}, nil
			}
			end := offset + limit
			if end > len(items) {
				end = len(items)
			}
			return items[offset:end], nil
		}

		collected := []TestItem{}
		for result := range PaginatedIterator(context.Background(), fetchFn, 2) {
			if result.Err != nil {
				t.Fatalf("unexpected error: %v", result.Err)
			}
			collected = append(collected, result.Value)
		}

		if len(collected) != 5 {
			t.Errorf("expected 5 items, got %d", len(collected))
		}
	})

	t.Run("error handling", func(t *testing.T) {
		fetchFn := func(ctx context.Context, limit, offset int) ([]TestItem, error) {
			return nil, errors.New("database error")
		}

		for result := range PaginatedIterator(context.Background(), fetchFn, 10) {
			if result.Err == nil {
				t.Error("expected error, got nil")
			}
			break
		}
	})

	t.Run("early termination", func(t *testing.T) {
		items := []TestItem{{1, "a"}, {2, "b"}, {3, "c"}}
		fetchFn := func(ctx context.Context, limit, offset int) ([]TestItem, error) {
			if offset >= len(items) {
				return []TestItem{}, nil
			}
			end := offset + limit
			if end > len(items) {
				end = len(items)
			}
			return items[offset:end], nil
		}

		count := 0
		for result := range PaginatedIterator(context.Background(), fetchFn, 1) {
			if result.Err != nil {
				t.Fatal(result.Err)
			}
			count++
			if count >= 2 {
				break // Stop early
			}
		}

		if count != 2 {
			t.Errorf("expected to collect 2 items, got %d", count)
		}
	})
}

func TestChunkedIterator(t *testing.T) {
	t.Run("full chunks", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5, 6}
		chunks := [][]int{}
		
		for chunk := range ChunkedIterator(items, 2) {
			chunks = append(chunks, chunk)
		}

		if len(chunks) != 3 {
			t.Errorf("expected 3 chunks, got %d", len(chunks))
		}
		if len(chunks[0]) != 2 || len(chunks[1]) != 2 || len(chunks[2]) != 2 {
			t.Error("chunk sizes incorrect")
		}
	})

	t.Run("partial last chunk", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		chunks := [][]int{}
		
		for chunk := range ChunkedIterator(items, 2) {
			chunks = append(chunks, chunk)
		}

		if len(chunks) != 3 {
			t.Errorf("expected 3 chunks, got %d", len(chunks))
		}
		if len(chunks[2]) != 1 {
			t.Errorf("last chunk should have 1 item, got %d", len(chunks[2]))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		items := []int{}
		count := 0
		
		for range ChunkedIterator(items, 2) {
			count++
		}

		if count != 0 {
			t.Errorf("expected 0 chunks, got %d", count)
		}
	})
}

func TestFilteredIterator(t *testing.T) {
	items := []TestItem{
		{1, "active"}, {2, "inactive"}, {3, "active"},
		{4, "inactive"}, {5, "active"},
	}

	predicate := func(item TestItem) bool {
		return item.Name == "active"
	}

	collected := []TestItem{}
	for item := range FilteredIterator(items, predicate) {
		collected = append(collected, item)
	}

	if len(collected) != 3 {
		t.Errorf("expected 3 active items, got %d", len(collected))
	}

	for _, item := range collected {
		if item.Name != "active" {
			t.Errorf("unexpected filtered item: %v", item)
		}
	}
}

func TestMappedIterator(t *testing.T) {
	items := []TestItem{{1, "a"}, {2, "b"}, {3, "c"}}
	
	mapper := func(item TestItem) int {
		return item.ID * 10
	}

	collected := []int{}
	for value := range MappedIterator(items, mapper) {
		collected = append(collected, value)
	}

	expected := []int{10, 20, 30}
	if len(collected) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(collected))
	}

	for i, val := range collected {
		if val != expected[i] {
			t.Errorf("at index %d: expected %d, got %d", i, expected[i], val)
		}
	}
}

func TestTakeIterator(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	t.Run("take less than length", func(t *testing.T) {
		collected := []int{}
		for item := range TakeIterator(items, 5) {
			collected = append(collected, item)
		}

		if len(collected) != 5 {
			t.Errorf("expected 5 items, got %d", len(collected))
		}
	})

	t.Run("take more than length", func(t *testing.T) {
		collected := []int{}
		for item := range TakeIterator(items, 20) {
			collected = append(collected, item)
		}

		if len(collected) != 10 {
			t.Errorf("expected 10 items, got %d", len(collected))
		}
	})

	t.Run("take zero", func(t *testing.T) {
		count := 0
		for range TakeIterator(items, 0) {
			count++
		}

		if count != 0 {
			t.Errorf("expected 0 items, got %d", count)
		}
	})
}

func TestSkipIterator(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	t.Run("skip first elements", func(t *testing.T) {
		collected := []int{}
		for item := range SkipIterator(items, 2) {
			collected = append(collected, item)
		}

		if len(collected) != 3 {
			t.Errorf("expected 3 items, got %d", len(collected))
		}
		if collected[0] != 3 {
			t.Errorf("expected first item to be 3, got %d", collected[0])
		}
	})

	t.Run("skip more than length", func(t *testing.T) {
		count := 0
		for range SkipIterator(items, 10) {
			count++
		}

		if count != 0 {
			t.Errorf("expected 0 items, got %d", count)
		}
	})
}

func TestPairIterator(t *testing.T) {
	t.Run("normal case", func(t *testing.T) {
		items := []int{1, 2, 3, 4}
		pairs := []Pair[int]{}
		
		for pair := range PairIterator(items) {
			pairs = append(pairs, pair)
		}

		if len(pairs) != 3 {
			t.Fatalf("expected 3 pairs, got %d", len(pairs))
		}

		expected := []Pair[int]{
			{1, 2}, {2, 3}, {3, 4},
		}

		for i, pair := range pairs {
			if pair != expected[i] {
				t.Errorf("pair %d: expected %v, got %v", i, expected[i], pair)
			}
		}
	})

	t.Run("insufficient items", func(t *testing.T) {
		items := []int{1}
		count := 0
		
		for range PairIterator(items) {
			count++
		}

		if count != 0 {
			t.Errorf("expected 0 pairs, got %d", count)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		items := []int{}
		count := 0
		
		for range PairIterator(items) {
			count++
		}

		if count != 0 {
			t.Errorf("expected 0 pairs, got %d", count)
		}
	})
}

func TestEnumerateIterator(t *testing.T) {
	items := []string{"a", "b", "c"}
	collected := []Indexed[string]{}
	
	for indexed := range EnumerateIterator(items) {
		collected = append(collected, indexed)
	}

	if len(collected) != 3 {
		t.Fatalf("expected 3 items, got %d", len(collected))
	}

	for i, item := range collected {
		if item.Index != i {
			t.Errorf("expected index %d, got %d", i, item.Index)
		}
		if item.Value != items[i] {
			t.Errorf("expected value %s, got %s", items[i], item.Value)
		}
	}
}

// Benchmarks
func BenchmarkPaginatedIterator(b *testing.B) {
	items := make([]TestItem, 1000)
	for i := range items {
		items[i] = TestItem{ID: i, Name: "test"}
	}

	fetchFn := func(ctx context.Context, limit, offset int) ([]TestItem, error) {
		if offset >= len(items) {
			return []TestItem{}, nil
		}
		end := offset + limit
		if end > len(items) {
			end = len(items)
		}
		return items[offset:end], nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range PaginatedIterator(context.Background(), fetchFn, 100) {
			// Consume iterator
		}
	}
}

func BenchmarkFilteredIterator(b *testing.B) {
	items := make([]TestItem, 1000)
	for i := range items {
		items[i] = TestItem{ID: i, Name: "test"}
	}

	predicate := func(item TestItem) bool {
		return item.ID%2 == 0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range FilteredIterator(items, predicate) {
			// Consume iterator
		}
	}
}

