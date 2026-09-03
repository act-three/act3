package ui

import "testing"

// TestRenderVisitsEachNodeOnce pins the linear-time contract: a render visits
// each node exactly once, with fill requests accumulating bottom-up on the
// boxes in the same pass. An accidental subtree re-walk — recomputing a
// subtree's fills at every level, re-rendering a subview per ancestor —
// multiplies these counts long before it would show up as wall-clock time on
// the small trees tests build.
func TestRenderVisitsEachNodeOnce(t *testing.T) {
	var counts []*int
	// count wraps a node and counts its render calls.
	count := func(n node) node {
		visits := new(int)
		counts = append(counts, visits)
		return func(env environment) box {
			*visits++
			return n(env)
		}
	}
	wrap := func(n node) View { return base{count(n)} }
	leaf := func(s string) node { return count(textLeaf(s).render) }

	// A deep chain interleaving the shapes whose lowering most tempts a
	// per-level re-walk: stacks resolving subview fills, definite frames
	// terminating them, and decoration layers.
	var v View = base{leaf("leaf")}
	for range 100 {
		v = wrap(nodeStack(axisV, []View{
			v,
			wrap(nodeSpacer),
			wrap(wrapFrame{h: newSize(40)}.modify(leaf("framed"))),
			wrap(wrapLayer{layer: count(nodeColor(cssColor("#000")))}.modify(leaf("decorated"))),
		}))
	}
	Render(v)

	for i, c := range counts {
		if *c != 1 {
			t.Fatalf("node %d: renders=%d, want 1", i, *c)
		}
	}
}
