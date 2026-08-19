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
	type variant struct{ class, x, y string }
	v := cmp.Or(map[AxisSet]variant{
		Horizontal:            {"ui-scroll-x", "auto", "hidden"},
		Vertical:              {"ui-scroll-y", "hidden", "auto"},
		Horizontal | Vertical: {"ui-scroll-xy", "auto", "auto"},
	}[s.along], variant{"ui-scroll-none", "clip", "clip"})
	// The scroll viewport is a single-cell grid establishing no axes.
	// It is equivalent to the root view context in a scrolling web page.
	env.add(attr.Class("ui-scroll", v.class))
	env.style.Set("display", "grid")
	env.style.Set("min-width", "0")
	env.style.Set("min-height", "0")
	env.style.Set("overflow-x", v.x)
	env.style.Set("overflow-y", v.y)
	env.style.Set("overscroll-behavior", "contain")
	env.style.Set("place-items", "start")
	env.style.Set("contain", "size") // Viewport size doesn't depend on its contents.
	content, _ := subviewsRendered(environment{sheet: env.sheet},
		s.contents.
			Modify(modFixedSize{axes: s.along}),
	)
	p := plan{
		fills:   Horizontal | Vertical,
		ideal:   rect{width: newSize(100), height: newSize(100)},
		content: content,
	}
	return build(env, p)
}
