package datatypes

import "math/big"

type AbstractValue struct {
	FakeSize *big.Int
}

func (abstract AbstractValue) Size() *big.Int { return abstract.FakeSize }
func (abstract AbstractValue) Validate() bool {
	return abstract.FakeSize.Cmp(big.NewInt(0)) >= 0
}
func (abstract AbstractValue) Payload() any { return nil }
