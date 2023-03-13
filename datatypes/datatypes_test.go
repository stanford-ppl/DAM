package datatypes

import (
	"math"
	"math/big"
	"testing"
)

func TestFixedPointFloatRoundTrip(t *testing.T) {
	fpt := FixedPointType{false, 3, 13}
	fp := FixedPoint{Tp: fpt}
	flt := big.NewFloat(1.125)
	fp.SetFloat(flt)
	resultFlt := fp.ToFloat()

	resultFlt.Sub(resultFlt, flt)
	resultFlt.Abs(resultFlt)
	result, _ := resultFlt.Float64()
	if result > 0.0001 {
		t.Errorf("Float-to-Float roundtrip failed, got error of %f", result)
	} else {
		t.Logf("Float-to-Float roundtrip error: %f", result)
	}
}

func TestFixedPointIntRoundTrip(t *testing.T) {
	fpt := FixedPointType{false, 3, 13}
	fp := FixedPoint{Tp: fpt}
	integer := big.NewInt(1)
	fp.SetInt(integer)
	result := fp.ToInt()
	if result.Int64() != 1 {
		t.Errorf("Float-to-Float roundtrip failed, got error of %d", result)
	}
}

func TestFixedPointMin(t *testing.T) {
	fpt := FixedPointType{true, 32, 0}
	minimum := fpt.Min().ToInt()
	reference := big.NewInt(math.MinInt32)
	minimum.Sub(minimum, reference)
	if minimum.Int64() > 0 {
		t.Errorf("I32 Min was incorrect: got error of: %d", minimum.Int64())
	}
}

func TestFixedPointMax(t *testing.T) {
	fpt := FixedPointType{true, 32, 0}
	max := fpt.Min().ToInt()
	reference := big.NewInt(math.MaxInt32)
	max.Sub(max, reference)
	if max.Int64() > 0 {
		t.Errorf("I32 Min was incorrect: got error of: %d", max.Int64())
	}
}

func TestFixedPointMaxUnsigned(t *testing.T) {
	fpt := FixedPointType{false, 32, 0}
	max := fpt.Min().ToInt()
	reference := big.NewInt(math.MaxUint32)
	max.Sub(max, reference)
	if max.Int64() > 0 {
		t.Errorf("I32 Min was incorrect: got error of: %d", max.Int64())
	}
}
