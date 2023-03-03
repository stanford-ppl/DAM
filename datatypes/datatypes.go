package datatypes

import (
	"fmt"
	"math/big"
)

type FixPointType struct {
	Signed bool
	Integer uint
	Fraction uint
}

func (fpt FixPointType) String() string {
	return fmt.Sprintf("Fix[%t, %d, %d]", fpt.Signed, fpt.Integer, fpt.Fraction)
}

// func (fpt FixPointType) maxValue() *big.Int {
// 	v := big.NewInt(1)
// 	var result big.Int
// 	result.Lsh(v, (fpt.Integer + 1))
// 	return result
// }

type FixedPoint struct {
	Tp FixPointType
	Underlying *big.Int
}

func (fp FixedPoint) SetInt(integer *big.Int) {
	if (!fp.Tp.Signed && integer.Sign() < 0) {
		panic("Attempting to convert a negative integer to an unsigned FixedPoint")
	}
	fp.Underlying.Lsh(integer, fp.Tp.Fraction)
}

func (fp FixedPoint) SetFloat(float *big.Float) {
	if (!fp.Tp.Signed && float.Signbit()) {
		panic("Attempting to convert a negative float to an unsigned FixedPoint")
	}
	numShift := big.NewInt(1)
	numShift.Lsh(numShift, fp.Tp.Fraction)
	intermediate := new(big.Float)
	intermediate.SetInt(numShift)
	intermediate.Mul(intermediate, float)
	intermediate.Int(fp.Underlying)
}

func (fp FixedPoint) ToRat() *big.Rat {
	result := new(big.Rat)
	denom := big.NewInt(1)
	denom.Lsh(denom, fp.Tp.Fraction)
	result.SetFrac(fp.Underlying, denom)
	return result
}

func (fp FixedPoint) ToFloat() *big.Float {
	result := new(big.Float)
	return result.SetRat(fp.ToRat())
}
