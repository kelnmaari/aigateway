// Package utils provides utility functions for common Go patterns.
package utils

// Ptr creates a pointer to the given value.
// This is useful for inline pointer creation in struct literals or function calls.
//
// Example:
//
//	config := &Config{
//	    Temperature: utils.Ptr(0.7),
//	    MaxTokens:   utils.Ptr(2048),
//	}
//
//go:fix inline
func Ptr[T any](v T) *T {
	return new(v)
}

// Value extracts the value from a pointer, returning the default value if the pointer is nil.
//
// Example:
//
//	temp := utils.Value(config.Temperature, 0.5)  // Returns 0.5 if Temperature is nil
func Value[T any](ptr *T, defaultVal T) T {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

// PtrOrNil returns a pointer to the value if it's not the zero value, otherwise returns nil.
// This is useful for optional fields where zero values should be omitted.
//
// Example:
//
//	name := utils.PtrOrNil("")       // Returns nil
//	count := utils.PtrOrNil(0)       // Returns nil
//	active := utils.PtrOrNil(true)   // Returns &true
func PtrOrNil[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// Deref dereferences a pointer, returning the zero value if the pointer is nil.
// This is a convenience wrapper around Value with zero value as default.
//
// Example:
//
//	val := utils.Deref(ptr)  // Returns *ptr or zero value of T
func Deref[T any](ptr *T) T {
	var zero T
	return Value(ptr, zero)
}

// Equal compares two pointers for equality, handling nil cases.
// Returns true if both are nil or both point to equal values.
//
// Example:
//
//	utils.Equal(utils.Ptr(5), utils.Ptr(5))   // true
//	utils.Equal(utils.Ptr(5), nil)            // false
//	utils.Equal[int](nil, nil)                // true
func Equal[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// Clone creates a new pointer with a copy of the value.
// Returns nil if the input pointer is nil.
//
// Example:
//
//	original := utils.Ptr(42)
//	clone := utils.Clone(original)  // Different pointer, same value
func Clone[T any](ptr *T) *T {
	if ptr == nil {
		return nil
	}
	v := *ptr
	return &v
}
