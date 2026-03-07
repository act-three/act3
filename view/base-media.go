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

func media(title string, washURL ...string) html.Element {
	return func(child ...html.Node) html.Node {
		return base(title)()(
			mediaWash(washURL),
			pageContainer(child...),
			mediaNavigationMenu(),
			turbo.Frame("player"),
		)
	}
}

func mediaWash(urls []string) html.Node {
	urls = slices.DeleteFunc(urls, func(s string) bool { return s == "" })
	url := static.Path("/static/cb.jpeg")
	if len(urls) > 0 {
		url = urls[rand.IntN(len(urls))]
	}
	return Box(Class("fixed inset-0 -z-1 blur-3xl saturate-180 opacity-20 scale-110"))(
		html.Img(
			Class("w-full h-full object-cover"),
			attr.Src(url),
		),
	)
}

func mediaNavigationMenu() html.Node {
	return FlexCol(
		stimulus.Controller("topbar"),
		Class(`
			fixed
			top-0
			w-full
			items-center

			backdrop-blur-xl
			backdrop-brightness-150
			backdrop-saturate-180
			bg-black/60
		`),
	)(
		pageContainer(
			Grid12()(
				FlexRow(
					ColSpan12,
					Gap8,
					Class(`
						relative
						items-center
						py-4
					`),
				)(
					Button(attr.Href("/"), ButtonGhost)(Icon("spotlight"), Text("Act Three")),
					Button(attr.Href("/collections"), ButtonGhost)(Text("Collections")),
					Box(Class("grow")),
					Button(attr.Href("/app/profile"), ButtonGhost)(Icon("settings")),
				),
			),
		),
	)
}
