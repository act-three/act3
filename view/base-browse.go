package view

import (
	"math/rand/v2"
	"slices"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/web/static"
)

func browse(title string, washURL ...string) html.Element {
	return func(child ...html.Node) html.Node {
		return base(title)()(
			browseWash(washURL),
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

func browseWash(urls []string) html.Node {
	urls = slices.DeleteFunc(urls, func(s string) bool { return s == "" })
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
		stimulus.Controller("topbar"),
		Class("v-media-nav"),
	)(
		browseContainer(
			Grid12()(
				FlexRow(
					ColSpan12,
					Gap8,
					Class("v-media-nav-row"),
				)(
					Button(attr.Href("/"), ButtonGhost)(Icon("line/spotlight"), Text("Act Three")),
					Button(attr.Href("/collections"), ButtonGhost)(Text("Collections")),
					Box(Class("v-media-nav-spacer")),
					Button(attr.Href("/app/profile"), ButtonGhost)(Icon("line/settings-01")),
				),
			),
		),
	)
}
