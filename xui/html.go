package ui

import "ily.dev/domi"

// HTML displays an HTML node as a view.
//
// The view must be placed directly inside a container.
// If a modifier is used on the view, it panics.
func HTML(n domi.Node) View { return base{domiNode{n: n}} }

type domiNode struct{ n domi.Node }

func (d domiNode) render(renderContext) box {
	return box{raw: d.n}
}
