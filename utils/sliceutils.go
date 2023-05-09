package utils

func Map[T any, U any](input []T, mapFunc func(T) U) []U {
	result := make([]U, len(input))
	for i, v := range input {
		result[i] = mapFunc(v)
	}
	return result
}

func Foreach[T any](input []T, doFunc func(T)) {
	for _, v := range input {
		doFunc(v)
	}
}

func Filter[T any](input []T, test func(T) bool) (ret []T) {
	for _, v := range input {
		if test(v) {
			ret = append(ret, v)
		}
	}
	return
}

func FilterNot[T any](input []T, test func(T) bool) (ret []T) {
	return Filter(input, func(x T) bool {
		return !test(x)
	})
}

func Exists[T any](input []T, test func(T) bool) bool {
	for _, v := range input {
		if test(v) {
			return true
		}
	}
	return false
}

func Forall[T any](input []T, test func(T) bool) bool {
	for _, v := range input {
		if !test(v) {
			return false
		}
	}
	return true
}

func MinElem[T any](input []T, lt func(T, T) bool) (result T) {
	result = input[0]
	for _, v := range input[1:] {
		if lt(v, result) {
			result = v
		}
	}
	return
}

func Fill[T any](input []T, gen func() T) []T {
	for i := range input {
		input[i] = gen()
	}
	return input
}

func FillConst[T any](input []T, fill T) []T {
	for i := range input {
		input[i] = fill
	}
	return input
}

func Tabulate[T any](input []T, gen func(int) T) []T {
	for i := range input {
		input[i] = gen(i)
	}
	return input
}

func IsEmpty[T any](input []T) bool {
	return len(input) == 0
}
