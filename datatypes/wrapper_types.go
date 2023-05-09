package datatypes

import "math/big"

type Bit struct {
	Value bool
}

func (b Bit) Payload() any   { return b }
func (b Bit) Size() *big.Int { return big.NewInt(1) }
func (b Bit) Validate() bool { return true }

var _ DAMType = (*Bit)(nil)
