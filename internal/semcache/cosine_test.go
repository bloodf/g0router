package semcache

import (
	"math"
	"testing"
)

func TestCosineIdentical(t *testing.T) {
	v := []float64{1, 2, 3}
	got := cosineSimilarity(v, v)
	if math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("cosine(identical) = %v, want 1.0", got)
	}
}

func TestCosineOrthogonal(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	got := cosineSimilarity(a, b)
	if math.Abs(got-0.0) > 1e-9 {
		t.Fatalf("cosine(orthogonal) = %v, want 0.0", got)
	}
}

func TestCosineOpposite(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{-1, -2, -3}
	got := cosineSimilarity(a, b)
	if math.Abs(got-(-1.0)) > 1e-9 {
		t.Fatalf("cosine(opposite) = %v, want -1.0", got)
	}
}

func TestCosineThresholdBoundary(t *testing.T) {
	// Exactly 0.95
	a := []float64{1, 0}
	b := []float64{0.95, math.Sqrt(1 - 0.95*0.95)}
	got := cosineSimilarity(a, b)
	if math.Abs(got-0.95) > 1e-6 {
		t.Fatalf("cosine = %v, want 0.95", got)
	}
	if !meetsThreshold(got, 0.95) {
		t.Fatal("expected 0.95 to meet threshold 0.95")
	}

	// Slightly below 0.95
	c := []float64{0.949, math.Sqrt(1 - 0.949*0.949)}
	got2 := cosineSimilarity(a, c)
	if meetsThreshold(got2, 0.95) {
		t.Fatalf("expected %v to not meet threshold 0.95", got2)
	}
}

func TestCosineEmptyVectors(t *testing.T) {
	got := cosineSimilarity([]float64{}, []float64{})
	if got != 0 {
		t.Fatalf("cosine(empty) = %v, want 0", got)
	}
}

func TestCosineDifferentLengths(t *testing.T) {
	got := cosineSimilarity([]float64{1, 2}, []float64{1, 2, 3})
	if got != 0 {
		t.Fatalf("cosine(different lengths) = %v, want 0", got)
	}
}

func TestCosineZeroVector(t *testing.T) {
	got := cosineSimilarity([]float64{0, 0, 0}, []float64{1, 2, 3})
	if got != 0 {
		t.Fatalf("cosine(zero) = %v, want 0", got)
	}
}
