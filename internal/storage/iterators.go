// Package storage provides custom iterators for database operations (Go 1.23+)
package storage

import (
	"context"
	"iter"
)

// PaginatedIterator creates an iterator for paginated database queries.
// This leverages Go 1.23's "range over func" feature for clean pagination.
//
// Example usage:
//
//	for item := range storage.PaginatedIterator(ctx, db.ListUsers, 20) {
//	    if item.Err != nil {
//	        log.Error(item.Err)
//	        break
//	    }
//	    process(item.Value)
//	}
func PaginatedIterator[T any](
	ctx context.Context,
	fetchFn func(ctx context.Context, limit, offset int) ([]T, error),
	pageSize int,
) iter.Seq[Result[T]] {
	return func(yield func(Result[T]) bool) {
		offset := 0
		for {
			items, err := fetchFn(ctx, pageSize, offset)
			if err != nil {
				yield(Result[T]{Err: err})
				return
			}

			if len(items) == 0 {
				return // No more items
			}

			for _, item := range items {
				if !yield(Result[T]{Value: item}) {
					return // Consumer stopped iteration
				}
			}

			if len(items) < pageSize {
				return // Last page (partial)
			}

			offset += pageSize
		}
	}
}

// Result represents a single item from an iterator with optional error.
type Result[T any] struct {
	Value T
	Err   error
}

// ChunkedIterator creates an iterator that processes items in chunks.
// Useful for batch operations like bulk inserts or parallel processing.
//
// Example usage:
//
//	for chunk := range storage.ChunkedIterator(allItems, 100) {
//	    db.BulkInsert(chunk)
//	}
func ChunkedIterator[T any](items []T, chunkSize int) iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		for i := 0; i < len(items); i += chunkSize {
			end := i + chunkSize
			if end > len(items) {
				end = len(items)
			}
			
			chunk := items[i:end]
			if !yield(chunk) {
				return // Consumer stopped iteration
			}
		}
	}
}

// FilteredIterator creates an iterator that filters items based on a predicate.
// This enables lazy filtering without creating intermediate slices.
//
// Example usage:
//
//	activeUsers := storage.FilteredIterator(allUsers, func(u User) bool {
//	    return u.IsActive
//	})
//	for user := range activeUsers {
//	    notify(user)
//	}
func FilteredIterator[T any](items []T, predicate func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range items {
			if predicate(item) {
				if !yield(item) {
					return
				}
			}
		}
	}
}

// MappedIterator creates an iterator that transforms items.
// This enables lazy mapping without allocating intermediate slices.
//
// Example usage:
//
//	userIDs := storage.MappedIterator(users, func(u User) string {
//	    return u.ID
//	})
//	for id := range userIDs {
//	    cache.Invalidate(id)
//	}
func MappedIterator[T, R any](items []T, mapper func(T) R) iter.Seq[R] {
	return func(yield func(R) bool) {
		for _, item := range items {
			if !yield(mapper(item)) {
				return
			}
		}
	}
}

// TakeIterator limits the number of items from an iterator.
//
// Example usage:
//
//	first10 := storage.TakeIterator(allItems, 10)
//	for item := range first10 {
//	    process(item)
//	}
func TakeIterator[T any](items []T, limit int) iter.Seq[T] {
	return func(yield func(T) bool) {
		count := 0
		for _, item := range items {
			if count >= limit {
				return
			}
			if !yield(item) {
				return
			}
			count++
		}
	}
}

// SkipIterator skips the first N items from an iterator.
//
// Example usage:
//
//	afterFirst20 := storage.SkipIterator(allItems, 20)
//	for item := range afterFirst20 {
//	    process(item)
//	}
func SkipIterator[T any](items []T, skip int) iter.Seq[T] {
	return func(yield func(T) bool) {
		count := 0
		for _, item := range items {
			if count < skip {
				count++
				continue
			}
			if !yield(item) {
				return
			}
		}
	}
}

// PairIterator creates an iterator over consecutive pairs.
// Useful for comparison operations or sliding window algorithms.
//
// Example usage:
//
//	for pair := range storage.PairIterator(timestamps) {
//	    delta := pair.Second - pair.First
//	    metrics.RecordInterval(delta)
//	}
func PairIterator[T any](items []T) iter.Seq[Pair[T]] {
	return func(yield func(Pair[T]) bool) {
		if len(items) < 2 {
			return
		}
		
		for i := 0; i < len(items)-1; i++ {
			pair := Pair[T]{
				First:  items[i],
				Second: items[i+1],
			}
			if !yield(pair) {
				return
			}
		}
	}
}

// Pair represents a consecutive pair of items.
type Pair[T any] struct {
	First  T
	Second T
}

// EnumerateIterator adds index to each item.
//
// Example usage:
//
//	for item := range storage.EnumerateIterator(users) {
//	    fmt.Printf("%d: %s\n", item.Index, item.Value.Name)
//	}
func EnumerateIterator[T any](items []T) iter.Seq[Indexed[T]] {
	return func(yield func(Indexed[T]) bool) {
		for i, item := range items {
			if !yield(Indexed[T]{Index: i, Value: item}) {
				return
			}
		}
	}
}

// Indexed represents an item with its index.
type Indexed[T any] struct {
	Index int
	Value T
}

