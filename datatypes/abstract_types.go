package datatypes

import "math/big"

type AbstractValue struct {
	size *big.Int
}

func (abstract AbstractValue) Size() *big.Int { return abstract.size }
func (abstract AbstractValue) Validate() bool {
	return abstract.size.Cmp(big.NewInt(0)) >= 0
}
func (abstract AbstractValue) Payload() any { return nil }
