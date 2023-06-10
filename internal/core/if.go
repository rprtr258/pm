package internal

type ccond[T any] struct {
	predicate bool
}

func If[T any](predicate bool) ccond[T] { //nolint:revive // don't export type
	return ccond[T]{predicate: predicate}
}

type thener[T any] struct {
	val func() T
	ccond[T]
}

func (cond ccond[T]) Then(value T) thener[T] {
	return thener[T]{
		ccond: cond,
		val:   func() T { return value },
	}
}

func (cond ccond[T]) ThenF(value func() T) thener[T] {
	return thener[T]{
		ccond: cond,
		val:   value,
	}
}

func (then thener[T]) Else(value T) T {
	if then.ccond.predicate {
		return then.val()
	}
	return value
}

func (then thener[T]) ElseF(value func() T) T {
	if then.ccond.predicate {
		return then.val()
	}
	return value()
}

type elseifer[T any] struct {
	thener    thener[T]
	predicate bool
}

func (then thener[T]) ElseIf(predicate bool) elseifer[T] {
	return elseifer[T]{
		predicate: predicate,
		thener:    then,
	}
}

func (els elseifer[T]) Then(value T) thener[T] {
	if els.thener.predicate {
		return els.thener
	}

	return thener[T]{
		val: func() T { return value },
		ccond: ccond[T]{
			predicate: els.predicate,
		},
	}
}

func (els elseifer[T]) ThenF(value func() T) thener[T] {
	if els.thener.predicate {
		return els.thener
	}

	return thener[T]{
		val: value,
		ccond: ccond[T]{
			predicate: els.predicate,
		},
	}
}
