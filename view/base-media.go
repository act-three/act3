package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
)

func media(title string) html.Element {
	return func(child ...html.Node) html.Node {
		return base(title)(Class("pt-14"))(
			Group(child...),
			mediaNavigationMenu(),
		)
	}
}

func mediaNavigationMenu() html.Node {
	return FlexCol(
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
		FlexRow(
			Gap8,
			Class(`
				relative
				items-center
				p-4
			`),
		)(
			Link("/")(Icon("spotlight"), Text("Act Three")),
			Button(attr.Href("/account/profile"))(Icon("settings")),
		),
	)
}
