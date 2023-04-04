package datatypes

import "math/big"

// TODO:  Does golang support another generic param to set length of array
// that way, we can embed a static-sized array in the struct itself without having
// to call `NewVector`
type Vector[T DAMType] struct {
	data []T
}

func (vector Vector[T]) Size() *big.Int {
	result := big.NewInt(0)
	for _, v := range vector.data {
		result.Add(result, v.Size())
	}
	return result
}

func (vector Vector[T]) Payload() any { return vector }

func NewVector[T DAMType](width int) Vector[T] {
	return Vector[T]{data: make([]T, width)}
}

func (v *Vector[T]) Width() int {
	return len(v.data)
}

// TODO:  Need to return optional error if index is out of bounds
func (v *Vector[T]) Set(index int, value T) {
	v.data[index] = value
}

func (v *Vector[T]) Get(index int) T {
	return v.data[index]
}

func (v Vector[T]) Validate() bool {
	for _, v := range v.data {
		if !v.Validate() {
			return false
		}
	}
	return true
}
