package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

func buttonPosterEdit(dialogURL string, im model.Image, addr []string) html.Node {
	return buttonImageEdit(dialogURL, im, addr, "2 / 3")
}

func buttonBannerEdit(dialogURL string, im model.Image, addr []string) html.Node {
	return buttonImageEdit(dialogURL, im, addr, "1000 / 185")
}

func buttonThumbnailEdit(dialogURL string, im model.Image, addr []string) html.Node {
	return buttonImageEdit(dialogURL, im, addr, "16 / 9")
}

func buttonImageEdit(dialogURL string, im model.Image, addr []string, ratio string) html.Node {
	return html.Div(
		Class("v-button-image-edit"),
		attr.Style("aspect-ratio: "+ratio),
	)(
		html.Button(
			stimulus.Controller("dialog-trigger"),
			stimulus.Value("dialog-trigger", "url")(dialogURL),
			stimulus.Action("click->dialog-trigger#open"),
		)(
			PosterImg(PosterFill, imgAttrs(im, addr)),
		),
		html.Div(Class("v-button-image-edit-overlay"))(
			Icon("line/edit-02"),
		),
	)
}
