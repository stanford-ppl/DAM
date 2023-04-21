package blas

import (
	"fmt"
	"math/rand"
	"testing"
)

// A random number in the interval (0, n]
func randLen(n int) uint {
	randInt := rand.Intn(n) + 1
	return uint(randInt)
}

// A random number in the interval [0, n)
func randIdx(n uint) uint {
	randInt := rand.Intn(int(n))
	return uint(randInt)
}

func TestMatrix(t *testing.T) {
	n := 10000

	for i := 0; i < 100; i++ {
		var rows, cols uint = randLen(n), randLen(n)
		var i, j uint = randIdx(rows), randIdx(cols)
		var val float64 = rand.Float64()

		fmt.Printf("rows = %d, cols = %d, i = %d, j = %d\n", rows, cols, i, j)
		mat := AllocMatrix(rows, cols)
		if mat.c_ptr == nil || mat.Rows() != rows || mat.Cols() != cols {
			t.Errorf("Fail: alloc returned null pointer")
		}

		mat.Set(i, j, val)
		val_ret := mat.Get(i, j)
		if val_ret != val {
			t.Logf("Fail: return value different from input")
		}

		mat.Free()
	}
}
