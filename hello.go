package main

import (
	"fmt"
	"math/big"

	datatypes "github.com/stanford-ppl/DAM/datatypes/fixed"
)

func main() {
	dtype := datatypes.FixPointType{true, 10, 22}
	value := new(datatypes.FixedPoint)
	value.Tp = dtype
	value.SetFloat(big.NewFloat(1.125))
	fmt.Println(value.Tp)
	fmt.Println(value.ToFloat())
	fmt.Println(dtype.Min().ToFloat())
}
