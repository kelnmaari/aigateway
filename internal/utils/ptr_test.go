package utils

import (
	"testing"
)

func TestPtr(t *testing.T) {
	t.Attr("category", "utils")
	t.Attr("type", "unit")
	t.Attr("go_version", "1.25")

	t.Run("string", func(t *testing.T) {
		s := "hello"
		ptr := new(s)
		if ptr == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *ptr != s {
			t.Errorf("expected %q, got %q", s, *ptr)
		}
	})

	t.Run("int", func(t *testing.T) {
		i := 42
		ptr := new(i)
		if ptr == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *ptr != i {
			t.Errorf("expected %d, got %d", i, *ptr)
		}
	})

	t.Run("float64", func(t *testing.T) {
		f := 3.14
		ptr := new(f)
		if ptr == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *ptr != f {
			t.Errorf("expected %f, got %f", f, *ptr)
		}
	})

	t.Run("bool", func(t *testing.T) {
		b := true
		ptr := new(b)
		if ptr == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *ptr != b {
			t.Errorf("expected %v, got %v", b, *ptr)
		}
	})
}

func TestValue(t *testing.T) {
	t.Attr("category", "utils")
	t.Attr("type", "unit")

	t.Run("non-nil pointer", func(t *testing.T) {
		s := "hello"
		ptr := &s
		result := Value(ptr, "default")
		if result != s {
			t.Errorf("expected %q, got %q", s, result)
		}
	})

	t.Run("nil pointer returns default", func(t *testing.T) {
		var ptr *string
		defaultVal := "default"
		result := Value(ptr, defaultVal)
		if result != defaultVal {
			t.Errorf("expected %q, got %q", defaultVal, result)
		}
	})

	t.Run("int with default", func(t *testing.T) {
		var ptr *int
		result := Value(ptr, 99)
		if result != 99 {
			t.Errorf("expected 99, got %d", result)
		}
	})

	t.Run("float64 non-nil", func(t *testing.T) {
		f := 2.71
		result := Value(&f, 0.0)
		if result != f {
			t.Errorf("expected %f, got %f", f, result)
		}
	})
}

func TestPtrOrNil(t *testing.T) {
	t.Attr("category", "utils")
	t.Attr("type", "unit")

	t.Run("zero value returns nil", func(t *testing.T) {
		if ptr := PtrOrNil(""); ptr != nil {
			t.Error("expected nil for empty string")
		}
		if ptr := PtrOrNil(0); ptr != nil {
			t.Error("expected nil for zero int")
		}
		if ptr := PtrOrNil(0.0); ptr != nil {
			t.Error("expected nil for zero float")
		}
		if ptr := PtrOrNil(false); ptr != nil {
			t.Error("expected nil for false bool")
		}
	})

	t.Run("non-zero value returns pointer", func(t *testing.T) {
		if ptr := PtrOrNil("hello"); ptr == nil || *ptr != "hello" {
			t.Error("expected non-nil pointer to 'hello'")
		}
		if ptr := PtrOrNil(42); ptr == nil || *ptr != 42 {
			t.Error("expected non-nil pointer to 42")
		}
		if ptr := PtrOrNil(true); ptr == nil || *ptr != true {
			t.Error("expected non-nil pointer to true")
		}
	})
}

func TestDeref(t *testing.T) {
	t.Attr("category", "utils")
	t.Attr("type", "unit")

	t.Run("non-nil pointer", func(t *testing.T) {
		s := "test"
		result := Deref(&s)
		if result != s {
			t.Errorf("expected %q, got %q", s, result)
		}
	})

	t.Run("nil pointer returns zero value", func(t *testing.T) {
		var ptr *string
		result := Deref(ptr)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("int zero value", func(t *testing.T) {
		var ptr *int
		result := Deref(ptr)
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})
}

func TestEqual(t *testing.T) {
	t.Attr("category", "utils")
	t.Attr("type", "unit")

	t.Run("both nil", func(t *testing.T) {
		var a, b *int
		if !Equal(a, b) {
			t.Error("expected true for both nil pointers")
		}
	})

	t.Run("one nil", func(t *testing.T) {
		a := new(5)
		var b *int
		if Equal(a, b) {
			t.Error("expected false when one pointer is nil")
		}
		if Equal(b, a) {
			t.Error("expected false when one pointer is nil (reversed)")
		}
	})

	t.Run("equal values", func(t *testing.T) {
		a := new(42)
		b := new(42)
		if !Equal(a, b) {
			t.Error("expected true for equal values")
		}
	})

	t.Run("different values", func(t *testing.T) {
		a := new(42)
		b := new(43)
		if Equal(a, b) {
			t.Error("expected false for different values")
		}
	})

	t.Run("string comparison", func(t *testing.T) {
		a := new("hello")
		b := new("hello")
		c := new("world")
		if !Equal(a, b) {
			t.Error("expected true for equal strings")
		}
		if Equal(a, c) {
			t.Error("expected false for different strings")
		}
	})
}

func TestClone(t *testing.T) {
	t.Attr("category", "utils")
	t.Attr("type", "unit")

	t.Run("nil pointer", func(t *testing.T) {
		var ptr *int
		clone := Clone(ptr)
		if clone != nil {
			t.Error("expected nil clone of nil pointer")
		}
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		original := new(42)
		clone := Clone(original)
		if clone == nil {
			t.Fatal("expected non-nil clone")
		}
		if *clone != *original {
			t.Errorf("expected equal values: %d != %d", *clone, *original)
		}
		if clone == original {
			t.Error("expected different pointer addresses")
		}
		// Modify clone to ensure independence
		*clone = 99
		if *original != 42 {
			t.Error("modifying clone affected original")
		}
	})

	t.Run("string clone", func(t *testing.T) {
		original := new("hello")
		clone := Clone(original)
		if clone == nil || *clone != *original {
			t.Error("string clone failed")
		}
		if clone == original {
			t.Error("expected different pointer addresses for string")
		}
	})
}

// Benchmarks
func BenchmarkPtr(b *testing.B) {
	b.Attr("category", "benchmarks")
	b.Attr("type", "performance")

	for i := 0; i < b.N; i++ {
		_ = new(42)
	}
}

func BenchmarkValue(b *testing.B) {
	b.Attr("category", "benchmarks")
	b.Attr("type", "performance")

	ptr := new(42)
	for i := 0; i < b.N; i++ {
		_ = Value(ptr, 0)
	}
}

func BenchmarkValueNil(b *testing.B) {
	b.Attr("category", "benchmarks")
	b.Attr("type", "performance")

	var ptr *int
	for i := 0; i < b.N; i++ {
		_ = Value(ptr, 99)
	}
}

func BenchmarkEqual(b *testing.B) {
	b.Attr("category", "benchmarks")
	b.Attr("type", "performance")

	a := new(42)
	b_ptr := new(42)
	for i := 0; i < b.N; i++ {
		_ = Equal(a, b_ptr)
	}
}

func BenchmarkClone(b *testing.B) {
	b.Attr("category", "benchmarks")
	b.Attr("type", "performance")

	original := new(42)
	for i := 0; i < b.N; i++ {
		_ = Clone(original)
	}
}
