package types

import (
	"math"
	"strconv"
	"testing"
)

const eps = 0.0000000001

func compareFloat64(v1, v2 float64) bool {
	if math.IsNaN(v1) && math.IsNaN(v2) {
		return true
	}
	if math.IsInf(v1, 1) && math.IsInf(v2, 1) {
		return true
	}

	if math.IsInf(v1, 0) && math.IsInf(v2, 0) {
		return true
	}

	d := math.Abs(v1 - v2)
	return d < eps
}

func TestWindowed(t *testing.T) {
	nan := math.NaN()

	w := NewWindowed(5)
	tests := []struct {
		name      string
		values    []float64
		wantSum   float64
		wantSumSq float64
		wantStdev float64
		wantMean  float64
		wantMin   float64
		wantMax   float64
	}{
		{
			name:    "Empty",
			values:  []float64{},
			wantSum: 0, wantSumSq: 0, wantStdev: 0, wantMean: nan, wantMin: 0, wantMax: 0,
		},
		{
			name:    "First is NaN",
			values:  []float64{nan},
			wantSum: 0, wantSumSq: 0, wantStdev: 0, wantMean: nan, wantMin: 0, wantMax: 0,
		},
		{
			name:    "< 50% is NaN",
			values:  []float64{nan},
			wantSum: 0, wantSumSq: 0, wantStdev: 0, wantMean: nan, wantMin: 0, wantMax: 0,
		},
		{
			name:    "> 50% is NaN",
			values:  []float64{nan, nan},
			wantSum: 0, wantSumSq: 0, wantStdev: 0, wantMean: nan, wantMin: 0, wantMax: 0,
		},
		{
			name:    "all is NaN",
			values:  []float64{nan},
			wantSum: 0, wantSumSq: 0, wantStdev: 0, wantMean: nan, wantMin: nan, wantMax: nan,
		},
		{
			name:    "{ 1 }",
			values:  []float64{1},
			wantSum: 1, wantSumSq: 1, wantStdev: 0, wantMean: 1, wantMin: 1, wantMax: 1,
		},
		{
			name:    "{ 1, 2 }",
			values:  []float64{2},
			wantSum: 3, wantSumSq: 5, wantStdev: 0.5, wantMean: 1.5, wantMin: 1, wantMax: 2,
		},
		{
			name:    "{ 1, 2, 3 }",
			values:  []float64{3},
			wantSum: 6, wantSumSq: 14, wantStdev: 0.8164965809, wantMean: 2, wantMin: 1, wantMax: 3,
		},
		{
			name:    "{ 2, 3, 2, 3, 4 }",
			values:  []float64{2, 3, 4},
			wantSum: 14, wantSumSq: 42, wantStdev: 0.7483314773, wantMean: 2.8, wantMin: 2, wantMax: 4,
		},
		{
			name:    "{ 3, 2, 3, 4, 1 }",
			values:  []float64{1},
			wantSum: 13, wantSumSq: 39, wantStdev: 1.0198039027, wantMean: 2.6, wantMin: 1, wantMax: 4,
		},
		{
			name:    "{ 2, 3, 4, 1, NaN }",
			values:  []float64{nan},
			wantSum: 10, wantSumSq: 30, wantStdev: 1.1180339887, wantMean: 2.5, wantMin: 1, wantMax: 4,
		},
	}
	for n, tt := range tests {
		t.Run(tt.name+"#"+strconv.Itoa(n), func(t *testing.T) {
			for _, v := range tt.values {
				w.Push(v)
			}
			n := w.Sum()
			if !compareFloat64(tt.wantSum, n) {
				t.Errorf("Windowed.Sum() want %v, got %v", tt.wantSum, n)
			}
			n = w.SumSQ()
			if !compareFloat64(tt.wantSumSq, n) {
				t.Errorf("Windowed.SumSQ() want %v, got %v", tt.wantSumSq, n)
			}
			n = w.Stdev()
			if !compareFloat64(tt.wantStdev, n) {
				t.Errorf("Windowed.Stdev() want %v, got %v", tt.wantStdev, n)
			}
			n = w.Min()
			if !compareFloat64(tt.wantMin, n) {
				t.Errorf("Windowed.Min() want %v, got %v", tt.wantMin, n)
			}
			n = w.Max()
			if !compareFloat64(tt.wantMax, n) {
				t.Errorf("Windowed.Max() want %v, got %v", tt.wantMax, n)
			}
			n = w.Mean()
			if !compareFloat64(tt.wantMean, n) {
				t.Errorf("Windowed.Mean() want %v, got %v", tt.wantMean, n)
			}

		})
	}
}
