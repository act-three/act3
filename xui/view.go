package ui

import (
	"cmp"

	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/xui/internal/sheet"
)

// A View is a user interface element,
// with modifiers to change its appearance and behavior.
//
// Views are immutable.
// A modifier always returns a new view
// that is a modified version of the underlying view.
//
// You can create custom views by writing functions that return View values.
//
//	func MyView() View {
//	    return Text("hello, world!")
//	}
//
// Compose the View by combining one or more of the built-in views
// provided in this package,
// like the Text view above,
// plus other custom view functions you define.
type View interface {
	// Background fills the background of the receiver.
	Background(Color) View

	// BorderShape sets the shape of the receiver's border.
	BorderShape(Shape) View

	// BorderStroke draws a line of the given width and color over the
	// inside edge of the receiver.
	BorderStroke(px float64, c Color) View

	// Disabled makes the receiver non-interactive. Controls such as
	// buttons, links, and text fields do not respond to input events.
	//
	// If any enclosing view sets Disabled(true), the view is disabled,
	// even when an inner view sets Disabled(false).
	// In this example, the button is disabled:
	//
	//     HStack(
	//         Button("/", Text("Home")).
	//             Disabled(false),
	//     ).
	//         Disabled(true)
	Disabled(d bool) View

	// FixedSize fixes the receiver at its ideal size.
	// This can cause it to exceed the bounds of its container.
	FixedSize() View

	// Font sets the font size for text in the receiver.
	Font(FontSize) View

	// Foreground uses c to draw foreground elements in the receiver,
	// such as text.
	Foreground(c Color) View

	// Frame positions the receiver inside an invisible frame with the
	// given dimensions and alignment.
	//
	// Note that type Alignment satisfies FrameOption.
	Frame(...FrameOption) View

	// FrameBounds positions the receiver inside an invisible frame with
	// the given bounds and alignment.
	//
	// Note that type Alignment satisfies FrameBoundsOption.
	FrameBounds(...FrameBoundsOption) View

	// FrameRatio positions the receiver inside an invisible frame
	// with a width:height aspect ratio of w:h.
	//
	// The anchor must be Horizontal or Vertical.
	// On the anchor axis, the frame adopts the size of the receiver.
	// On the other axis, its size is derived by the ratio.
	//
	// To display a 16:9 thumbnail, as wide as its available space:
	//
	//	Image(url).
	//		FramedAs(ScaledToFill).
	//		FrameRatio(16, 9, Horizontal)
	//
	// Note that type Alignment satisfies FrameRatioOption.
	FrameRatio(w, h int, anchor AxisSet, o ...FrameRatioOption) View

	// Overlay displays o as a layer on top of the receiver. Opaque
	// regions of o obscure the receiver where they overlap.
	//
	// The given Alignment sets the position of o relative to the receiver.
	//
	// Overlay(a, o) is equivalent to OverlayAt(a, a, o).
	Overlay(a Alignment, o View) View

	// OverlayAt displays o as a layer on top of the receiver.
	// It places the anchor point on o at the point on the
	// receiver specified by at.
	//
	// To place a badge centered at the top trailing corner of a view:
	//
	//     view.OverlayAt(TopTrailing, Center, badge)
	//
	// To place a menu below a view on the leading edge:
	//
	//     view.OverlayAt(BottomLeading, TopLeading, menu)
	//
	// To place a popover centered below a view:
	//
	//     view.OverlayAt(Bottom, Top, popover)
	//
	// When at and anchor are different, neither point can include
	// FirstBaseline. If either one does, the view panics.
	OverlayAt(at, anchor Alignment, o View) View

	// Underlay displays u as a layer beneath the receiver. Opaque
	// regions of the receiver obscure u where they overlap.
	//
	// The given Alignment sets the position of u relative to the receiver.
	//
	// Underlay(a, u) is equivalent to UnderlayAt(a, a, u).
	Underlay(a Alignment, u View) View

	// UnderlayAt displays u as a layer beneath the receiver.
	// It places the anchor point on u at the point on the
	// receiver specified by at.
	//
	// When at and anchor are different, neither point can include
	// FirstBaseline. If either one does, the view panics.
	UnderlayAt(at, anchor Alignment, u View) View

	// LineLimit limits the number of lines text can occupy
	// in the receiver.
	LineLimit(n int) View

	// Opacity sets the receiver's opacity to x, from 0 (transparent) to
	// 1 (opaque).
	Opacity(x float64) View

	// Padding adds the empty space defined by s around the receiver. If
	// more than one value s is provided, they are added together.
	Padding(s ...EdgeSpace) View

	// WhileHovered applies m to the receiver while in the Hovered state.
	//
	// It is equivalent to Modify(m, Hovered).
	WhileHovered(m Modifier) View

	// WhileFocused applies m to the receiver while in the Focused state.
	//
	// It is equivalent to Modify(m, Focused).
	WhileFocused(m Modifier) View

	// WhilePressed applies m to the receiver while in the Pressed state.
	//
	// It is equivalent to Modify(m, Pressed).
	WhilePressed(m Modifier) View

	// WhileDisabled applies m to the receiver while in the Disabled state.
	//
	// It is equivalent to Modify(m, Disabled).
	WhileDisabled(m Modifier) View

	// WhileChecked applies m to the receiver while in the Checked state.
	//
	// It is equivalent to Modify(m, Checked).
	WhileChecked(m Modifier) View

	// WhileInvalid applies m to the receiver while in the Invalid state.
	//
	// It is equivalent to Modify(m, Invalid).
	WhileInvalid(m Modifier) View

	// WhilePlaceholder applies m to the receiver while in the
	// Placeholder state.
	//
	// It is equivalent to Modify(m, Placeholder).
	WhilePlaceholder(m Modifier) View

	// Modify applies a modifier to the receiver. It applies only while
	// all given states are active. If no states are given, it applies
	// unconditionally.
	Modify(Modifier, ...State) View

	// Attr adds the given HTML attributes to the outermost HTML element
	// generated by the receiver.
	Attr(...domi.Attr) View

	// Class adds the given CSS classes to the outermost HTML element
	// generated by the receiver.
	Class(...string) View

	// Tag sets the HTML tag name of the outermost HTML element
	// generated by the receiver.
	Tag(string) View

	// modify applies m to the receiver.
	// It is the unexported equivalent of Modify,
	// accepting any internal modifier.
	modify(m modifier) View

	nodes() []node
}

