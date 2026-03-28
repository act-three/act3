package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

// Port is the container where dialogs and popovers are appended.
// Place it in the base app layout before [NotePort]
// so that notifications naturally stack above.
func Port() html.Node {
	return html.Div(attr.ID("port"))()
}
