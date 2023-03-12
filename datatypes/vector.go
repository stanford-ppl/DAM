package datatypes

import (
	"fmt"
	"math/big"
)

type Vector[T datatypes.DAMType] struct {
	width int
	data []T
}

func (Vector* v[T datatypes.DAMType]) newVector(vecWidth int) {

	v.width = vecWidth
	v.data = make([]T , vecWidth)

}

func (Vector* v[T datatypes.DAMType]) width() int {
	return v.width
}

func (Vector* v[T datatypes.DAMType]) at(index int) T {
	return v.data[index]
}

func (Vector* v) Validate() bool {
	return len(v.data) == width
}