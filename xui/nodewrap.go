package ui

import (
	"cmp"
	"reflect"
	"strconv"

	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/xui/internal/canon"
)

// wrapSubview renders n as the content of a fresh wrapper plan.
// The subview sits in the wrapper's single-cell grid,
// and the wrapper forwards the subview's fill request, rigid axes,
// and ancillary data,
// so it is layout-preserving initially.
// (A frame then masks the forwarded axes its own geometry governs.)
// wrapSubview strips env's box values before the subview renders,
// so they cannot land on the subview's box.
func wrapSubview(env environment, n node) plan {
	return wrapSubviewIn(env, containerGrid, n)
}

// wrapSubviewIn is wrapSubview for a wrapper of the given container kind.
func wrapSubviewIn(env environment, kind containerKind, n node) plan {
	env.container = kind
	return renderSubviewNode(env, n)
}

// wrapMod builds a pass-through wrapper box around n,
// consuming the environment's pending box values.
func wrapMod(env environment, n node) box {
	p := wrapSubview(env, n)
	env.tag = cmp.Or(env.tag, "ui-box")
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	Center.setItemsOn(&env.style)
	return build(env, p)
}

// wrapLayer layers a view over or under a base view.
// It lowers to CSS absolute positioning.
// The base negotiates its layout independently of the layer,
// and the layer receives available space defined by the layout's box.
type wrapLayer struct {
	layer  node // the overlay or underlay layer
	over   bool
	at     Alignment // point in the base view where the layer view is placed
	anchor Alignment // point in the layer view placed onto at
}

const (
	zUnderlay = -1
	// base view has z-index auto
	zOverlay     = 2
	zLayerStroke = 3
)

func (w wrapLayer) modify(n node) node {
	return func(env environment) box { return w.render(env, n) }
}

func (w wrapLayer) render(env environment, n node) box {
	if w.over && canOverlayRoot(env) {
		baseEnv := env
		// Using modStyle here would eg make canScrollDocument return false.
		baseEnv.root.style.Set("isolation", "isolate")
		b := n(baseEnv)
		layer, title := w.renderLayerElement(env, "fixed")
		b.node = domi.Fragment(b.node, layer)
		b.title = cmp.Or(b.title, title)
		return b
	}

	// Prevent high-z-index subviews
	// from painting on top of the overlay or border stroke.
	p := wrapSubview(env, modStyle("isolation", "isolate")(n))
	layer, layerTitle := w.renderLayerElement(env, "absolute")
	p.content = domi.Fragment(p.content, layer)
	p.title = cmp.Or(p.title, layerTitle)
	env.tag = cmp.Or(env.tag, "ui-layer")
	// The container hosts the base subview in its own single-cell
	// grid; the isolated z ladder sandwiches the in-flow subview
	// between the layers.
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	Center.setItemsOn(&env.style)
	env.style.Set("position", "relative")
	env.style.Set("isolation", "isolate")
	// A pending stroke's ring must clear the layers' z ladder.
	// Elsewhere, its tree position suffices.
	if len(env.stroke) > 0 {
		env.style.SetPseudo("::after", "z-index", strconv.Itoa(zLayerStroke))
	}
	return build(env, p)
}

// renderLayerElement renders the overlay or underlay view in a covering grid.
// position is absolute for an ordinary layer and fixed for a hoisted root
// overlay.
func (w wrapLayer) renderLayerElement(env environment, position string) (domi.Node, string) {
	var lss canon.StyleSet
	lss.Set("display", "grid")
	lss.Set("grid-template-columns", "100%")
	lss.Set("grid-template-rows", "100%")
	w.at.setItemsOn(&lss)
	lss.Set("position", position)
	EdgeSpace{}.setOn(&lss, "inset")
	tag := "ui-underlay"
	view := w.layer
	if w.anchor != w.at {
		// Placement puts the layer view's at-point onto the base's.
		// Shift by the difference of the two points, in the layer's
		// coordinates, so its anchor point lands there instead.
		x := w.at.horizontal().point() - w.anchor.horizontal().point()
		y := w.at.vertical().point() - w.anchor.vertical().point()
		view = modStyle("translate", strconv.Itoa(x)+"% "+strconv.Itoa(y)+"%")(view)
	}
	if w.over {
		tag = "ui-overlay"
		lss.Set("z-index", strconv.Itoa(zOverlay))
		// The overlay box blankets the base; input falls through it
		// to the base, and only the layered subviews take hits.
		lss.Set("pointer-events", "none")
		view = modStyle("pointer-events", "auto")(view)
	} else {
		lss.Set("z-index", strconv.Itoa(zUnderlay))
	}
	layer := renderLayer(env, view)
	return domi.Tag(tag, attr.Class(env.sheet.ClassFor(lss.Decls())))(layer.content), layer.title
}

// canOverlayRoot reports whether removing the ordinary layer wrapper
// preserves every pending box value. Root environment requirements pass
// through this lowering by contract; any ordinary one-shot value belongs to
// the composite box and keeps the ordinary lowering.
func canOverlayRoot(env environment) bool {
	if !env.root.atRoot {
		return false
	}
	return reflect.ValueOf(env.nextenv).IsZero()
}

// renderLayer renders a node inside its grid layer,
// where, as in a ZStack, both axes are minor.
// Its fill and rigid requests don't propagate outside the layer.
func renderLayer(env environment, n node) plan {
	env.lc = axes[axisZ].lc
	env.container = containerGrid
	env.unbounded = 0
	p := renderSubviewNode(env, n)
	p.fills = 0
	p.rigid = 0
	return p
}

// wrapPadding is a padded box.
// the subview keeps its own box, inset within the wrapper.
// Padding doesn't affect the subview's layout or appearance.
// It is not CSS padding on the subview itself.
type wrapPadding struct {
	space EdgeSpace
}

func (w wrapPadding) modify(n node) node {
	return func(env environment) box { return w.render(env, n) }
}

func (w wrapPadding) render(env environment, n node) box {
	p := wrapSubview(env, n)
	env.tag = cmp.Or(env.tag, "ui-padding")
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	Center.setItemsOn(&env.style)
	w.space.setOn(&env.style, "padding")
	return build(env, p)
}

// wrapSticky wraps it's subview's box in an element
// with CSS sticky positioning applied.
type wrapSticky struct {
	inset EdgeSpace
}

func (w wrapSticky) modify(n node) node {
	return func(env environment) box { return w.render(env, n) }
}

func (w wrapSticky) render(env environment, n node) box {
	p := wrapSubview(env, n)
	env.tag = cmp.Or(env.tag, "ui-sticky")
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	Center.setItemsOn(&env.style)
	env.style.Set("position", "sticky")
	w.inset.setOn(&env.style, "inset")
	env.style.Set("z-index", "1") // The scroll viewport isolates the z-index.
	return build(env, p)
}
