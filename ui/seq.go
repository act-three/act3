package ui

import "ily.dev/domi"

// rangeNodes maps f over slice s and renders the results in order.
func rangeNodes[S ~[]E, E any](s S, f func(E) domi.Node) domi.Node {
	nodes := make([]domi.Node, len(s))
	for i, e := range s {
		nodes[i] = f(e)
	}
	return domi.Fragment(nodes...)
}
