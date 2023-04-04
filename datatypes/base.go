package datatypes

import "math/big"

type DAMType interface {
	Validate() bool
	Size() *big.Int // In bits
	Payload() any
}

func IsConcrete(dt DAMType) bool {
	return dt.Payload() != nil
}
