package sidebar

import "ily.dev/domi"

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
