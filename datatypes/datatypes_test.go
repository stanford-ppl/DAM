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

func TestFixedPointNegInt(t *testing.T) {
	fpt := FixedPointType{true, 4, 4}
	fp := FixedPoint{Tp: fpt}
	integer := big.NewInt(-3)
	fp.SetInt(integer)
	if fp.String() != "0b1101.0000" {
		t.Errorf("Expected -3 to be 0b1101.0000, got %s", fp.String())
	}
}

func TestFixedPointMin(t *testing.T) {
	fpt := FixedPointType{true, 32, 0}
	minimum := fpt.Min().ToInt()
	reference := big.NewInt(math.MinInt32)
	minimum.Sub(minimum, reference)
	if minimum.Int64() > 0 {
		t.Errorf("I32 Min was incorrect: got error of: %d (%d, %d)", minimum.Int64(), math.MinInt32, fpt.Min().ToInt())
	}
}

func TestFixedPointMax(t *testing.T) {
	fpt := FixedPointType{true, 32, 0}
	max := fpt.Max().ToInt()
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

func TestFixedPointValidateFail(t *testing.T) {
	fpt := FixedPointType{true, 32, 32}
	// Create a very large number outside of fpt
	val := FixedPoint{Tp: fpt}
	val.SetInt(big.NewInt(1 << 35))
	if val.Validate() {
		// This shouldn't validate!
		t.Errorf("%s should not have validated!", val)
	}
}

func TestFixedPointValidatePass(t *testing.T) {
	fpt := FixedPointType{true, 32, 32}
	// Create a very large number inside of range
	val := FixedPoint{Tp: fpt}
	val.SetInt(big.NewInt(1 << 30))
	if !val.Validate() {
		// This shouldn't validate!
		t.Errorf("%s should have validated!", val)
	}
}

func TestFixedPointAdd(t *testing.T) {
	fpt := FixedPointType{true, 32, 32}
	val1 := FixedPoint{Tp: fpt}
	val1.SetInt(big.NewInt(-3))
	val2 := FixedPoint{Tp: fpt}
	val2.SetInt(big.NewInt(5))
	val3 := FixedAdd(val1, val2)
	if val3.ToInt().Int64() != 2 {
		t.Errorf("Expected -3 + 5 = 2, got %s", val3.String())
	}
}

func TestFixedPointMul(t *testing.T) {
	fpt := FixedPointType{true, 8, 8}
	val1 := FixedPoint{Tp: fpt}
	val1.SetFloat(big.NewFloat(-3.5))
	t.Logf("Bit pattern for -3.5: %s\n", val1.String())
	t.Logf("Sign bit: %t", val1.Signbit())
	val2 := FixedPoint{Tp: fpt}
	val2.SetFloat(big.NewFloat(5))
	t.Logf("Sign bit: %t", val2.Signbit())
	val3 := FixedMulFull(val1, val2)
	t.Logf("-3.5 * 5 = %s\n", val3.ToFloat().String())
	flt, _ := val3.ToFloat().Float64()
	if math.Abs(flt+17.5) > 0.0001 {
		t.Errorf("Expected -3.5 * 5 = -17.5, got %s", val3.ToFloat().String())
	}
}
