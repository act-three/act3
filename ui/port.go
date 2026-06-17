package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"
)

// Port is the container where dialogs and popovers are appended.
// Place it in the base app layout before [NotePort]
// so that notifications naturally stack above.
func Port() domi.Node {
	return html.Div(attr.ID("port"))()
}
