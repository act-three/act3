package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func EditSystemTransmission(config *model.ConfigTransmission) html.Node {
	return app("Transmission",
		html.Div(
			attr.Class("h-full w-full p-4"),
		)(
			html.Div()(html.Text("Transmission")),
			html.Form(
				attr.Method("post"),
				attr.Action("/-/do/update-transmission-settings"),
			)(
				html.Div(attr.Class("py-4"))(
					html.Div()(html.Text("RPC URL")),
					InputText(
						attr.Name("url"),
						attr.Class("max-w-xs"),
						attr.Value(config.BaseURL),
					),
				),
				html.Div(attr.Class("py-4"))(
					html.Div()(html.Text("Download Folder")),
					InputText(
						attr.Name("path"),
						attr.Class("max-w-xs"),
						attr.Value(config.Path),
					),
				),
				html.Div(attr.Class("py-4"))(
					InputSubmit(
						attr.Value("Save"),
					),
				),
			),
		),
	)
}
