package view

import (
	"iter"

	"ily.dev/domi"
)

// iff returns f() if cond is true, otherwise nil.
func iff(cond bool, f func() domi.Node) domi.Node {
	if cond {
		return f()
	}
	return nil
}

// rangeNodes maps f over slice s and renders the results in order.
func rangeNodes[S ~[]E, E any](s S, f func(E) domi.Node) domi.Node {
	nodes := make([]domi.Node, len(s))
	for i, e := range s {
		nodes[i] = f(e)
	}
	return domi.Fragment(nodes...)
}

// rangeSeq maps f over s and renders the results in order.
func rangeSeq[E any](s iter.Seq[E], f func(E) domi.Node) domi.Node {
	var nodes []domi.Node
	for e := range s {
		nodes = append(nodes, f(e))
	}
	return domi.Fragment(nodes...)
}

// rangeSeq2 maps f over s and renders the results in order.
func rangeSeq2[K, V any](s iter.Seq2[K, V], f func(K, V) domi.Node) domi.Node {
	var nodes []domi.Node
	for k, v := range s {
		nodes = append(nodes, f(k, v))
	}
	return domi.Fragment(nodes...)
}
