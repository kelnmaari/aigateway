// Package types provides generic type aliases for common patterns (Go 1.24+)
package types

// ConfigMap is a generic type alias for configuration maps with typed values.
// This provides type safety for configuration storage and retrieval.
//
// Example usage:
//
//	type ServerConfig = types.ConfigMap[string]
//	config := ServerConfig{
//	    "host": "localhost",
//	    "port": "8080",
//	}
//
//	type NumericConfig = types.ConfigMap[int]
//	limits := NumericConfig{
//	    "max_connections": 100,
//	    "timeout_seconds": 30,
//	}
type ConfigMap[T any] = map[string]T

// Metadata is a generic type alias for metadata storage with typed values.
// Commonly used for request metadata, user attributes, etc.
//
// Example usage:
//
//	type UserMetadata = types.Metadata[string]
//	meta := UserMetadata{
//	    "region": "us-east-1",
//	    "tier": "premium",
//	}
type Metadata[T any] = map[string]T

// IDMap is a generic type alias for maps keyed by string IDs.
// Useful for caches, lookups, and collections indexed by ID.
//
// Example usage:
//
//	type UserCache = types.IDMap[*User]
//	cache := UserCache{
//	    "user-123": &User{ID: "123", Name: "John"},
//	    "user-456": &User{ID: "456", Name: "Jane"},
//	}
type IDMap[T any] = map[string]T

// KeyValue represents a generic key-value pair.
// Useful for converting maps to slices or iterating with order.
//
// Example usage:
//
//	pairs := []types.KeyValue[string, int]{
//	    {Key: "apples", Value: 5},
//	    {Key: "oranges", Value: 3},
//	}
type KeyValue[K comparable, V any] struct {
	Key   K
	Value V
}

// Result represents an operation result with either value or error.
// This is similar to Rust's Result<T, E> pattern.
//
// Example usage:
//
//	func fetchUser(id string) types.Result[*User] {
//	    user, err := db.GetUser(id)
//	    if err != nil {
//	        return types.Result[*User]{Err: err}
//	    }
//	    return types.Result[*User]{Value: user, Ok: true}
//	}
//
//	result := fetchUser("123")
//	if result.Ok {
//	    process(result.Value)
//	} else {
//	    log.Error(result.Err)
//	}
type Result[T any] struct {
	Value T
	Err   error
	Ok    bool
}

// Option represents an optional value (similar to Rust's Option<T>).
// Provides a null-safe alternative to pointers.
//
// Example usage:
//
//	func findUser(id string) types.Option[User] {
//	    user, found := cache.Get(id)
//	    if !found {
//	        return types.None[User]()
//	    }
//	    return types.Some(user)
//	}
type Option[T any] struct {
	value   T
	present bool
}

// Some creates an Option with a value present.
func Some[T any](value T) Option[T] {
	return Option[T]{value: value, present: true}
}

// None creates an empty Option.
func None[T any]() Option[T] {
	return Option[T]{present: false}
}

// IsSome returns true if the Option contains a value.
func (o Option[T]) IsSome() bool {
	return o.present
}

// IsNone returns true if the Option is empty.
func (o Option[T]) IsNone() bool {
	return !o.present
}

// Unwrap returns the value, panicking if none present.
// Use only when you're certain the value exists.
func (o Option[T]) Unwrap() T {
	if !o.present {
		panic("called Unwrap on None")
	}
	return o.value
}

// UnwrapOr returns the value or a default if none present.
func (o Option[T]) UnwrapOr(defaultValue T) T {
	if o.present {
		return o.value
	}
	return defaultValue
}

// UnwrapOrElse returns the value or computes a default.
func (o Option[T]) UnwrapOrElse(fn func() T) T {
	if o.present {
		return o.value
	}
	return fn()
}

// Map transforms the Option's value if present.
func (o Option[T]) Map(fn func(T) T) Option[T] {
	if !o.present {
		return o
	}
	return Some(fn(o.value))
}

// StringSet is a type-safe set implementation using map[string]struct{}.
//
// Example usage:
//
//	visited := types.NewStringSet()
//	visited.Add("page1")
//	visited.Add("page2")
//	
//	if visited.Contains("page1") {
//	    // ...
//	}
type StringSet map[string]struct{}

// NewStringSet creates a new empty StringSet.
func NewStringSet(items ...string) StringSet {
	set := make(StringSet, len(items))
	for _, item := range items {
		set.Add(item)
	}
	return set
}

// Add adds an item to the set.
func (s StringSet) Add(item string) {
	s[item] = struct{}{}
}

// Contains checks if an item is in the set.
func (s StringSet) Contains(item string) bool {
	_, exists := s[item]
	return exists
}

// Remove removes an item from the set.
func (s StringSet) Remove(item string) {
	delete(s, item)
}

// Size returns the number of items in the set.
func (s StringSet) Size() int {
	return len(s)
}

// Items returns all items as a slice.
func (s StringSet) Items() []string {
	items := make([]string, 0, len(s))
	for item := range s {
		items = append(items, item)
	}
	return items
}

// Clear removes all items from the set.
func (s StringSet) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// Set is a generic set implementation.
//
// Example usage:
//
//	userIDs := types.NewSet[string]()
//	userIDs.Add("user-123")
//	userIDs.Add("user-456")
//	
//	if userIDs.Contains("user-123") {
//	    // ...
//	}
type Set[T comparable] map[T]struct{}

// NewSet creates a new empty Set.
func NewSet[T comparable](items ...T) Set[T] {
	set := make(Set[T], len(items))
	for _, item := range items {
		set.Add(item)
	}
	return set
}

// Add adds an item to the set.
func (s Set[T]) Add(item T) {
	s[item] = struct{}{}
}

// Contains checks if an item is in the set.
func (s Set[T]) Contains(item T) bool {
	_, exists := s[item]
	return exists
}

// Remove removes an item from the set.
func (s Set[T]) Remove(item T) {
	delete(s, item)
}

// Size returns the number of items in the set.
func (s Set[T]) Size() int {
	return len(s)
}

// Items returns all items as a slice.
func (s Set[T]) Items() []T {
	items := make([]T, 0, len(s))
	for item := range s {
		items = append(items, item)
	}
	return items
}

// Clear removes all items from the set.
func (s Set[T]) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// Union returns a new set with items from both sets.
func (s Set[T]) Union(other Set[T]) Set[T] {
	result := make(Set[T], len(s)+len(other))
	for item := range s {
		result.Add(item)
	}
	for item := range other {
		result.Add(item)
	}
	return result
}

// Intersection returns a new set with items in both sets.
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	result := NewSet[T]()
	for item := range s {
		if other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

// Difference returns a new set with items in s but not in other.
func (s Set[T]) Difference(other Set[T]) Set[T] {
	result := NewSet[T]()
	for item := range s {
		if !other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

// Cache is a generic type alias for simple in-memory caches.
//
// Example usage:
//
//	type UserCache = types.Cache[string, *User]
//	cache := make(UserCache)
//	cache["user-123"] = &User{ID: "123"}
type Cache[K comparable, V any] = map[K]V

// Counter is a generic type alias for counting occurrences.
//
// Example usage:
//
//	type WordCounter = types.Counter[string]
//	counter := make(WordCounter)
//	counter["hello"]++
//	counter["world"]++
type Counter[K comparable] = map[K]int

// Index is a generic type alias for inverted indexes.
//
// Example usage:
//
//	type TagIndex = types.Index[string, string]  // tag -> document IDs
//	index := make(TagIndex)
//	index["golang"] = []string{"doc1", "doc2"}
type Index[K comparable, V any] = map[K][]V

