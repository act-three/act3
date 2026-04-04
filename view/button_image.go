package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
)

func buttonPosterEdit(dialogURL, imageURL string) html.Node {
	return buttonImageEdit(dialogURL, imageURL, "2 / 3")
}

func buttonBannerEdit(dialogURL, imageURL string) html.Node {
	return buttonImageEdit(dialogURL, imageURL, "1000 / 185")
}

func buttonThumbnailEdit(dialogURL, imageURL string) html.Node {
	return buttonImageEdit(dialogURL, imageURL, "16 / 9")
}

func buttonImageEdit(dialogURL, imageURL, ratio string) html.Node {
	return html.Div(
		Class("v-button-image-edit"),
		attr.Style("aspect-ratio: "+ratio),
	)(
		html.Form(attr.Method("get"), attr.Action(dialogURL))(
			html.Button()(
				PosterImg(PosterFill, attr.Src(imageURL)),
			),
		),
		html.Div(Class("v-button-image-edit-overlay"))(
			Icon("line/edit-02"),
		),
	)
}
