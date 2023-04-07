package utils

import (
	"math/big"
	"testing"
)

func TestBigIntMax(t *testing.T) {
	a := big.NewInt(32)
	b := big.NewInt(64)
	max := new(big.Int)
	Max[*big.Int](a, b, max)
	if max.Int64() != 64 {
		t.Errorf("Expected Max(32, 64) = 64, got %d", max.Int64())
	}
}

func TestBigIntMin(t *testing.T) {
	a := big.NewInt(32)
	b := big.NewInt(64)
	min := new(big.Int)
	Min[*big.Int](a, b, min)
	if min.Int64() != 32 {
		t.Errorf("Expected Min(32, 64) = 32, got %d", min.Int64())
	}
}

func TestBigFloatMax(t *testing.T) {
	a := big.NewFloat(32.5)
	b := big.NewFloat(64.3)
	max := new(big.Float)
	Max[*big.Float](a, b, max)
	v, _ := max.Float64()
	if v != 64.3 {
		t.Errorf("Expected Max(32.5, 64.3) = 64.3, got %f", v)
	}
}
