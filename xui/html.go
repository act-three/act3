package ui

import "ily.dev/domi"

// HTML displays an HTML node as a view.
//
// Certain modifier methods cause the returned view to panic.
func HTML(n domi.Node) View { return base{domiNode{n: n}} }

type domiNode struct{ n domi.Node }

func (d domiNode) render(env environment) box {
	if p := env.takePending(); p.attrs != nil || p.tag != "" {
		panic("ui: modifier applied to a Domi view, which has no ui-managed element")
	}
	return box{raw: d.n}
}
