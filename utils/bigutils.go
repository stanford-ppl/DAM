package utils

type ComparableAndSetable[T any] interface {
	Cmp(T) int
	Set(T) T
}

// We want big.Int to be ComparableAndSetable[*big.Int]
// We can't define new methods on non-local (i.e. imported) types, so we'll have to make do with this.
func Max[T ComparableAndSetable[T]](x, y, recv ComparableAndSetable[T]) ComparableAndSetable[T] {
	if x.Cmp(y.(T)) >= 0 {
		recv.Set(x.(T))
	} else {
		recv.Set(y.(T))
	}
	return recv
}

func Min[T ComparableAndSetable[T]](x, y, recv ComparableAndSetable[T]) ComparableAndSetable[T] {
	if x.Cmp(y.(T)) <= 0 {
		recv.Set(x.(T))
	} else {
		recv.Set(y.(T))
	}
	return recv
}
