package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// TextField renders an inline-updating text field.
// It POSTs to action with a form body containing all
// child hidden inputs plus the text input's value on blur.
// Children become hidden inputs (for context like IDs).
func TextField(action, name, value string, attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Form(
			attr.Class("u-text-field"),
			stimulus.Controller("text-field"),
			stimulus.Value("text-field", "url")(action),
			attr.Group(attrs...),
		)(append(nodes,
			InputText(
				attr.Name(name),
				attr.Value(value),
				stimulus.Target("text-field", "input"),
				stimulus.Action("blur->text-field#save"),
				stimulus.Action("keydown->text-field#keydown"),
			),
		)...)
	}
}
