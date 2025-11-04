// Package vector provides generic vector operations for RAG embeddings
package vector

import (
	"fmt"
	"math"
)

// Float is a constraint for floating-point types.
type Float interface {
	~float32 | ~float64
}

// Vector represents a generic embedding vector.
// Supports both float32 and float64 for flexibility and precision.
//
// Example usage:
//
//	vec1 := vector.New[float32]([]float32{1.0, 2.0, 3.0})
//	vec2 := vector.New[float32]([]float32{4.0, 5.0, 6.0})
//
//	similarity := vec1.CosineSimilarity(vec2)
//	fmt.Printf("Similarity: %.4f\n", similarity)
type Vector[T Float] []T

// New creates a new vector from a slice.
func New[T Float](data []T) Vector[T] {
	return Vector[T](data)
}

// Len returns the dimensionality of the vector.
func (v Vector[T]) Len() int {
	return len(v)
}

// Dot computes the dot product with another vector.
//
// Example:
//
//	v1 := vector.New[float64]([]float64{1, 2, 3})
//	v2 := vector.New[float64]([]float64{4, 5, 6})
//	dot := v1.Dot(v2) // 1*4 + 2*5 + 3*6 = 32
func (v Vector[T]) Dot(other Vector[T]) (T, error) {
	if len(v) != len(other) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(v), len(other))
	}

	var sum T
	for i := range v {
		sum += v[i] * other[i]
	}
	return sum, nil
}

// Magnitude computes the Euclidean norm (L2 norm) of the vector.
//
// Example:
//
//	v := vector.New[float64]([]float64{3, 4})
//	mag := v.Magnitude() // sqrt(3^2 + 4^2) = 5
func (v Vector[T]) Magnitude() T {
	var sum T
	for _, val := range v {
		sum += val * val
	}
	return T(math.Sqrt(float64(sum)))
}

// Normalize returns a unit vector (magnitude = 1).
//
// Example:
//
//	v := vector.New[float64]([]float64{3, 4})
//	normalized := v.Normalize() // [0.6, 0.8]
func (v Vector[T]) Normalize() Vector[T] {
	mag := v.Magnitude()
	if mag == 0 {
		return v
	}

	result := make(Vector[T], len(v))
	for i, val := range v {
		result[i] = val / mag
	}
	return result
}

// CosineSimilarity computes cosine similarity with another vector.
// Returns a value between -1 (opposite) and 1 (identical).
//
// Example:
//
//	v1 := vector.New[float64]([]float64{1, 0, 0})
//	v2 := vector.New[float64]([]float64{1, 0, 0})
//	sim := v1.CosineSimilarity(v2) // 1.0 (identical)
func (v Vector[T]) CosineSimilarity(other Vector[T]) (float64, error) {
	dot, err := v.Dot(other)
	if err != nil {
		return 0, err
	}

	mag1 := v.Magnitude()
	mag2 := other.Magnitude()

	if mag1 == 0 || mag2 == 0 {
		return 0, fmt.Errorf("cannot compute similarity: zero magnitude vector")
	}

	return float64(dot) / (float64(mag1) * float64(mag2)), nil
}

// EuclideanDistance computes L2 distance to another vector.
//
// Example:
//
//	v1 := vector.New[float64]([]float64{0, 0})
//	v2 := vector.New[float64]([]float64{3, 4})
//	dist := v1.EuclideanDistance(v2) // 5.0
func (v Vector[T]) EuclideanDistance(other Vector[T]) (float64, error) {
	if len(v) != len(other) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(v), len(other))
	}

	var sum T
	for i := range v {
		diff := v[i] - other[i]
		sum += diff * diff
	}
	return math.Sqrt(float64(sum)), nil
}

// ManhattanDistance computes L1 distance to another vector.
//
// Example:
//
//	v1 := vector.New[float64]([]float64{1, 2})
//	v2 := vector.New[float64]([]float64{4, 6})
//	dist := v1.ManhattanDistance(v2) // |1-4| + |2-6| = 7.0
func (v Vector[T]) ManhattanDistance(other Vector[T]) (float64, error) {
	if len(v) != len(other) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(v), len(other))
	}

	var sum float64
	for i := range v {
		sum += math.Abs(float64(v[i] - other[i]))
	}
	return sum, nil
}

// Add performs element-wise addition.
//
// Example:
//
//	v1 := vector.New[float64]([]float64{1, 2})
//	v2 := vector.New[float64]([]float64{3, 4})
//	result := v1.Add(v2) // [4, 6]
func (v Vector[T]) Add(other Vector[T]) (Vector[T], error) {
	if len(v) != len(other) {
		return nil, fmt.Errorf("vector dimensions mismatch: %d != %d", len(v), len(other))
	}

	result := make(Vector[T], len(v))
	for i := range v {
		result[i] = v[i] + other[i]
	}
	return result, nil
}

// Sub performs element-wise subtraction.
//
// Example:
//
//	v1 := vector.New[float64]([]float64{5, 7})
//	v2 := vector.New[float64]([]float64{2, 3})
//	result := v1.Sub(v2) // [3, 4]
func (v Vector[T]) Sub(other Vector[T]) (Vector[T], error) {
	if len(v) != len(other) {
		return nil, fmt.Errorf("vector dimensions mismatch: %d != %d", len(v), len(other))
	}

	result := make(Vector[T], len(v))
	for i := range v {
		result[i] = v[i] - other[i]
	}
	return result, nil
}

