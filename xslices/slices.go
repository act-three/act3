package xslices

import "iter"

// Group returns an iterator over consecutive sub-slices of s.
// It calls eq on each pair of adjacent elements.
// If eq returns true,
// the second element will be included in the same group as the first.
// Otherwise, the second element will begin a new group.
//
// All sub-slices are clipped to have no capacity beyond the length.
// If s is empty, the sequence is empty:
// there is no empty slice in the sequence.
func Group[S ~[]E, E any](s S, eq func(E, E) bool) iter.Seq[S] {
	return func(yield func(S) bool) {
		i := 0
		for i < len(s) {
			j := i + 1
			for j < len(s) && eq(s[j-1], s[j]) {
				j++
			}
			// Set the capacity of each group so that appending to a group does
			// not modify the original slice.
			if !yield(s[i:j:j]) {
				return
			}
			i = j
		}
	}
}

// GroupBy returns an iterator over consecutive sub-slices of s.
// It calls key on each element, yielding a key.
// If two adjacent keys compare equal,
// the second element will be included in the same group as the first.
// Otherwise, the second element will begin a new group.
//
// All sub-slices are clipped to have no capacity beyond the length.
// If s is empty, the sequence is empty:
// there is no empty slice in the sequence.
func GroupBy[K comparable, S ~[]E, E any](s S, key func(E) K) iter.Seq2[K, S] {
	return func(yield func(K, S) bool) {
		i := 0
		for i < len(s) {
			k := key(s[i])
			j := i + 1
			for j < len(s) && k == key(s[j]) {
				j++
			}
			// Set the capacity of each group so that appending to a group does
			// not modify the original slice.
			if !yield(k, s[i:j:j]) {
				return
			}
			i = j
		}
	}
}
