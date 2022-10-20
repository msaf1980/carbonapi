package tests

import (
	"math"
	"math/rand"
)

func GetData(rangeSize int) []float64 {
	var data = make([]float64, rangeSize)
	var r = rand.New(rand.NewSource(99))
	for i := range data {
		data[i] = math.Floor(1000 * r.Float64())
	}

	return data
}
