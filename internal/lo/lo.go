package lo

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
