package main

import (
  "fmt"
  "math/big"
  "github.com/stanford-ppl/DAM/datatypes"
)

func main() {
	dtype := datatypes.FixPointType{true, 10, 22}
  value := datatypes.FixedPoint{dtype, new(big.Int)}
  value.SetFloat(big.NewFloat(1.125))
	fmt.Println(value.Tp)
  fmt.Println(value.ToFloat())
}