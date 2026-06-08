package view

import (
	"ily.dev/domi"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
)

func buttonImageEdit(open msg.Msg, im model.Image, a Aspect) domi.Node {
	return html.Div(
		Class("v-button-image-edit"),
		Stylef("aspect-ratio: %s", a),
	)(
		html.Button(onClick(open))(
			PosterImg(a, PosterFill, imgAttrs(im)),
		),
		html.Div(Class("v-button-image-edit-overlay"))(
			Icon("line/edit-02"),
		),
	)
}
