package ui

import (
	"cmp"
	"reflect"
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
//
// When a ScrollView is the root view of the page,
// it uses document viewport scrolling
// instead of HTML element scrolling.
// This lets the browser
// navigate to page fragment anchors
// and save scroll position during page reload and navigation.
func ScrollView(axis AxisSet, v View) View {
	return base{nodeScroll{
		along:    axis,
		contents: unary(VStack, v),
	}.render}
}

type nodeScroll struct {
	along    AxisSet
	contents node
}

func (s nodeScroll) render(env environment) box {
	// A stroke's carrier would sit in the scrollable overflow and
	// scroll away with the content, so pending strokes box out
	// around the viewport.
	if len(env.stroke) > 0 {
		return wrapMod(env, s.render)
	}
	inner := env
	inner.root.atRoot = false
	inner.lc = layoutContext{}
	inner.container = containerGrid
	inner.unbounded = 0
	contents := modFixedSize(s.along)(s.contents)
	if canScrollDocument(env) {
		b := contents(inner)
		b.pageScroll = s.along
		return b
	}
	// Along a scroll axis, the content's available space is unbounded.
	// On a non-scrolling axis the available space is the viewport's own size.
	type overflow struct{ x, y string }
	v := cmp.Or(map[AxisSet]overflow{
		Horizontal:            {"auto", "hidden"},
		Vertical:              {"hidden", "auto"},
		Horizontal | Vertical: {"auto", "auto"},
	}[s.along], overflow{"clip", "clip"})
	// The scroll viewport is a single-cell grid establishing no axes.
	// It is equivalent to the root view context in a scrolling web page.
	env.tag = cmp.Or(env.tag, "ui-scroll")
	env.style.Set("display", "grid")
	env.style.Set("min-width", "0")
	env.style.Set("min-height", "0")
	env.style.Set("overflow-x", v.x)
	env.style.Set("overflow-y", v.y)
	env.style.Set("overscroll-behavior-x", "contain")
	env.style.Set("overscroll-behavior-y", "contain")
	TopLeading.setItemsOn(&env.style)
	env.style.Set("contain", "size")      // Viewport size doesn't depend on its contents.
	env.style.Set("isolation", "isolate") // Isolate the wrapSticky z-index.
	p := renderSubviewNode(inner, contents)
	p.fills = Horizontal | Vertical
	p.rigid = 0 // Content rigidity does not escape its viewport.
	p.ideal = rect{width: newSize(100), height: newSize(100)}
	return build(env, p)
}

// canScrollDocument reports whether removing the ScrollView's viewport box
// preserves every pending next-box value. Persistent subtree context and root
// environment requirements remain valid when applied to the contents; every
// ordinary one-shot value remains destined for the viewport box.
//
// Checking the whole remaining value makes future nextenv fields reject this
// lowering by default until ScrollView explicitly accommodates them.
func canScrollDocument(env environment) bool {
	if !env.root.atRoot {
		return false
	}
	return reflect.ValueOf(env.nextenv).IsZero()
}
