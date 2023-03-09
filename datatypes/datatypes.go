package datatypes

import (
	"fmt"
	"math/big"
)

type FixPointType struct {
	Signed   bool
	Integer  uint
	Fraction uint
}

func (fpt *FixPointType) String() string {
	return fmt.Sprintf("Fix[%t, %d, %d]", fpt.Signed, fpt.Integer, fpt.Fraction)
}

func (fpt *FixPointType) Validate() bool {
	if fpt.Signed {
		return fpt.Integer > 0
	}

	return true
}

func (fmt *FixPointType) Min() *FixedPoint {
	result := new(FixedPoint)
	result.Tp = *fmt
	if fmt.Signed {
		// The minimum is -(1 << Integer)
		tmp := big.NewInt(1)
		tmp.Lsh(tmp, fmt.Integer)
		tmp.Rsh(tmp, 1)
		tmp.Neg(tmp)
		result.SetInt(tmp)
	} else {
		result.Underlying.SetInt64(0)
	}
	return result
}

func (fmt *FixPointType) Max() (result *FixedPoint) {
	result = new(FixedPoint)
	result.Tp = *fmt
	shift := fmt.Integer + fmt.Fraction
	if fmt.Signed {
		shift -= 1
	}
	result.Underlying.Lsh(big.NewInt(1), shift)
	result.Underlying.Sub(&result.Underlying, big.NewInt(1))
	return
}

type FixedPoint struct {
	Tp         FixPointType
	Underlying big.Int
}

func (fp *FixedPoint) SetInt(integer *big.Int) {
	if !fp.Tp.Signed && integer.Sign() < 0 {
		panic("Attempting to convert a negative integer to an unsigned FixedPoint")
	}
	fp.Underlying.Lsh(integer, fp.Tp.Fraction)
}

func (fp *FixedPoint) Validate() bool {
	return true
}

func (fp *FixedPoint) SetFloat(float *big.Float) {
	if !fp.Tp.Signed && float.Signbit() {
		panic("Attempting to convert a negative float to an unsigned FixedPoint")
	}
	numShift := big.NewInt(1)
	numShift.Lsh(numShift, fp.Tp.Fraction)
	intermediate := new(big.Float)
	intermediate.SetInt(numShift)
	intermediate.Mul(intermediate, float)
	intermediate.Int(&fp.Underlying)
}

func (fp *FixedPoint) ToRat() *big.Rat {
	result := new(big.Rat)
	denom := big.NewInt(1)
	denom.Lsh(denom, fp.Tp.Fraction)
	result.SetFrac(&fp.Underlying, denom)
	return result
}

func (fp *FixedPoint) ToFloat() *big.Float {
	result := new(big.Float)
	return result.SetRat(fp.ToRat())
}

func (fp *FixedPoint) ToInt() *big.Int {
	result := new(big.Int)
	result.Rsh(&fp.Underlying, fp.Tp.Fraction)
	return result
}
