package datatypes

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

type FixedPointType struct {
	Signed   bool
	Integer  uint
	Fraction uint
}

func (fpt FixedPointType) String() string {
	return fmt.Sprintf("Fix[%t, %d, %d]", fpt.Signed, fpt.Integer, fpt.Fraction)
}

func (fpt FixedPointType) NBits() uint {
	return fpt.Integer + fpt.Fraction
}

func (fpt FixedPointType) Validate() bool {
	if fpt.Signed {
		return fpt.Integer > 0
	}

	return true
}

func (fpt FixedPointType) Min() *FixedPoint {
	result := new(FixedPoint)
	result.Tp = fpt
	if fpt.Signed {
		// The minimum is -(1 << Integer)
		tmp := big.NewInt(1)
		tmp.Lsh(tmp, fpt.Integer+fpt.Fraction-1)
		result.Underlying.Set(tmp)
	} else {
		result.Underlying.SetInt64(0)
	}
	return result
}

func (fmt FixedPointType) Max() (result *FixedPoint) {
	result = new(FixedPoint)
	result.Tp = fmt
	shift := fmt.Integer + fmt.Fraction
	if fmt.Signed {
		shift -= 1
	}
	result.Underlying.Lsh(big.NewInt(1), shift)
	result.Underlying.Sub(&result.Underlying, big.NewInt(1))
	return
}

type FixedPoint struct {
	Tp         FixedPointType
	Underlying big.Int
}

func (fp FixedPoint) Validate() bool {
	min := fp.Tp.Min()
	if Cmp(fp, *min) < 0 {
		fmt.Printf("%s < %s", fp, min)
		return false
	}
	max := fp.Tp.Max()
	if Cmp(fp, *max) > 0 {
		fmt.Printf("%s > %s", fp, max)
		return false
	}
	return true
}

func (fp FixedPoint) Size() *big.Int {
	return big.NewInt(int64(fp.Tp.NBits()))
}

func (fp FixedPoint) Payload() any {
	return fp
}

func (fp *FixedPoint) NegInPlace() {
	if !fp.Tp.Signed {
		panic("Cannot negate an unsigned number!")
	}
	// To negate a 2-s complement number taking N bits, we subtract it from 1 << N.
	// 1111.0000 (True, 4, 4) represents -1
	// 1 << bits => 1 << 8 => 100000000
	// 1 0000 0000 - 1111 0000 = 0001 0000
	total := big.NewInt(1)
	total.Lsh(total, fp.Tp.Fraction+fp.Tp.Integer)
	fp.Underlying.Sub(total, &fp.Underlying)
}

func (fp *FixedPoint) SetInt(integer *big.Int) {
	if !fp.Tp.Signed && integer.Sign() < 0 {
		panic("Attempting to convert a negative integer to an unsigned FixedPoint")
	}
	if integer.Cmp(big.NewInt(0)) < 0 {
		v := new(big.Int)
		v.Abs(integer)
		fp.SetInt(v)
		fp.NegInPlace()
	} else {
		fp.Underlying.Lsh(integer, fp.Tp.Fraction)
	}
}

func (fp *FixedPoint) SetFloat(float *big.Float) {
	if !fp.Tp.Signed && float.Signbit() {
		panic("Attempting to convert a negative float to an unsigned FixedPoint")
	}
	if float.Cmp(big.NewFloat(0)) < 0 {
		neg := new(big.Float)
		neg.Neg(float)
		fp.SetFloat(neg)
		fp.NegInPlace()
		return
	}
	numShift := big.NewInt(1)
	numShift.Lsh(numShift, fp.Tp.Fraction)
	intermediate := new(big.Float)
	intermediate.SetInt(numShift)
	intermediate.Mul(intermediate, float)
	intermediate.Int(&fp.Underlying)
}

func (fp FixedPoint) ToRat() *big.Rat {
	result := new(big.Rat)
	denom := big.NewInt(1)
	denom.Lsh(denom, fp.Tp.Fraction)
	if fp.Signbit() {
		cpy := fp.Copy()
		cpy.NegInPlace()
		result.SetFrac(&cpy.Underlying, denom)
		result.Neg(result)
	} else {
		result.SetFrac(&fp.Underlying, denom)
	}
	return result
}

func (fp FixedPoint) ToFloat() *big.Float {
	result := new(big.Float)
	return result.SetRat(fp.ToRat())
}

