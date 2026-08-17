package ui

import (
	"cmp"

	"ily.dev/domi/attr"
)

// A ScrollView displays v inside a viewport
// that scrolls along the requested axis.
// If v is larger than the viewport,
// part of it will be hidden until scrolled into view.
//
// To request scrolling along both axes,
// combine the axis values with bitwise-or:
//
//	ScrollView(Horizontal|Vertical, v)
//
// The viewport expands to fill available space along both axes,
// regardless of the specified scroll axis.
func ScrollView(axis AxisSet, v View) View {
	return base{scrollNode{along: axis, contents: v}}
}

type scrollNode struct {
	along    AxisSet
	contents View
}

func (s scrollNode) render(env environment) box {
	// A stroke's carrier would sit in the scrollable overflow and
	// scroll away with the content, so pending strokes box out
	// around the viewport.
	if len(env.stroke) > 0 {
		return wrapMod(env, s)
	}
	// Along a scroll axis, the content's available space is unbounded.
	// On a non-scrolling axis the available space is the viewport's own size.
	variant := cmp.Or(map[AxisSet]string{
		Horizontal:            "ui-scroll-x",
		Vertical:              "ui-scroll-y",
		Horizontal | Vertical: "ui-scroll-xy",
	}[s.along], "ui-scroll-none")
	// The scroll viewport is a single-cell grid establishing no axes.
	// It is equivalent to the root view context in a scrolling web page.
	env.add(attr.Class("ui-scroll", variant))
	env.style.Set("place-items", "start")
	// TODO: fold these classes into plan.ideal. Its min-* lowering
	// assumes a box's content contributes nothing beyond the ideal,
	// but an overflow box contributes its full content size, so the
	// viewport needs a definite size to force overflow. contain: size
	// + contain-intrinsic-size would cap the contribution at the
	// ideal, making plan.ideal's contract hold here too.
	if env.unbounded.hasAll(Horizontal) {
		env.add(attr.Class("ui-scroll-ideal-x"))
	}
	if env.unbounded.hasAll(Vertical) {
		env.add(attr.Class("ui-scroll-ideal-y"))
	}
	content, _ := subviewsRendered(environment{sheet: env.sheet},
		s.contents.
			Modify(modFixedSize{axes: s.along}),
	)
	p := plan{
		fills:   Horizontal | Vertical,
		content: content,
	}
	return build(env, p)
}
