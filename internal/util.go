package internal

import "github.com/samber/lo"

func MapDict[T comparable, R any](collection []T, dict map[T]R) []R {
	result := make([]R, len(collection))

	for i, item := range collection {
		result[i] = dict[item]
	}

	return result
}

func IfNotNil[T, R any](ptr *T, mapper func(T) R) R {
	if ptr == nil {
		return lo.Empty[R]()
	}
	return mapper(*ptr)
}

func FilterMapToSlice[K comparable, V, R any](in map[K]V, mapper func(key K, value V) (R, bool)) []R {
	result := make([]R, 0, len(in))

	for k, v := range in {
		y, ok := mapper(k, v)
		if !ok {
			continue
		}
		result = append(result, y)
	}

	return result
}
