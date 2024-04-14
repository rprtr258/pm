package lo

import "github.com/rprtr258/fun"

// Flatten returns array single level deep.
func Flatten[T any](collection ...[]T) []T {
	total := 0
	for _, coll := range collection {
		total += len(coll)
	}

	res := make([]T, 0, total)
	for _, coll := range collection {
		res = append(res, coll...)
	}
	return res
}

// Some returns true if at least 1 element of subset is contained in collection.
// If subset is empty, returns false.
func Some[T comparable](subset []T, collection ...T) bool {
	for _, elem := range subset {
		if fun.Contains(elem, collection...) {
			return true
		}
	}
	return false
}

func MapValues[K comparable, V, R any](in map[K]V, f func(K, V) R) map[K]R {
	result := make(map[K]R, len(in))
	for k, v := range in {
		result[k] = f(k, v)
	}
	return result
}
