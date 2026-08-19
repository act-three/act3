package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// HTML displays node inside an adapter HTML element.
//
// The returned view expands to fill available space.
// Use [View.FixedSize] to size it by its content instead.
//
// The adapter element provides a CSS layout context for node.
// Initially, it has the following CSS declarations:
//
//	display: grid;
//	grid-template-columns: 100%;
//	grid-template-rows: 100%;
//	place-items: center;
//
// Client code is free to customize
// the adapter element's interior layout
// using [View.Class] and [View.Attr] with [attr.Style].
//
//	HTML(myToolbar).
//		Attr(attr.Style("place-items:stretch"))
//
//	HTML(domi.Fragment(myButton1, myButton2)).
//		Attr(attr.Style("display:flex")).
//		Attr(attr.Style("gap:8px")).
//		FixedSize()
func HTML(node domi.Node) View { return base{nodeHTML{node: node}} }

type nodeHTML struct{ node domi.Node }

func (h nodeHTML) render(env environment) box {
	env.add(attr.Class("ui-html"))
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	env.style.Set("place-items", Center.placeItems())
	p := plan{
		fills:   Horizontal | Vertical,
		content: h.node,
	}
	return build(env, p)
}
