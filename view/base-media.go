package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

func media(title string) html.Element {
	return func(child ...html.Node) html.Node {
		return base(title)()(
			Group(child...),
			mediaNavigationMenu(),
			turbo.Frame("player"),
		)
	}
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
				Button(attr.Href("/account/profile"), ButtonGhost)(Icon("settings")),
			),
		),
	)
}
