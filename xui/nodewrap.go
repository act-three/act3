package ui

import (
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"
)

// An elementWrapper returns a fresh box containing the subview.
// An elementWrapper receives the subview's fill request f and rigid
// axes r. A layout-transparent wrapper forwards both to its own box;
// a frame answers for the axes its own geometry governs and forwards
// the rest.
type elementWrapper interface {
	wrapElement(rc renderContext, content domi.Node, f, r AxisSet) box
}

// nodeWrap applies wrapper to node's box.
type nodeWrap struct {
	node    node
	wrapper elementWrapper
}

func (m nodeWrap) render(rc renderContext) box {
	shape := rc.shapeClass()
	inner := rc
	inner.container = containerGrid
	if c, ok := m.wrapper.(contextual); ok {
		inner = c.context(inner)
	}
	x := m.node.render(inner)
	if x.raw != nil {
		panic(fmt.Sprintf("ui: %T applied to a Domi view, which has no ui-managed element", m.wrapper))
	}
	b := m.wrapper.wrapElement(rc, x.build(inner), x.fills, x.rigid)
	b.add(shape)
	return b
}

// wrap applies w to each of v's nodes.
func (v base) wrap(w elementWrapper) base {
	out := make(base, len(v))
	for i, n := range v {
		out[i] = nodeWrap{node: n, wrapper: w}
	}
	return out
}

// LayerUnder displays u under v.
// Opaque regions of v obscure u where they overlap.
//
// The given Alignment sets the position of u relative to v.
func (v base) LayerUnder(a Alignment, u View) View {
	return v.wrap(wrapLayer{view: u, over: false, alignment: a})
}

// LayerOver displays o over v.
// Opaque regions of o obscure v where they overlap.
//
// The given Alignment sets the position of o relative to v.
func (v base) LayerOver(a Alignment, o View) View {
	return v.wrap(wrapLayer{view: o, over: true, alignment: a})
}

// Padding adds the empty space defined by s around v.
// If more than one value s is provided, they are all added.
func (v base) Padding(s ...EdgeSpace) View {
	var sum EdgeSpace
	for _, s := range s {
		sum = sum.add(s)
	}
	return v.wrap(wrapPadding{space: sum})
}

// wrapLayer layers a view over or under a base view.
// It lowers to CSS absolute positioning.
// The base negotiates its layout independently of the layer,
// and the layer receives available space defined by the layout's box.
type wrapLayer struct {
	view      View
	over      bool
	alignment Alignment // placement within the layer
}

func (w wrapLayer) wrapElement(rc renderContext, content domi.Node, f, r AxisSet) box {
	class := "ui-underlay"
	if w.over {
		class = "ui-overlay"
	}
	attrs := []domi.Attr{attr.Class(class)}
	if w.alignment != Center {
		attrs = append(attrs, attr.Class(w.alignment.placeClass()))
	}
	baseLayer := html.Div(attr.Class("ui-layer-base"))(content)
	layer := html.Div(attrs...)(renderLayer(rc, w.view))
	return box{
		fills:   f,
		rigid:   r,
		attrs:   attr.Class("ui-layers"),
		content: domi.Fragment(baseLayer, layer),
	}
}

// renderLayer renders a view inside its grid layer,
// where, as in a ZStack, both axes are minor.
// v's fill requests don't propagate outside the layer.
func renderLayer(rc renderContext, v View) domi.Node {
	rc = renderContext{lc: axes[axisZ].lc, container: containerGrid, sheet: rc.sheet}
	content, _ := subviewsRendered(rc, v)
	return content
}

// wrapPadding is a padded box.
// the subview keeps its own box, inset within the wrapper.
// Padding doesn't affect the subview's layout or appearance.
// It is not CSS padding on the subview itself.
type wrapPadding struct {
	space EdgeSpace
}

func (w wrapPadding) wrapElement(_ renderContext, content domi.Node, f, r AxisSet) box {
	b := box{
		fills:   f,
		rigid:   r,
		attrs:   attr.Class("ui-padding"),
		content: content,
	}
	w.space.setPadding(&b)
	return b
}