// A node is a unary view.
// Its whole job is to lower itself to a single box.
type node interface {
	render(environment) box
}

type containerKind int

const (
	containerGrid containerKind = iota
	containerFlex
	// containerGridRotated is a grid whose writing mode is
	// rotated a quarter turn, so its inline axis is vertical.
	containerGridRotated
)

// environment carries the top-down state of a lowering pass.
type environment struct {
	lc        layoutContext
	container containerKind
	unbounded AxisSet
	disabled  bool
	lineLimit int // max lines per text, or 0 for no limit
	sheet     *sheet.Sheet
	boxenv    // must be zeroed before rendering a subview
}

// boxenv contains environment values
// that must be cleared before rendering a subview.
// They are "one-shot" or "one-box" values.
type boxenv struct {
	tag     string
	attrs   domi.Attr
	style   sheet.StyleSet
	fg      []term[color]
	bg      []term[color]
	stroke  []term[stroke]
	shape   []term[Shape]
	font    []term[FontSize]
	opacity []term[float64]

	text textStyle

	// fillMask is the set of axes to be stripped
	// from the box's fill request.
	// It is set at the outermost box of an unbounded subtree.
	fillMask AxisSet
	hasPaint bool // set by every paint modifier
}

// A stroke is one pending border line.
type stroke struct {
	px float64
	c  color
}

// add prepends attributes to the environment,
// keeping an inner writer's attributes before an outer's,
// so they land on the box in application order.
func (env *environment) add(a ...domi.Attr) {
	env.attrs = domi.Group(domi.Group(a...), env.attrs)
}

// A plan is an HTML element under construction.
type plan struct {
	content domi.Node
	fills   AxisSet // A fill request is the physical axes a box wants to fill.
	rigid   AxisSet
	ideal   rect // ideal is a box's size when available space is unbounded.
}

// A box is a rendered node.
// It contains the HTML node,
// plus ancillary data needed by its consumer.
type box struct {
	node  domi.Node
	fills AxisSet
	rigid AxisSet
}

// subviewsRendered is a generic combinator for lists of subviews.
// It renders the given views and merges their fill requests.
// It strips env's box values before any subview renders,
// so they cannot land on a subview's box.
func subviewsRendered(env environment, vs ...View) (domi.Node, AxisSet) {
	env.boxenv = boxenv{}
	var ns []domi.Node
	var f AxisSet
	for _, v := range vs {
		for _, n := range v.nodes() {
			b := n.render(env)
			f |= b.fills
			ns = append(ns, b.node)
		}
	}
	return domi.Fragment(ns...), f
}

// build returns the box described by env and p.
func build(env environment, p plan) box {
	a := env.attrs
	fills := p.fills &^ env.fillMask
	// A box is always rigid on an unbounded axis.
	rigid := p.rigid | env.unbounded
	ss := env.style
	addPaintStylesTo(&ss, env.boxenv)
	// Text styling is written closest to the view, so it beats the
	// paint modifiers.
	// BUG: a state-scoped paint modifier overrides text styling while
	// its states are active; it should not.
	env.text.setStyles(&ss)
	addIdealStylesTo(&ss, p.ideal, env.unbounded, fills)
	fills.addFillStylesTo(&ss, env)
	rigid.addRigidStylesTo(&ss, env)
	// Keep the generated class after the named classes in rendered output.
	a = domi.Group(a, attr.Class(env.sheet.ClassFor(ss)))
	return box{
		node:  domi.Tag(cmp.Or(env.tag, "div"), a)(p.content),
		fills: fills,
		rigid: rigid,
	}
}

// addIdealStylesTo adds CSS declarations for a box's ideal size.
// An ideal size applies only on an axis with unbounded available space.
// A box with no fill request simply uses its ideal size.
// A box with a fill request can expand beyond its ideal size.
// It contributes the ideal as a minimum length
// to the enclosing container's resolved extent,
// then expands to that extent.
func addIdealStylesTo(ss *sheet.StyleSet, i rect, unbounded, fills AxisSet) {
	for _, a := range [...]struct {
		axis AxisSet
		name string
		size size
	}{
		{Horizontal, "width", i.width},
		{Vertical, "height", i.height},
	} {
		if !a.size.definite || !unbounded.hasAll(a.axis) {
			continue
		}
		if fills.hasAll(a.axis) {
			ss.Set("min-"+a.name, a.size.css())
		} else {
			ss.Set(a.name, a.size.css())
		}
	}
}
