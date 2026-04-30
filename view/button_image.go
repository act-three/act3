package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

func buttonImageEdit(dialogURL string, im model.Image, addr []string, a Aspect) html.Node {
	return html.Div(
		Class("v-button-image-edit"),
		Stylef("aspect-ratio: %s", a),
	)(
		html.Button(
			stimulus.Controller("dialog-trigger"),
			stimulus.Value("dialog-trigger", "url")(dialogURL),
			stimulus.Action("click->dialog-trigger#open"),
		)(
			PosterImg(a, PosterFill, imgAttrs(im, addr)),
		),
		html.Div(Class("v-button-image-edit-overlay"))(
			Icon("line/edit-02"),
		),
	)
}
