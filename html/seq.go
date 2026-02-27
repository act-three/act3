package html

import (
	"io"
	"iter"
	"slices"

	"ily.dev/act3/expr"
)

type seq iter.Seq[Node]

func Seq(s iter.Seq[Node]) Node {
	return seq(func(yield func(Node) bool) {
		for n := range s {
			if n != nil && !yield(n) {
				return
			}
		}
	})
}

func Group(child ...Node) Node {
	return Seq(slices.Values(child))
}

func RangeSeq[E any](s iter.Seq[E], f func(E) Node) Node {
	return Seq(expr.Range(s, f))
}

func RangeSeq2[K, V any](s iter.Seq2[K, V], f func(K, V) Node) Node {
	return Seq(expr.Range2(s, f))
}

func Range[S ~[]E, E any](s S, f func(E) Node) Node {
	return RangeSeq(slices.Values(s), f)
}

func If(cond bool, f func() Node) Node {
	if cond {
		return f()
	}
	return nil
}

func (s seq) renderTo(w io.Writer) error {
	for n := range s {
		err := n.renderTo(w)
		if err != nil {
			return err
		}
	}
	return nil
}
