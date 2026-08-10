package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// A StackView displays its subviews in a line.
type StackView struct{ base }

// A VStack arranges its subviews in a vertical line.
func VStack(v ...View) StackView {
	return StackView{base{stackNode{axisV, v, defaultGap, Center}}}
}

// An HStack arranges its subviews in a horizontal line.
func HStack(v ...View) StackView {
	return StackView{base{stackNode{axisH, v, defaultGap, Center}}}
}

// Alignment aligns the subviews in v.
// They are placed along v's minor axis
// so their alignment points are colinear.
//
// The default alignment is [Center].
func (v StackView) Alignment(a Alignment) StackView {
	n := v.base[0].(stackNode)
	n.align = a
	v.base = base{n}
	return v
}

// Gap sets the distance between adjacent subviews in v.
// The default gap is 8 px.
func (v StackView) Gap[L Length](px L) StackView {
	n := v.base[0].(stackNode)
	n.gap = float64(px)
	v.base = base{n}
	return v
}

// A ZStackView overlays its subviews, aligning them in both axes.
//
// It assigns each successive subview a higher z-axis value
// than the one before it,
// so later subviews obscure earlier ones where they overlap.
type ZStackView struct{ base }

// ZStack overlays the given views.
func ZStack(v ...View) ZStackView {
	return ZStackView{base{stackNode{axisZ, v, defaultGap, Center}}}
}

// Alignment aligns the subviews in v on both axes.
//
// The default alignment is [Center].
func (v ZStackView) Alignment(a Alignment) ZStackView {
	n := v.base[0].(stackNode)
	n.align = a
	v.base = base{n}
	return v
}

const defaultGap = 8

type stackNode struct {
	dir      stackAxis
	subviews []View
	gap      float64
	align    Alignment
}

func (s stackNode) render(rc renderContext) box {
	rc.lc = axes[s.dir].lc
	rc.container = axes[s.dir].container
	// The stack fills an axis when any subview does.
	content, f := subviewsRendered(rc, s.subviews...)
	return box{
		fills:   f,
		attrs:   domi.Group(attr.Class(axes[s.dir].class), s.attrs()),
		content: content,
	}
}

// attrs returns the stack's gap and alignment attributes.
func (s stackNode) attrs() domi.Attr {
	var a []domi.Attr
	if s.gap != defaultGap {
		a = append(a, attr.Style("gap:"+cssPx(s.gap)))
	}
	if s.align != Center {
		switch s.dir {
		case axisV:
			a = append(a, alignClass(s.align.horizontal()))
		case axisH:
			a = append(a, alignClass(s.align.vertical()))
		default:
			a = append(a, attr.Class(s.align.placeClass()))
		}
	}
	return domi.Group(a...)
}

// A Spacer occupies empty space.
// It expands along the major axis of the nearest enclosing stack.
// If there is no major axis, such as inside a [ZStack],
// it takes no space.
func Spacer() View { return base{spacerNode{}} }

type spacerNode struct{}

func (spacerNode) render(rc renderContext) box {
	var a domi.Attr
	if rc.lc.majorAxis.hasAll(Horizontal) {
		a = attr.Class("ui-spacer-h")
	}
	if rc.lc.majorAxis.hasAll(Vertical) {
		a = attr.Class("ui-spacer-v")
	}
	return box{
		fills: rc.lc.majorAxis,
		attrs: domi.Group(attr.Class("ui-spacer"), a),
	}
}

// A Divider is a thin line that can be used to separate other views.
// It expands along the minor axis of the nearest enclosing stack.
// If there is no major axis, such as inside a [ZStack],
// it expands horizontally.
func Divider() View { return base{dividerNode{}} }

type dividerNode struct{}

func (dividerNode) render(rc renderContext) box {
	f := rc.lc.minorAxes()
	a := []domi.Attr{attr.Class("ui-divider")}
	if rc.lc.majorAxis.hasAll(Horizontal) {
		// Major axis horizontal: vertical line.
		a = append(a, attr.Class("ui-divider-v"))
		if rc.unbounded.hasAll(Vertical) {
			a = append(a, attr.Class("ui-divider-ideal-y"))
			f &^= Vertical
		}
	} else {
		a = append(a, attr.Class("ui-divider-h"))
		if rc.unbounded.hasAll(Horizontal) {
			a = append(a, attr.Class("ui-divider-ideal-x"))
			f &^= Horizontal
		}
	}
	return box{
		fills: f,
		attrs: domi.Group(a...),
	}
}
