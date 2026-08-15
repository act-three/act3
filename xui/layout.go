package ui

import (
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// AxisSet is a set of the two-dimensional cartesian axes:
// Horizontal, Vertical, or Horizontal|Vertical.
// The zero value is the empty set.
type AxisSet int

const (
	Horizontal AxisSet = 1 << iota
	Vertical
)

// hasAll returns whether s contains every axis in x.
func (s AxisSet) hasAll(x AxisSet) bool { return s&x == x }

// hasAny returns whether s contains at least one axis in x.
func (s AxisSet) hasAny(x AxisSet) bool { return s&x != 0 }

func (s AxisSet) complement() AxisSet { return (Horizontal | Vertical) &^ s }

// fillAttr lowers the receiver as a fill request, picking the CSS
// mechanism the parent context responds to: grow or stretch as a flex item,
// self-stretch in a grid cell. A single mechanism cannot serve both.
func (s AxisSet) fillAttr(env environment) (a domi.Attr) {
	if s == 0 {
		return nil
	}
	switch env.container {
	case containerFlex:
		if s.hasAny(env.lc.majorAxis) {
			a = domi.Group(a, attr.Class("ui-grow"))
		}
		if s.hasAny(env.lc.minorAxes()) {
			a = domi.Group(a, attr.Class("ui-stretch"))
		}
	default:
		if s.hasAll(Horizontal) {
			a = domi.Group(a, attr.Class("ui-cell-fill-x"))
		}
		if s.hasAll(Vertical) {
			a = domi.Group(a, attr.Class("ui-cell-fill-y"))
		}
	}
	return a
}

// rigidAttr lowers a box's rigid axes, such as the axes of a fixed-size
// frame, to avoid CSS default shrink behavior.
// This is only necessary along a flex major axis,
// where flex shrink would compress the box.
// Everywhere else a fixed size is rigid by default in CSS.
func (s AxisSet) rigidAttr(env environment) domi.Attr {
	if env.container != containerFlex {
		return nil
	}
	if s.hasAll(env.lc.majorAxis) {
		return attr.Class("ui-rigid")
	}
	return nil
}

// stackAxis is the major-axis direction a stack establishes for its subviews.
type stackAxis int

const (
	axisZ stackAxis = iota
	axisH
	axisV
)

// axes is each stack axis's whole lowering personality:
// its CSS class, the kind of container its subviews sit in,
// and the layout context it establishes for them.
var axes = [...]struct {
	class     string
	container containerKind
	lc        layoutContext
}{
	axisZ: {"ui-zstack", containerGrid, layoutContext{}},
	axisH: {"ui-hstack", containerFlex, layoutContext{majorAxis: Horizontal}},
	axisV: {"ui-vstack", containerFlex, layoutContext{majorAxis: Vertical}},
}

type layoutContext struct {
	majorAxis AxisSet // can be 0 (no axis), eg for zstack
}

// minorAxes is the complement of majorAxis.
// It's plural because both axes can be minor, eg for zstack.
func (lc layoutContext) minorAxes() AxisSet { return lc.majorAxis.complement() }

// An Alignment specifies a point on a view's bounding rectangle.
// For two views to be aligned,
// they are placed so their alignment points coincide.
//
// Alignment also satisfies [FrameOption] and [FrameBoundsOption].
//
// The default alignment is Center.
type Alignment int

var (
	_ FrameOption       = Alignment(0)
	_ FrameBoundsOption = Alignment(0)
)

const (
	Center        Alignment = 0
	Leading       Alignment = 1 << 0
	Trailing      Alignment = 1 << 1
	Top           Alignment = 1 << 2
	Bottom        Alignment = 1 << 3
	FirstBaseline Alignment = 1 << 4

	TopLeading     Alignment = Top | Leading
	TopTrailing    Alignment = Top | Trailing
	BottomLeading  Alignment = Bottom | Leading
	BottomTrailing Alignment = Bottom | Trailing

	FirstBaselineLeading  Alignment = FirstBaseline | Leading
	FirstBaselineTrailing Alignment = FirstBaseline | Trailing
)

func (a Alignment) applyFrame(w *wrapFrame) { w.align = a }

func (a Alignment) applyFrameBounds(w *wrapFrameBounds) { w.align = a }

// keyword maps a single-axis projection to its CSS alignment keyword.
// It panics on an alignment with more than one bit set in an axis,
// such as Top|Bottom.
func (a Alignment) keyword() string {
	switch a {
	case Center:
		return "center"
	case Leading, Top:
		return "start"
	case Trailing, Bottom:
		return "end"
	case FirstBaseline:
		return "baseline"
	}
	panic(fmt.Sprintf("ui: invalid Alignment %#b", int(a)))
}

// horizontal returns the alignment's horizontal component.
func (a Alignment) horizontal() Alignment { return a & (Leading | Trailing) }

// vertical returns the alignment's vertical component.
func (a Alignment) vertical() Alignment { return a & (Top | Bottom | FirstBaseline) }

// placeItems maps an alignment to its place-items value,
// which takes the block component first,
// which in this package is always the vertical axis.
func (a Alignment) placeItems() string {
	if a == Center {
		return "center"
	}
	return a.vertical().keyword() + " " + a.horizontal().keyword()
}
