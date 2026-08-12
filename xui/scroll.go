package ui

import (
	"cmp"

	"ily.dev/domi"
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
	// Along a scroll axis, the content's available space is unbounded.
	// On a non-scrolling axis the available space is the viewport's own size.
	inner := environment{unbounded: s.along, sheet: env.sheet}
	variant := cmp.Or(map[AxisSet]string{
		Horizontal:            "ui-scroll-x",
		Vertical:              "ui-scroll-y",
		Horizontal | Vertical: "ui-scroll-xy",
	}[s.along], "ui-scroll-none")
	// The scroll viewport is a single-cell grid establishing no axes.
	// It is equivalent to the root view context in a scrolling web page.
	a := []domi.Attr{attr.Class("ui-scroll", variant)}
	if env.unbounded.hasAll(Horizontal) {
		a = append(a, attr.Class("ui-scroll-ideal-x"))
	}
	if env.unbounded.hasAll(Vertical) {
		a = append(a, attr.Class("ui-scroll-ideal-y"))
	}
	a = append(a, env.shapeClass())
	content, _ := subviewsRendered(inner, s.contents)
	return box{
		fills:   (Horizontal | Vertical) &^ env.unbounded,
		attrs:   domi.Group(a...),
		content: content,
	}
}
