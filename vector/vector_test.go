package vector

import (
	"testing"
	"github.com/stanford-ppl/DAM/datatypes/fixed"	
)

func TestNewVector(t *testing.T) {
	v := NewVector[datatypes.FixedPoint](5)
	if v.Width() != 5 {
		t.Errorf("Fail: Vector should be of length: %d but is instead of length: %d" , 5 , v.Width())
	} else {
		t.Logf("Pass")
	}
}

func TestGetAndSet(t *testing.T) {

	v := NewVector[datatypes.FixedPointType](5)

	v.set(0 , datatypes.FixedPointType{false, 3, 13})
	v.set(1 , datatypes.FixedPointType{true, 1, 21})

	var elem0 datatypes.FixedPointType = v.get(0)
	var elem1 datatypes.FixedPointType = v.get(1)

	if elem0.Signed != false || elem0.Integer != 3 || elem0.Fraction != 13 {
		t.Errorf("Fail:  Vector Element at index 0 != Element set at index 0")
	} else {
		t.Logf("Pass")
	}
	
	if elem1.Signed != true || elem1.Integer != 1 || elem1.Fraction != 21 {
		t.Errorf("Fail:  Vector Element at index 1 != Element set at index 1")
	} else {
		t.Logf("Pass")
	}
}


