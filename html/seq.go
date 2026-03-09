package html

import (
	"io"
	"iter"
	"slices"

	"ily.dev/act3/expr"
)

type seq iter.Seq[Node]

// Seq returns a Node that renders each element of s in order,
// skipping nil elements.
func Seq(s iter.Seq[Node]) Node {
	return seq(func(yield func(Node) bool) {
		for n := range s {
			if n != nil && !yield(n) {
				return
			}
		}
	})
}

// Group renders its children, in order.
func Group(child ...Node) Node {
	return Seq(slices.Values(child))
}

// RangeSeq maps f over s and renders the results as a [Seq].
func RangeSeq[E any](s iter.Seq[E], f func(E) Node) Node {
	return Seq(expr.Range(s, f))
}

// RangeSeq2 maps f over s and renders the results as a [Seq].
func RangeSeq2[K, V any](s iter.Seq2[K, V], f func(K, V) Node) Node {
	return Seq(expr.Range2(s, f))
}

// Range maps f over slice s and renders the results as a [Seq].
func Range[S ~[]E, E any](s S, f func(E) Node) Node {
	return RangeSeq(slices.Values(s), f)
}

// If returns f() if cond is true, otherwise nil.
func If(cond bool, f func() Node) Node {
	if cond {
		return f()
	}
	return nil
}

// The only errors returned are from w (via child nodes).
func (s seq) renderTo(w io.Writer) error {
	for n := range s {
		err := n.renderTo(w)
		if err != nil {
			return err
		}
	}
	return nil
}
