package types

import (
	"errors"
	"testing"
)

func TestConfigMap(t *testing.T) {
	type ServerConfig = ConfigMap[string]
	config := ServerConfig{
		"host": "localhost",
		"port": "8080",
	}

	if config["host"] != "localhost" {
		t.Errorf("expected localhost, got %s", config["host"])
	}

	type NumericConfig = ConfigMap[int]
	limits := NumericConfig{
		"max_connections": 100,
	}

	if limits["max_connections"] != 100 {
		t.Errorf("expected 100, got %d", limits["max_connections"])
	}
}

func TestMetadata(t *testing.T) {
	type UserMetadata = Metadata[string]
	meta := UserMetadata{
		"region": "us-east-1",
		"tier":   "premium",
	}

	if meta["region"] != "us-east-1" {
		t.Errorf("expected us-east-1, got %s", meta["region"])
	}
}

func TestIDMap(t *testing.T) {
	type User struct {
		ID   string
		Name string
	}

	type UserCache = IDMap[*User]
	cache := UserCache{
		"user-123": &User{ID: "123", Name: "John"},
		"user-456": &User{ID: "456", Name: "Jane"},
	}

	if cache["user-123"].Name != "John" {
		t.Errorf("expected John, got %s", cache["user-123"].Name)
	}
}

func TestResult(t *testing.T) {
	t.Run("success result", func(t *testing.T) {
		result := Result[int]{Value: 42, Ok: true}
		if !result.Ok {
			t.Error("expected Ok to be true")
		}
		if result.Value != 42 {
			t.Errorf("expected 42, got %d", result.Value)
		}
		if result.Err != nil {
			t.Errorf("expected nil error, got %v", result.Err)
		}
	})

	t.Run("error result", func(t *testing.T) {
		err := errors.New("test error")
		result := Result[int]{Err: err, Ok: false}
		if result.Ok {
			t.Error("expected Ok to be false")
		}
		if result.Err != err {
			t.Errorf("expected %v, got %v", err, result.Err)
		}
	})
}

func TestOption(t *testing.T) {
	t.Run("Some", func(t *testing.T) {
		opt := Some(42)
		if !opt.IsSome() {
			t.Error("expected IsSome to be true")
		}
		if opt.IsNone() {
			t.Error("expected IsNone to be false")
		}
		if opt.Unwrap() != 42 {
			t.Errorf("expected 42, got %d", opt.Unwrap())
		}
	})

	t.Run("None", func(t *testing.T) {
		opt := None[int]()
		if opt.IsSome() {
			t.Error("expected IsSome to be false")
		}
		if !opt.IsNone() {
			t.Error("expected IsNone to be true")
		}
	})

	t.Run("UnwrapOr", func(t *testing.T) {
		some := Some(42)
		if some.UnwrapOr(99) != 42 {
			t.Error("expected original value")
		}

		none := None[int]()
		if none.UnwrapOr(99) != 99 {
			t.Error("expected default value")
		}
	})

	t.Run("UnwrapOrElse", func(t *testing.T) {
		some := Some(42)
		result := some.UnwrapOrElse(func() int { return 99 })
		if result != 42 {
			t.Error("expected original value")
		}

		none := None[int]()
		result = none.UnwrapOrElse(func() int { return 99 })
		if result != 99 {
			t.Error("expected computed default")
		}
	})

	t.Run("Map", func(t *testing.T) {
		opt := Some(42)
		mapped := opt.Map(func(v int) int { return v * 2 })
		if mapped.Unwrap() != 84 {
			t.Errorf("expected 84, got %d", mapped.Unwrap())
		}

		none := None[int]()
		mapped = none.Map(func(v int) int { return v * 2 })
		if !mapped.IsNone() {
			t.Error("expected None after mapping None")
		}
	})

	t.Run("Unwrap panics on None", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got none")
			}
		}()

		none := None[int]()
		none.Unwrap() // Should panic
	})
}

