package vector

import (
	"math"
	"testing"
)

const epsilon = 1e-6

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestNew(t *testing.T) {
	v := New[float64]([]float64{1, 2, 3})
	if v.Len() != 3 {
		t.Errorf("expected length 3, got %d", v.Len())
	}
}

func TestDot(t *testing.T) {
	v1 := New[float64]([]float64{1, 2, 3})
	v2 := New[float64]([]float64{4, 5, 6})

	dot, err := v1.Dot(v2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 1*4 + 2*5 + 3*6 // 32
	if dot != float64(expected) {
		t.Errorf("expected %v, got %v", expected, dot)
	}
}

func TestDot_DimensionMismatch(t *testing.T) {
	v1 := New[float64]([]float64{1, 2})
	v2 := New[float64]([]float64{1, 2, 3})

	_, err := v1.Dot(v2)
	if err == nil {
		t.Error("expected error for dimension mismatch")
	}
}

func TestMagnitude(t *testing.T) {
	v := New[float64]([]float64{3, 4})
	mag := v.Magnitude()

	expected := 5.0 // sqrt(3^2 + 4^2)
	if !almostEqual(float64(mag), expected) {
		t.Errorf("expected %.6f, got %.6f", expected, mag)
	}
}

func TestNormalize(t *testing.T) {
	v := New[float64]([]float64{3, 4})
	normalized := v.Normalize()

	mag := normalized.Magnitude()
	if !almostEqual(float64(mag), 1.0) {
		t.Errorf("expected magnitude 1.0, got %.6f", mag)
	}

	if !almostEqual(float64(normalized[0]), 0.6) {
		t.Errorf("expected first element 0.6, got %.6f", normalized[0])
	}
	if !almostEqual(float64(normalized[1]), 0.8) {
		t.Errorf("expected second element 0.8, got %.6f", normalized[1])
	}
}

func TestCosineSimilarity(t *testing.T) {
	t.Run("identical vectors", func(t *testing.T) {
		v1 := New[float64]([]float64{1, 0, 0})
		v2 := New[float64]([]float64{1, 0, 0})

		sim, err := v1.CosineSimilarity(v2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !almostEqual(sim, 1.0) {
			t.Errorf("expected 1.0, got %.6f", sim)
		}
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		v1 := New[float64]([]float64{1, 0, 0})
		v2 := New[float64]([]float64{0, 1, 0})

		sim, err := v1.CosineSimilarity(v2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !almostEqual(sim, 0.0) {
			t.Errorf("expected 0.0, got %.6f", sim)
		}
	})

	t.Run("opposite vectors", func(t *testing.T) {
		v1 := New[float64]([]float64{1, 0, 0})
		v2 := New[float64]([]float64{-1, 0, 0})

		sim, err := v1.CosineSimilarity(v2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !almostEqual(sim, -1.0) {
			t.Errorf("expected -1.0, got %.6f", sim)
		}
	})
}

func TestEuclideanDistance(t *testing.T) {
	v1 := New[float64]([]float64{0, 0})
	v2 := New[float64]([]float64{3, 4})

	dist, err := v1.EuclideanDistance(v2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 5.0
	if !almostEqual(dist, expected) {
		t.Errorf("expected %.6f, got %.6f", expected, dist)
	}
}

func TestManhattanDistance(t *testing.T) {
	v1 := New[float64]([]float64{1, 2})
	v2 := New[float64]([]float64{4, 6})

	dist, err := v1.ManhattanDistance(v2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 7.0 // |1-4| + |2-6|
	if !almostEqual(dist, expected) {
		t.Errorf("expected %.6f, got %.6f", expected, dist)
	}
}

func TestAdd(t *testing.T) {
	v1 := New[float64]([]float64{1, 2})
	v2 := New[float64]([]float64{3, 4})

	result, err := v1.Add(v2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result[0] != 4 || result[1] != 6 {
		t.Errorf("expected [4, 6], got [%.1f, %.1f]", result[0], result[1])
	}
}

func TestSub(t *testing.T) {
	v1 := New[float64]([]float64{5, 7})
	v2 := New[float64]([]float64{2, 3})

	result, err := v1.Sub(v2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result[0] != 3 || result[1] != 4 {
		t.Errorf("expected [3, 4], got [%.1f, %.1f]", result[0], result[1])
	}
}

func TestScale(t *testing.T) {
	v := New[float64]([]float64{1, 2, 3})
	scaled := v.Scale(2)

	expected := []float64{2, 4, 6}
	for i, val := range scaled {
		if val != expected[i] {
			t.Errorf("at index %d: expected %.1f, got %.1f", i, expected[i], val)
		}
	}
}

func TestMean(t *testing.T) {
	v := New[float64]([]float64{1, 2, 3, 4, 5})
	mean := v.Mean()

	expected := 3.0
	if mean != expected {
		t.Errorf("expected %.1f, got %.1f", expected, mean)
	}
}

func TestMaxMin(t *testing.T) {
	v := New[float64]([]float64{3, 1, 4, 1, 5, 9, 2, 6})

	max := v.Max()
	if max != 9 {
		t.Errorf("expected max 9, got %.1f", max)
	}

	min := v.Min()
	if min != 1 {
		t.Errorf("expected min 1, got %.1f", min)
	}
}

func TestClone(t *testing.T) {
	v := New[float64]([]float64{1, 2, 3})
	clone := v.Clone()

	if len(clone) != len(v) {
		t.Errorf("expected same length")
	}

	// Modify clone
	clone[0] = 999

	if v[0] == 999 {
		t.Error("modifying clone affected original")
	}
}

func TestToFloat32(t *testing.T) {
	v := New[float64]([]float64{1.5, 2.5, 3.5})
	f32 := v.ToFloat32()

	if len(f32) != 3 {
		t.Errorf("expected length 3, got %d", len(f32))
	}
	if f32[0] != 1.5 {
		t.Errorf("expected 1.5, got %.1f", f32[0])
	}
}

func TestToFloat64(t *testing.T) {
	v := New[float32]([]float32{1.5, 2.5, 3.5})
	f64 := v.ToFloat64()

	if len(f64) != 3 {
		t.Errorf("expected length 3, got %d", len(f64))
	}
	if !almostEqual(f64[0], 1.5) {
		t.Errorf("expected 1.5, got %.1f", f64[0])
	}
}

func TestBatchCosineSimilarity(t *testing.T) {
	query := New[float64]([]float64{1, 0, 0})
	candidates := []Vector[float64]{
		New[float64]([]float64{1, 0, 0}),
		New[float64]([]float64{0, 1, 0}),
		New[float64]([]float64{0, 0, 1}),
	}

	similarities, err := BatchCosineSimilarity(query, candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(similarities) != 3 {
		t.Fatalf("expected 3 similarities, got %d", len(similarities))
	}

	// First should be 1.0 (identical)
	if !almostEqual(similarities[0], 1.0) {
		t.Errorf("expected 1.0, got %.6f", similarities[0])
	}

	// Second and third should be 0.0 (orthogonal)
	if !almostEqual(similarities[1], 0.0) {
		t.Errorf("expected 0.0, got %.6f", similarities[1])
	}
}

func TestTopK(t *testing.T) {
	query := New[float64]([]float64{1, 0, 0})
	candidates := []Vector[float64]{
		New[float64]([]float64{1, 0, 0}),    // similarity: 1.0
		New[float64]([]float64{0.9, 0.1, 0}), // similarity: ~0.995
		New[float64]([]float64{0, 1, 0}),    // similarity: 0.0
		New[float64]([]float64{0.8, 0.2, 0}), // similarity: ~0.970
	}

	topIndices, err := TopK(query, candidates, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(topIndices) != 2 {
		t.Fatalf("expected 2 indices, got %d", len(topIndices))
	}

	// First result should be index 0 (highest similarity)
	if topIndices[0] != 0 {
		t.Errorf("expected index 0, got %d", topIndices[0])
	}
}

func TestAverageVector(t *testing.T) {
	vectors := []Vector[float64]{
		New[float64]([]float64{1, 2}),
		New[float64]([]float64{3, 4}),
	}

	avg, err := AverageVector(vectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if avg[0] != 2 || avg[1] != 3 {
		t.Errorf("expected [2, 3], got [%.1f, %.1f]", avg[0], avg[1])
	}
}

func TestCentroid(t *testing.T) {
	vectors := []Vector[float64]{
		New[float64]([]float64{0, 0}),
		New[float64]([]float64{2, 0}),
		New[float64]([]float64{2, 2}),
		New[float64]([]float64{0, 2}),
	}

	centroid, err := Centroid(vectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Centroid of square should be center
	if centroid[0] != 1 || centroid[1] != 1 {
		t.Errorf("expected [1, 1], got [%.1f, %.1f]", centroid[0], centroid[1])
	}
}

// Benchmarks
func BenchmarkDot(b *testing.B) {
	v1 := New[float64](make([]float64, 1024))
	v2 := New[float64](make([]float64, 1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v1.Dot(v2)
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	v1 := New[float64](make([]float64, 1024))
	v2 := New[float64](make([]float64, 1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v1.CosineSimilarity(v2)
	}
}

func BenchmarkNormalize(b *testing.B) {
	v := New[float64](make([]float64, 1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Normalize()
	}
}

func BenchmarkBatchCosineSimilarity(b *testing.B) {
	query := New[float64](make([]float64, 1024))
	candidates := make([]Vector[float64], 100)
	for i := range candidates {
		candidates[i] = New[float64](make([]float64, 1024))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = BatchCosineSimilarity(query, candidates)
	}
}

func BenchmarkTopK(b *testing.B) {
	query := New[float64](make([]float64, 1024))
	candidates := make([]Vector[float64], 1000)
	for i := range candidates {
		candidates[i] = New[float64](make([]float64, 1024))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = TopK(query, candidates, 10)
	}
}

