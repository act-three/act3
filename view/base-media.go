package view

import (
	"ily.dev/act3/html"
	. "ily.dev/act3/ui"
)

func media(title string) html.Element {
	return func(child ...html.Node) html.Node {
		return base(title)()(
			Box(
				Class("inset-x-0 bottom-0 flex flex-col items-center"),
			)(
				Box(
					Class(`
						flex
						flex-row
						items-center
						p-7
						gap-8
						rounded-2xl
						bg-gray-2
					`),
				)(
					Link("/")(Text("Home")),
					Text("Act Three"),
					Link("/account/profile")(Text("Settings")),
				),
			),
			Group(child...),
		)
	}
}