func TestStringSet(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		set := NewStringSet()
		set.Add("apple")
		set.Add("banana")
		set.Add("apple") // Duplicate

		if set.Size() != 2 {
			t.Errorf("expected size 2, got %d", set.Size())
		}

		if !set.Contains("apple") {
			t.Error("expected to contain apple")
		}

		if set.Contains("orange") {
			t.Error("expected not to contain orange")
		}

		set.Remove("apple")
		if set.Contains("apple") {
			t.Error("expected apple to be removed")
		}
	})

	t.Run("constructor with items", func(t *testing.T) {
		set := NewStringSet("a", "b", "c")
		if set.Size() != 3 {
			t.Errorf("expected size 3, got %d", set.Size())
		}
	})

	t.Run("Items", func(t *testing.T) {
		set := NewStringSet("x", "y", "z")
		items := set.Items()
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}
	})

	t.Run("Clear", func(t *testing.T) {
		set := NewStringSet("a", "b", "c")
		set.Clear()
		if set.Size() != 0 {
			t.Errorf("expected size 0 after clear, got %d", set.Size())
		}
	})
}

func TestSet(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		set := NewSet[int]()
		set.Add(1)
		set.Add(2)
		set.Add(1) // Duplicate

		if set.Size() != 2 {
			t.Errorf("expected size 2, got %d", set.Size())
		}

		if !set.Contains(1) {
			t.Error("expected to contain 1")
		}

		set.Remove(1)
		if set.Contains(1) {
			t.Error("expected 1 to be removed")
		}
	})

	t.Run("Union", func(t *testing.T) {
		set1 := NewSet(1, 2, 3)
		set2 := NewSet(3, 4, 5)
		union := set1.Union(set2)

		if union.Size() != 5 {
			t.Errorf("expected size 5, got %d", union.Size())
		}
	})

	t.Run("Intersection", func(t *testing.T) {
		set1 := NewSet(1, 2, 3)
		set2 := NewSet(2, 3, 4)
		intersection := set1.Intersection(set2)

		if intersection.Size() != 2 {
			t.Errorf("expected size 2, got %d", intersection.Size())
		}
		if !intersection.Contains(2) || !intersection.Contains(3) {
			t.Error("intersection should contain 2 and 3")
		}
	})

	t.Run("Difference", func(t *testing.T) {
		set1 := NewSet(1, 2, 3)
		set2 := NewSet(2, 3, 4)
		diff := set1.Difference(set2)

		if diff.Size() != 1 {
			t.Errorf("expected size 1, got %d", diff.Size())
		}
		if !diff.Contains(1) {
			t.Error("difference should contain 1")
		}
	})
}

func TestCache(t *testing.T) {
	type UserCache = Cache[string, string]
	cache := make(UserCache)

	cache["user-123"] = "John"
	cache["user-456"] = "Jane"

	if cache["user-123"] != "John" {
		t.Errorf("expected John, got %s", cache["user-123"])
	}
}

func TestCounter(t *testing.T) {
	type WordCounter = Counter[string]
	counter := make(WordCounter)

	counter["hello"]++
	counter["hello"]++
	counter["world"]++

	if counter["hello"] != 2 {
		t.Errorf("expected 2, got %d", counter["hello"])
	}
	if counter["world"] != 1 {
		t.Errorf("expected 1, got %d", counter["world"])
	}
}

func TestIndex(t *testing.T) {
	type TagIndex = Index[string, string]
	index := make(TagIndex)

	index["golang"] = []string{"doc1", "doc2"}
	index["rust"] = []string{"doc3"}

	if len(index["golang"]) != 2 {
		t.Errorf("expected 2 docs for golang, got %d", len(index["golang"]))
	}
}

// Benchmarks
func BenchmarkSet_Add(b *testing.B) {
	set := NewSet[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Add(i)
	}
}

func BenchmarkSet_Contains(b *testing.B) {
	set := NewSet[int]()
	for i := range 1000 {
		set.Add(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Contains(i % 1000)
	}
}

func BenchmarkOption_Unwrap(b *testing.B) {
	opt := Some(42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = opt.Unwrap()
	}
}

func BenchmarkOption_UnwrapOr(b *testing.B) {
	opt := None[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = opt.UnwrapOr(99)
	}
}
