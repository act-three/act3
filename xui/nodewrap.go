package ui

import (
	"strconv"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/xui/internal/sheet"
)

// wrapSubview renders n as the content of a fresh wrapper plan.
// The subview sits in the wrapper's single-cell grid,
// and the wrapper forwards the subview's fill request and rigid axes,
// staying layout-transparent.
// A frame masks the forwarded axes its own geometry governs.
// wrapSubview strips env's box values before the subview renders,
// so they cannot land on the subview's box.
func wrapSubview(env environment, n node) plan {
	env.boxenv = boxenv{}
	env.container = containerGrid
	b := n.render(env)
	return plan{
		fills:   b.fills,
		rigid:   b.rigid,
		content: b.node,
	}
}

// wrapMod builds a pass-through wrapper box around n,
// consuming the environment's pending box values.
func wrapMod(env environment, n node) box {
	p := wrapSubview(env, n)
	env.add(attr.Class("ui-mod"))
	env.style.Set("place-items", Center.placeItems())
	return build(env, p)
}

// LayerUnder displays u under v.
// Opaque regions of v obscure u where they overlap.
//
// The given Alignment sets the position of u relative to v.
func (v base) LayerUnder(a Alignment, u View) View {
	return v.Modify(wrapLayer{view: u, over: false, alignment: a})
}

// LayerOver displays o over v.
// Opaque regions of o obscure v where they overlap.
//
// The given Alignment sets the position of o relative to v.
func (v base) LayerOver(a Alignment, o View) View {
	return v.Modify(wrapLayer{view: o, over: true, alignment: a})
}

// Padding adds the empty space defined by s around v.
// If more than one value s is provided, they are all added.
func (v base) Padding(s ...EdgeSpace) View {
	var sum EdgeSpace
	for _, s := range s {
		sum = sum.add(s)
	}
	return v.Modify(wrapPadding{space: sum})
}

// The z ladder of a box's composite: its layers,
// and the stroke ring painted over all of them.
// The static ui-underlay, ui-layer-base, and ui-overlay rules
// in ui.css use the same values.
const (
	zUnderlay = iota
	zLayerBase
	zOverlay
	zStroke
)

// wrapLayer layers a view over or under a base view.
// It lowers to CSS absolute positioning.
// The base negotiates its layout independently of the layer,
// and the layer receives available space defined by the layout's box.
type wrapLayer struct {
	view      View
	over      bool
	alignment Alignment // placement within the layer
	node      node
}

func (w wrapLayer) modify(n node) node { w.node = n; return w }

func (w wrapLayer) render(env environment) box {
	var bss, lss sheet.StyleSet
	classFor := func(ss sheet.StyleSet) domi.Attr {
		return attr.Class(env.sheet.ClassFor(ss))
	}
	class := "ui-underlay"
	if w.over {
		class = "ui-overlay"
	}
	bss.Set("place-items", Center.placeItems())
	lss.Set("place-items", w.alignment.placeItems())
	p := wrapSubview(env, w.node)
	p.content = domi.Fragment(
		html.Div(attr.Class("ui-layer-base"), classFor(bss))(p.content),
		html.Div(attr.Class(class), classFor(lss))(renderLayer(env, w.view)),
	)
	env.add(attr.Class("ui-layers"))
	// A pending stroke's ring must clear the layers' z ladder.
	// Elsewhere, its tree position suffices.
	if len(env.stroke) > 0 {
		env.style.SetPseudo("::after", "z-index", strconv.Itoa(zStroke))
	}
	return build(env, p)
}

// renderLayer renders a view inside its grid layer,
// where, as in a ZStack, both axes are minor.
// v's fill requests don't propagate outside the layer.
func renderLayer(env environment, v View) domi.Node {
	env = environment{lc: axes[axisZ].lc, container: containerGrid, sheet: env.sheet}
	content, _ := subviewsRendered(env, v)
	return content
}

// wrapPadding is a padded box.
// the subview keeps its own box, inset within the wrapper.
// Padding doesn't affect the subview's layout or appearance.
// It is not CSS padding on the subview itself.
type wrapPadding struct {
	space EdgeSpace
	node  node
}

func (w wrapPadding) modify(n node) node { w.node = n; return w }

func (w wrapPadding) render(env environment) box {
	p := wrapSubview(env, w.node)
	env.add(attr.Class("ui-padding"))
	env.style.Set("place-items", Center.placeItems())
	w.space.setPadding(&env.style)
	return build(env, p)
}
