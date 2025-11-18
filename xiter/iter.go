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

// DropUntil calls f on each element of s until f returns true.
// It returns a sequence beginning at the element
// for which f returns true.
// It does not subsequently call f.
func DropUntil[E any](s iter.Seq[E], f func(E) bool) iter.Seq[E] {
	return func(yield func(E) bool) {
		ok := false
		for v := range s {
			if !(ok || f(v)) {
				continue
			}
			ok = true
			if !yield(v) {
				return
			}
		}
	}
}

// Keep yields the first n elements of s.
func Keep[E any](s iter.Seq[E], n int) iter.Seq[E] {
	return func(yield func(E) bool) {
		for v := range s {
			n--
			if !yield(v) || n == 0 {
				return
			}
		}
	}
}
