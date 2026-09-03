package ui

import (
	"cmp"

	"ily.dev/act3/xui/internal/canon"
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
	return stackView{base{nodeStack(axisV, v)}}
}

// An HStack arranges its subviews in a horizontal line.
func HStack(v ...View) StackView {
	return stackView{base{nodeStack(axisH, v)}}
}

type stackView struct{ base }

func (v stackView) Alignment(a Alignment) StackView {
	v.base = v.modify(modAlignment(a))
	return v
}

func (v stackView) Gap(px float64) StackView {
	v.base = v.modify(modGap(px))
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
	return zstackView{base{nodeStack(axisZ, v)}}
}

type zstackView struct{ base }

func (v zstackView) Alignment(a Alignment) ZStackView {
	v.base = v.modify(modAlignment(a))
	return v
}

const defaultGap float64 = 8

func nodeStack(dir stackAxis, subviews []View) node {
	return func(env environment) box {
		inner := env
		inner.lc = axes[dir].lc
		inner.container = axes[dir].container
		subviews := Group(subviews...)
		if dir == axisZ {
			subviews = subviews.
				modify(modStyle("grid-row-start", "1")).
				modify(modStyle("grid-column-start", "1")).
				// A subview that forms a stacking context, or is positioned,
				// would otherwise paint above every later subview that does
				// neither. With z-index:0, each subview paints in order,
				// within the stack's own stacking context.
				modify(modStyle("z-index", "0"))
		}
		p := renderSubviewList(inner, subviews)
		env.tag = cmp.Or(env.tag, axes[dir].tag)
		gap := *cmp.Or(env.gap, new(defaultGap))
		dir.addStackStylesTo(&env.style, gap, env.alignment)
		return build(env, p)
	}
}

// addStackStylesTo adds the declarations of a stack along dir to ss:
// its display and flow, its gap,
// and its alignment — the minor-axis projection for a line,
// both axes for a ZStack.
func (dir stackAxis) addStackStylesTo(ss *canon.StyleSet, gap float64, align Alignment) {
	switch dir {
	case axisZ:
		ss.Set("display", "grid")
		ss.Set("grid-template-columns", "100%")
		ss.Set("grid-template-rows", "100%")
		align.setItemsOn(ss)
		ss.Set("isolation", "isolate") // Z-Index Rule. See theory.go.
		return
	case axisH:
		ss.Set("flex-direction", "row")
		ss.Set("align-items", align.vertical().keyword())
	case axisV:
		ss.Set("flex-direction", "column")
		ss.Set("align-items", align.horizontal().keyword())
	}
	ss.Set("display", "inline-flex")
	ss.Set("row-gap", cssPx(gap))
	ss.Set("column-gap", cssPx(gap))
}

// A Spacer occupies empty space.
// It expands along the major axis of the nearest enclosing stack.
// If there is no major axis, such as inside a [ZStack],
// it takes no space.
func Spacer() View { return base{nodeSpacer} }

func nodeSpacer(env environment) box {
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
func Divider() View { return base{nodeDivider} }

func nodeDivider(env environment) box {
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
