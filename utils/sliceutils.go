package utils

func MinMax[T any](elts []T, cmp func(T, T) int) (min T, max T, minIndex int, maxIndex int) {
	min = elts[0]
	max = elts[0]
	minIndex = 0
	maxIndex = 0
	for i, v := range elts {
		if cmp(v, min) < 0 {
			minIndex = i
			min = v
		}
		if cmp(v, max) > 0 {
			maxIndex = i
			max = v
		}
	}
	return
}
