package xiter

import "iter"

// Drop drops the first n elements of s and yields the rest.
func Drop[E any](s iter.Seq[E], n int) iter.Seq[E] {
	return func(yield func(E) bool) {
		for v := range s {
			n--
			if n >= 0 {
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
}
