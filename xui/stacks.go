package ui

import (
	"cmp"

	"ily.dev/act3/xui/internal/sheet"
)

// A StackView displays its subviews in a line.
type StackView interface {
	View

	// Alignment aligns the subviews in the stack.
	// They are placed along the stack's minor axis
	// so their alignment points are colinear.
	//
	// The default alignment is Center.
	Alignment(Alignment) StackView

	// Gap sets the distance between adjacent subviews
	// in the stack. The default gap is 8 px.
	Gap(px float64) StackView
}

// A VStack arranges its subviews in a vertical line.
func VStack(v ...View) StackView {
	return stackView{base{stackNode{axisV, v, defaultGap, Center}}}
}

// An HStack arranges its subviews in a horizontal line.
func HStack(v ...View) StackView {
	return stackView{base{stackNode{axisH, v, defaultGap, Center}}}
}

type stackView struct{ base }

func (v stackView) Alignment(a Alignment) StackView {
	n := v.base[0].(stackNode)
	n.align = a
	v.base = base{n}
	return v
}

func (v stackView) Gap(px float64) StackView {
	n := v.base[0].(stackNode)
	n.gap = px
	v.base = base{n}
	return v
}

// A ZStackView overlays its subviews, aligning them in both axes.
//
// It assigns each successive subview a higher z-axis value
// than the one before it,
// so later subviews obscure earlier ones where they overlap.
type ZStackView interface {
	View

	// Alignment aligns the subviews in the stack on both axes.
	//
	// The default alignment is Center.
	Alignment(Alignment) ZStackView
}

// ZStack overlays the given views.
func ZStack(v ...View) ZStackView {
	return zstackView{base{stackNode{axisZ, v, defaultGap, Center}}}
}

type zstackView struct{ base }

func (v zstackView) Alignment(a Alignment) ZStackView {
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
	subviews := Group(s.subviews...)
	if s.dir == axisZ {
		subviews = subviews.modify(modStyle{"grid-area", "1 / 1"})
	}
	content, f := subviewsRendered(inner, subviews)
	env.tag = cmp.Or(env.tag, axes[s.dir].tag)
	s.addStackStylesTo(&env.style)
	return build(env, plan{
		fills:   f,
		content: content,
	})
}

// addStackStylesTo adds the stack's declarations to ss:
// its display and flow, its gap,
// and its alignment — the minor-axis projection for a line,
// both axes for a ZStack.
func (s stackNode) addStackStylesTo(ss *sheet.StyleSet) {
	switch s.dir {
	case axisZ:
		ss.Set("display", "grid")
		ss.Set("grid-template-columns", "100%")
		ss.Set("grid-template-rows", "100%")
		ss.Set("place-items", s.align.placeItems())
		return
	case axisH:
		ss.Set("flex-direction", "row")
		ss.Set("align-items", s.align.vertical().keyword())
	case axisV:
		ss.Set("flex-direction", "column")
		ss.Set("align-items", s.align.horizontal().keyword())
	}
	ss.Set("display", "inline-flex")
	ss.Set("gap", cssPx(s.gap))
}

// A Spacer occupies empty space.
// It expands along the major axis of the nearest enclosing stack.
// If there is no major axis, such as inside a [ZStack],
// it takes no space.
func Spacer() View { return base{spacerNode{}} }

type spacerNode struct{}

func (spacerNode) render(env environment) box {
	env.tag = cmp.Or(env.tag, "ui-spacer")
	env.style.Set("flex-basis", "0")
	minWidth, minHeight := "0", "0"
	if env.lc.majorAxis.hasAll(Horizontal) {
		minWidth = "8px"
	}
	if env.lc.majorAxis.hasAll(Vertical) {
		minHeight = "8px"
	}
	env.style.Set("min-width", minWidth)
	env.style.Set("min-height", minHeight)
	return build(env, plan{fills: env.lc.majorAxis})
}

// A Divider is a thin line that can be used to separate other views.
// It expands along the minor axis of the nearest enclosing stack.
// If there is no major axis, such as inside a [ZStack],
// it expands horizontally.
func Divider() View { return base{dividerNode{}} }

type dividerNode struct{}

func (dividerNode) render(env environment) box {
	env.tag = cmp.Or(env.tag, "ui-divider")
	env.bg = append(env.bg, term[color]{value: borderColor.color()})
	p := plan{fills: env.lc.minorAxes(), rigid: env.lc.majorAxis}
	if env.lc.majorAxis.hasAll(Horizontal) {
		// Major axis horizontal: vertical line.
		env.style.Set("width", "1px")
		p.ideal.height = newSize(10)
	} else {
		env.style.Set("height", "1px")
		p.ideal.width = newSize(10)
	}
	return build(env, p)
}