func (fp FixedPoint) ToInt() *big.Int {
	if fp.Signbit() {
		cpy := fp.Copy()
		cpy.NegInPlace()
		result := new(big.Int)
		result.Rsh(&cpy.Underlying, fp.Tp.Fraction)
		result.Neg(result)
		return result
	} else {
		result := new(big.Int)
		result.Rsh(&fp.Underlying, fp.Tp.Fraction)
		return result
	}
}

func (fp FixedPoint) String() string {
	// Input: 111.11111 in Q3.5
	// Plus 1 for decimal point
	result := make([]string, fp.Tp.NBits()+1)
	var writePtr int = 0
	for bitIndex := int(fp.Tp.NBits()) - 1; bitIndex >= 0; bitIndex-- {
		result[writePtr] = strconv.FormatUint(uint64(fp.Underlying.Bit(bitIndex)), 2)
		writePtr++
		// If we're crossing the decimal boundary, push in a decimal
		if bitIndex == int(fp.Tp.Fraction) {
			result[writePtr] = "."
			writePtr++
		}
	}

	return "0b" + strings.Join(result, "")
}

func Cmp(a, b FixedPoint) int {
	return a.ToRat().Cmp(b.ToRat())
}

func checkFormats(seq ...FixedPoint) {
	if len(seq) == 0 {
		return
	}
	refTp := seq[0].Tp
	for _, v := range seq {
		if refTp != v.Tp {
			panic(fmt.Sprintf("Fixed Point Type Mismatch: %s %s", refTp, v.Tp))
		}
	}
}

func (a *FixedPoint) Signbit() bool {
	if !a.Tp.Signed {
		return false
	}
	return a.Underlying.Bit(int(a.Tp.NBits())-1) > 0
}

func FixedAdd(a FixedPoint, b FixedPoint) FixedPoint {
	checkFormats(a, b)
	result := FixedPoint{Tp: a.Tp}
	result.Underlying.Add(&a.Underlying, &b.Underlying)
	// If we overflowed, zero out the next bit
	result.Underlying.SetBit(&result.Underlying, int(a.Tp.NBits()), 0)
	return result
}

func (a *FixedPoint) Copy() FixedPoint {
	result := FixedPoint{Tp: a.Tp}
	result.Underlying.Set(&a.Underlying)
	return result
}

func FixedMulFull(a, b FixedPoint) FixedPoint {
	// Make sure that A and B are both positive
	aSign := a.Signbit()
	aCpy := a.Copy()
	if aSign {
		aCpy.NegInPlace()
	}
	bSign := b.Signbit()
	bCpy := b.Copy()
	if bSign {
		bCpy.NegInPlace()
	}
	fmt.Printf("Multiplying: %s %s\n", aCpy.String(), bCpy.String())
	nInt := a.Tp.Integer + b.Tp.Integer
	if a.Tp.Signed && b.Tp.Signed {
		nInt -= 1
	}
	result := FixedPoint{Tp: FixedPointType{
		Signed:   a.Tp.Signed || b.Tp.Signed,
		Integer:  nInt,
		Fraction: a.Tp.Fraction + b.Tp.Fraction,
	}}
	result.Underlying.Mul(&aCpy.Underlying, &bCpy.Underlying)
	fmt.Printf("Result: %s\n", result.String())
	if aSign != bSign {
		result.NegInPlace()
	}
	return result
}

func (fix *FixedPoint) FixedToFixed(newTp FixedPointType) FixedPoint {
	result := FixedPoint{
		Tp: newTp,
	}

	// First, take the absolute value
	cpy := fix.Copy()
	if fix.Signbit() {
		cpy.NegInPlace()
	}
	// If new frac has more bits, then we need to left shift
	if fix.Tp.Fraction < newTp.Fraction {
		cpy.Underlying.Lsh(&cpy.Underlying, newTp.Fraction-fix.Tp.Fraction)
	}

	if fix.Tp.Fraction > newTp.Fraction {
		cpy.Underlying.Rsh(&cpy.Underlying, fix.Tp.Fraction-newTp.Fraction)
	}

	bitMask := big.NewInt(1)
	bitMask.Rsh(bitMask, newTp.Fraction+newTp.Integer)
	bitMask.Sub(bitMask, big.NewInt(1))
	cpy.Underlying.And(&cpy.Underlying, bitMask)
	result.Underlying.Set(&cpy.Underlying)
	if fix.Signbit() {
		result.NegInPlace()
	}

	return result
}
