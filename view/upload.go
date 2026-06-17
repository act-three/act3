package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

// buttonUpload renders a file-upload form.
func buttonUpload() domi.Element {
	return func(nodes ...domi.Node) domi.Node {
		return html.Form(
			Class("v-button-upload"),
			attr.Method("POST"),
			attr.Enctype("multipart/form-data"),
			attr.Action("/-/do/upload"),
			stimulus.Controller("upload"),
		)(
			html.Input(
				Class("v-button-upload-picker"),
				attr.Type("file"),
				attr.Name("file"),
				stimulus.Target("upload", "picker"),
				stimulus.Action("change->upload#upload"),
			),
			Group(nodes...),
			html.Button(
				Class("v-button-upload-button"),
				stimulus.Action("click->upload#open:prevent"),
			),
			html.Div(
				Class("v-button-upload-overlay"),
			)(
				Icon("line/upload-03"),
			),
		)
	}
}
