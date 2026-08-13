package ui

import "testing"

// countingNode wraps a node and counts its render calls, so the linearity
// test below can assert how often the machinery visits each node.
type countingNode struct {
	inner  node
	visits *int
}

func (c countingNode) render(env environment) box {
	*c.visits++
	return c.inner.render(env)
}

// TestRenderVisitsEachNodeOnce pins the linear-time contract: a render visits
// each node exactly once, with fill requests accumulating bottom-up on the
// boxes in the same pass. An accidental subtree re-walk — recomputing a
// subtree's fills at every level, re-rendering a subview per ancestor —
// multiplies these counts long before it would show up as wall-clock time on
// the small trees tests build.
func TestRenderVisitsEachNodeOnce(t *testing.T) {
	var counts []*int
	count := func(n node) node {
		visits := new(int)
		counts = append(counts, visits)
		return countingNode{inner: n, visits: visits}
	}
	wrap := func(n node) View { return base{count(n)} }
	leaf := func(s string) node { return count(textNode{text: s}) }

	// A deep chain interleaving the shapes whose lowering most tempts a
	// per-level re-walk: stacks resolving subview fills, definite frames
	// terminating them, and decoration layers.
	var v View = base{leaf("leaf")}
	for range 100 {
		v = wrap(stackNode{dir: axisV, subviews: []View{
			v,
			wrap(spacerNode{}),
			wrap(wrapFrame{h: newSize(40), node: leaf("framed")}),
			wrap(wrapLayer{view: wrap(colorFillNode{color: "#000"}), node: leaf("decorated")}),
		}})
	}
	new(Renderer).Render(v)

	for i, c := range counts {
		if *c != 1 {
			t.Fatalf("node %d: renders=%d, want 1", i, *c)
		}
	}
}
