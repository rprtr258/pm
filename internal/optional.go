package internal

import "os"

type Optional[T any] struct {
	Value T
	Valid bool
}

func Valid[T any](value T) Optional[T] {
	return Optional[T]{
		Value: value,
		Valid: true,
	}
}

func Invalid[T any]() Optional[T] {
	return Optional[T]{
		Valid: false,
	}
}

func (opt Optional[T]) Ptr() *T {
	if !opt.Valid {
		return nil
	}

	return &opt.Value
}

// OrDefault - return first valid value, default if none found
func OrDefault[T any](defaultValue T, optionals ...Optional[T]) T {
	for _, optional := range optionals {
		if optional.Valid {
			return optional.Value
		}
	}
	return defaultValue
}

func Or[T any](optional Optional[T], or ...Optional[T]) Optional[T] {
	if optional.Valid {
		return optional
	}
	for _, ori := range or {
		if ori.Valid {
			return ori
		}
	}
	return Invalid[T]()
}

func Map[T, R any](optional Optional[T], mapper func(T) R) Optional[R] {
	if !optional.Valid {
		return Invalid[R]()
	}
	return Valid(mapper(optional.Value))
}

func GetEnv(varName string) Optional[string] {
	value, valid := os.LookupEnv(varName)
	return Optional[string]{
		Value: value,
		Valid: valid,
	}
}
