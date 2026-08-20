package ui

import (
	"cmp"
	"strconv"

	"ily.dev/domi"
	"ily.dev/domi/attr"

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
	env.tag = cmp.Or(env.tag, "ui-box")
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	env.style.Set("place-items", Center.placeItems())
	return build(env, p)
}

// wrapLayer layers a view over or under a base view.
// It lowers to CSS absolute positioning.
// The base negotiates its layout independently of the layer,
// and the layer receives available space defined by the layout's box.
type wrapLayer struct {
	view   View
	over   bool
	at     Alignment // point in the base view where the layer view is placed
	anchor Alignment // point in the layer view placed onto at
	node   node
}

func (w wrapLayer) modify(n node) node { w.node = n; return w }

func (w wrapLayer) render(env environment) box {
	const (
		zUnderlay = -1
		// base view has z-index auto
		zOverlay = 2
		zStroke  = 3
	)
	var lss sheet.StyleSet
	lss.Set("display", "grid")
	lss.Set("grid-template-columns", "100%")
	lss.Set("grid-template-rows", "100%")
	lss.Set("place-items", w.at.placeItems())
	lss.Set("position", "absolute")
	lss.Set("inset", "0")
	tag := "ui-underlay"
	view := w.view
	if w.anchor != w.at {
		// Placement puts the layer view's at-point onto the base's.
		// Shift by the difference of the two points, in the layer's
		// coordinates, so its anchor point lands there instead.
		x := w.at.horizontal().point() - w.anchor.horizontal().point()
		y := w.at.vertical().point() - w.anchor.vertical().point()
		view = view.Modify(modStyle{"translate", strconv.Itoa(x) + "% " + strconv.Itoa(y) + "%"})
	}
	if w.over {
		tag = "ui-overlay"
		lss.Set("z-index", strconv.Itoa(zOverlay))
		// The overlay box blankets the base; input falls through it
		// to the base, and only the layered subviews take hits.
		lss.Set("pointer-events", "none")
		view = view.Modify(modStyle{"pointer-events", "auto"})
	} else {
		lss.Set("z-index", strconv.Itoa(zUnderlay))
	}
	// Prevent high-z-index subviews
	// from painting on top of the overlay or border stroke.
	p := wrapSubview(env, modStyle{"isolation", "isolate"}.modify(w.node))
	p.content = domi.Fragment(
		p.content,
		domi.Tag(tag, attr.Class(env.sheet.ClassFor(lss)))(
			renderLayer(env, view),
		),
	)
	env.tag = cmp.Or(env.tag, "ui-layer")
	// The container hosts the base subview in its own single-cell
	// grid; the isolated z ladder sandwiches the in-flow subview
	// between the layers.
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	env.style.Set("place-items", Center.placeItems())
	env.style.Set("position", "relative")
	env.style.Set("isolation", "isolate")
	env.style.Set("overflow", "visible")
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
	env.tag = cmp.Or(env.tag, "ui-padding")
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	env.style.Set("place-items", Center.placeItems())
	w.space.setPadding(&env.style)
	return build(env, p)
}
