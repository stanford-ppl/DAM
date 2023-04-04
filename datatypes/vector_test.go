package datatypes

import (
	"testing"
)

func TestNewVector(t *testing.T) {
	v := NewVector[FixedPoint](5)
	if v.Width() != 5 {
		t.Errorf("Fail: Vector should be of length: %d but is instead of length: %d", 5, v.Width())
	} else {
		t.Logf("Pass")
	}
}

func TestGetAndSet(t *testing.T) {
	v := NewVector[FixedPoint](5)

	v.Set(0, FixedPoint{Tp: FixedPointType{false, 3, 13}})
	v.Set(1, FixedPoint{Tp: FixedPointType{true, 1, 21}})

	elem0 := v.Get(0)
	elem1 := v.Get(1)

	if elem0.Tp.Signed != false || elem0.Tp.Integer != 3 || elem0.Tp.Fraction != 13 {
		t.Errorf("Fail:  Vector Element at index 0 != Element set at index 0")
	} else {
		t.Logf("Pass")
	}

	if elem1.Tp.Signed != true || elem1.Tp.Integer != 1 || elem1.Tp.Fraction != 21 {
		t.Errorf("Fail:  Vector Element at index 1 != Element set at index 1")
	} else {
		t.Logf("Pass")
	}
}
