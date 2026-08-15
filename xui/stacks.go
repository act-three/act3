package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/xui/internal/sheet"
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

func (s stackNode) render(env environment) box {
	inner := env
	inner.lc = axes[s.dir].lc
	inner.container = axes[s.dir].container
	// The stack fills an axis when any subview does.
	content, f := subviewsRendered(inner, s.subviews...)
	env.add(attr.Class(axes[s.dir].class))
	s.addAlignStyleTo(&env.style)
	if s.dir != axisZ {
		env.style.Set("gap", cssPx(s.gap))
	}
	return build(env, plan{
		fills:   f,
		content: content,
	})
}

// addAlignStyleTo adds the stack's alignment declaration to ss:
// the minor-axis projection for a line, both axes for a ZStack.
func (s stackNode) addAlignStyleTo(ss *sheet.StyleSet) {
	switch s.dir {
	case axisV:
		ss.Set("align-items", s.align.horizontal().keyword())
	case axisH:
		ss.Set("align-items", s.align.vertical().keyword())
	default:
		ss.Set("place-items", s.align.placeItems())
	}
}

// A Spacer occupies empty space.
// It expands along the major axis of the nearest enclosing stack.
// If there is no major axis, such as inside a [ZStack],
// it takes no space.
func Spacer() View { return base{spacerNode{}} }

type spacerNode struct{}

func (spacerNode) render(env environment) box {
	var a domi.Attr
	if env.lc.majorAxis.hasAll(Horizontal) {
		a = attr.Class("ui-spacer-h")
	}
	if env.lc.majorAxis.hasAll(Vertical) {
		a = attr.Class("ui-spacer-v")
	}
	env.add(attr.Class("ui-spacer"), a)
	return build(env, plan{fills: env.lc.majorAxis})
}

// A Divider is a thin line that can be used to separate other views.
// It expands along the minor axis of the nearest enclosing stack.
// If there is no major axis, such as inside a [ZStack],
// it expands horizontally.
func Divider() View { return base{dividerNode{}} }

type dividerNode struct{}

func (dividerNode) render(env environment) box {
	env.add(attr.Class("ui-divider"))
	if env.lc.majorAxis.hasAll(Horizontal) {
		// Major axis horizontal: vertical line.
		env.add(attr.Class("ui-divider-v"))
		if env.unbounded.hasAll(Vertical) {
			env.add(attr.Class("ui-divider-ideal-y"))
		}
	} else {
		env.add(attr.Class("ui-divider-h"))
		if env.unbounded.hasAll(Horizontal) {
			env.add(attr.Class("ui-divider-ideal-x"))
		}
	}
	return build(env, plan{fills: env.lc.minorAxes()})
}