// Scale multiplies all elements by a scalar.
//
// Example:
//
//	v := vector.New[float64]([]float64{1, 2, 3})
//	scaled := v.Scale(2) // [2, 4, 6]
func (v Vector[T]) Scale(scalar T) Vector[T] {
	result := make(Vector[T], len(v))
	for i, val := range v {
		result[i] = val * scalar
	}
	return result
}

// Mean computes the average of all elements.
func (v Vector[T]) Mean() T {
	if len(v) == 0 {
		return 0
	}

	var sum T
	for _, val := range v {
		sum += val
	}
	return sum / T(len(v))
}

// Max returns the maximum element.
func (v Vector[T]) Max() T {
	if len(v) == 0 {
		return 0
	}

	max := v[0]
	for _, val := range v[1:] {
		if val > max {
			max = val
		}
	}
	return max
}

// Min returns the minimum element.
func (v Vector[T]) Min() T {
	if len(v) == 0 {
		return 0
	}

	min := v[0]
	for _, val := range v[1:] {
		if val < min {
			min = val
		}
	}
	return min
}

// Clone creates a deep copy of the vector.
func (v Vector[T]) Clone() Vector[T] {
	result := make(Vector[T], len(v))
	copy(result, v)
	return result
}

// ToFloat32 converts vector to float32 representation.
func (v Vector[T]) ToFloat32() []float32 {
	result := make([]float32, len(v))
	for i, val := range v {
		result[i] = float32(val)
	}
	return result
}

// ToFloat64 converts vector to float64 representation.
func (v Vector[T]) ToFloat64() []float64 {
	result := make([]float64, len(v))
	for i, val := range v {
		result[i] = float64(val)
	}
	return result
}

// Batch operations for processing multiple vectors

// BatchCosineSimilarity computes cosine similarity for multiple vectors.
// Returns similarities in the same order as input vectors.
//
// Example:
//
//	query := vector.New[float64]([]float64{1, 0, 0})
//	candidates := []vector.Vector[float64]{
//	    vector.New[float64]([]float64{1, 0, 0}),
//	    vector.New[float64]([]float64{0, 1, 0}),
//	}
//	similarities := vector.BatchCosineSimilarity(query, candidates)
//	// [1.0, 0.0]
func BatchCosineSimilarity[T Float](query Vector[T], candidates []Vector[T]) ([]float64, error) {
	results := make([]float64, len(candidates))
	for i, candidate := range candidates {
		sim, err := query.CosineSimilarity(candidate)
		if err != nil {
			return nil, fmt.Errorf("failed to compute similarity for vector %d: %w", i, err)
		}
		results[i] = sim
	}
	return results, nil
}

// TopK returns indices of k vectors with highest similarity to query.
//
// Example:
//
//	query := vector.New[float64]([]float64{1, 0, 0})
//	candidates := []vector.Vector[float64]{...}
//	topIndices := vector.TopK(query, candidates, 5)
//	// Returns indices of 5 most similar vectors
func TopK[T Float](query Vector[T], candidates []Vector[T], k int) ([]int, error) {
	if k > len(candidates) {
		k = len(candidates)
	}

	type scored struct {
		index int
		score float64
	}

	scores := make([]scored, len(candidates))
	for i, candidate := range candidates {
		sim, err := query.CosineSimilarity(candidate)
		if err != nil {
			return nil, fmt.Errorf("failed to compute similarity for vector %d: %w", i, err)
		}
		scores[i] = scored{index: i, score: sim}
	}

	// Simple sort for top k (for large datasets, use heap)
	for i := 0; i < k; i++ {
		maxIdx := i
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[maxIdx].score {
				maxIdx = j
			}
		}
		scores[i], scores[maxIdx] = scores[maxIdx], scores[i]
	}

	result := make([]int, k)
	for i := 0; i < k; i++ {
		result[i] = scores[i].index
	}
	return result, nil
}

// AverageVector computes the centroid of multiple vectors.
//
// Example:
//
//	vectors := []vector.Vector[float64]{
//	    vector.New[float64]([]float64{1, 2}),
//	    vector.New[float64]([]float64{3, 4}),
//	}
//	avg := vector.AverageVector(vectors) // [2, 3]
func AverageVector[T Float](vectors []Vector[T]) (Vector[T], error) {
	if len(vectors) == 0 {
		return nil, fmt.Errorf("cannot average empty vector list")
	}

	dim := len(vectors[0])
	result := make(Vector[T], dim)

	for _, vec := range vectors {
		if len(vec) != dim {
			return nil, fmt.Errorf("vector dimension mismatch: expected %d, got %d", dim, len(vec))
		}
		for i, val := range vec {
			result[i] += val
		}
	}

	scale := T(1.0) / T(len(vectors))
	return result.Scale(scale), nil
}

// Centroid is an alias for AverageVector.
func Centroid[T Float](vectors []Vector[T]) (Vector[T], error) {
	return AverageVector(vectors)
}
