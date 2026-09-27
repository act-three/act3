package hi

import (
	"cmp"
	"fmt"
	"reflect"
	"strconv"

	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/hi/internal/canon"
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
	env.tag = cmp.Or(env.tag, "hi-box")
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
		layer := w.renderFixedLayer(env)
		b.node = domi.Fragment(b.node, layer.content)
		b.title = cmp.Or(b.title, layer.title)
		return b
	}

	// Prevent high-z-index subviews
	// from painting on top of the overlay or border stroke.
	p := wrapSubview(env, modStyle("isolation", "isolate")(n))
	layer, layerTitle := w.renderLayerElement(env)
	p.content = domi.Fragment(p.content, layer)
	p.title = cmp.Or(p.title, layerTitle)
	env.tag = cmp.Or(env.tag, "hi-layer")
	// The container hosts the base subview in its own single-cell
	// grid; the isolated z ladder sandwiches the in-flow subview
	// between the layers.
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	Center.setItemsOn(&env.style)
	env.position.interior = true
	env.style.Set("isolation", "isolate")
	// Pending foreground paint must paint in front by z-index.
	// Elsewhere, its tree position suffices.
	if len(env.stroke) > 0 || len(env.outline) > 0 {
		env.style.SetPseudo("::after", "z-index", strconv.Itoa(zLayerStroke))
	}
	return build(env, p)
}

// renderLayerElement renders the overlay or underlay view in a covering grid.
func (w wrapLayer) renderLayerElement(env environment) (domi.Node, string) {
	var lss canon.StyleSet
	lss.Set("display", "grid")
	lss.Set("grid-template-columns", "100%")
	lss.Set("grid-template-rows", "100%")
	w.at.setItemsOn(&lss)
	EdgeSpace{}.setOn(&lss, "inset")
	tag := "hi-underlay"
	view := modEnv(func(env environment) environment {
		setAttachmentTranslation(&env.style, w.at, w.anchor)
		return env
	})(w.layer)
	if w.over {
		tag = "hi-overlay"
		lss.Set("z-index", strconv.Itoa(zOverlay))
		// The overlay box blankets the base; input falls through it
		// to the base, and only the layered subviews take hits.
		lss.Set("pointer-events", "none")
		view = modStyle("pointer-events", "auto")(view)
	} else {
		lss.Set("z-index", strconv.Itoa(zUnderlay))
	}
	layer := renderLayer(env, view)
	styles := lss.Decls()
	(position{exterior: positionAbsolute}).setOn(&styles)
	return domi.Tag(tag, attr.Class(env.sheet.ClassFor(styles)))(layer.content), layer.title
}

func (w wrapLayer) renderFixedLayer(env environment) plan {
	return renderLayer(env, modEnv(func(env environment) environment {
		env.position.exterior = positionFixed
		// Safari collapses percentage grid rows in auto-height fixed
		// boxes. Explicit content sizing keeps their children in bounds;
		// authored heights and vertical fill override this default.
		env.style.Set("height", "fit-content")
		// Insets define the alignment area, not the box's size. Fill
		// requests can override this self-alignment with stretch.
		// Unsafe alignment preserves the grid's overflow placement.
		// There is no baseline-sharing group at the viewport.
		a := w.at.withoutBaseline()
		env.style.Set("align-self", "unsafe "+a.vertical().keyword())
		env.style.Set("justify-self", "unsafe "+a.horizontal().keyword())
		EdgeSpace{}.setOn(&env.style, "inset")
		env.style.Set("z-index", fmt.Sprint(zOverlay))
		setAttachmentTranslation(&env.style, w.at, w.anchor)
		return env
	})(w.layer))
}

func setAttachmentTranslation(ss *canon.StyleSet, at, anchor Alignment) {
	if anchor == at {
		return
	}
	// Placement aligns the view's at-point; shift by the difference
	// in its own coordinates so the anchor point lands there instead.
	x := at.horizontal().point() - anchor.horizontal().point()
	y := at.vertical().point() - anchor.vertical().point()
	ss.Set("translate", fmt.Sprintf("%d%% %d%%", x, y))
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
	env.tag = cmp.Or(env.tag, "hi-padding")
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
	env.tag = cmp.Or(env.tag, "hi-sticky")
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	Center.setItemsOn(&env.style)
	// A container may already have placed this box out of flow.
	if env.position.exterior == positionFlow {
		env.position.exterior = positionSticky
		w.inset.setOn(&env.style, "inset")
		env.style.Set("z-index", "1") // The scroll viewport isolates the z-index.
	}
	return build(env, p)
}
