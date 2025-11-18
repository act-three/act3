package expr

import "iter"

func Empty[E any]() iter.Seq[E] {
	return func(yield func(E) bool) {}
}

func Range[S, T any](s iter.Seq[S], f func(S) T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s {
			if !yield(f(v)) {
				return
			}
		}
	}
}

func Range2[S0, S1, T any](s iter.Seq2[S0, S1], f func(S0, S1) T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for s0, s1 := range s {
			if !yield(f(s0, s1)) {
				return
			}
		}
	}
}

func IfElse[T any](condition bool, consequent, alternative func() T) T {
	if condition {
		return consequent()
	} else {
		return alternative()
	}
}
