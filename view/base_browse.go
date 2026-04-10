package view

import (
	"math/rand/v2"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/web/static"
)

func browse(title string, washImages ...model.Image) html.Element {
	return func(child ...html.Node) html.Node {
		return base(title)(attr.Style("padding:var(--nav-h) 0 8rem"))(
			browseWash(washImages),
			browseContainer(child...),
			browseNavigationMenu(),
			turbo.Frame("player"),
		)
	}
}

func browseContainer(child ...html.Node) html.Node {
	return FlexCol(Class("v-media-container"))(
		Group(child...),
	)
}

func browseWash(images []model.Image) html.Node {
	var urls []string
	for _, im := range images {
		if im.IsPlaceholder() {
			continue
		}
		urls = append(urls, im.SmallestURL())
	}
	url := static.Path("/static/cb.jpeg")
	if len(urls) > 0 {
		url = urls[rand.IntN(len(urls))]
	}
	return Box(Class("v-media-wash"))(
		PosterImg(PosterFill, attr.Src(url)),
	)
}

func browseNavigationMenu() html.Node {
	return FlexCol(
		Class("v-media-nav"),
	)(
		browseContainer(
			Grid12()(
				FlexRow(
					ColSpan12,
					Gap8,
					Class("v-media-nav-row"),
				)(
					Link("/")(wordmark()),
					Button(attr.Href("/collections"), ButtonGhost)(Text("Collections")),
					Box(Class("v-media-nav-spacer")),
					Button(attr.Href("/app/profile"), ButtonGhost)(Icon("line/settings-01")),
				),
			),
		),
	)
}
