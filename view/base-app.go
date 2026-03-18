package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view/sidebar"
)

func app(title string, child ...html.Node) html.Node {
	return base(title)()(
		turbo.StreamSource("/-/events"),
		html.Div(
			attr.Attr("data-slot")("sidebar-wrapper"),
			attr.Class("v-app"),
			attr.Style("--sidebar-width: 200px; --sidebar-width-mobile: 20rem;"),
		)(
			sidebar.Sidebar(),
			html.Div(
				attr.Role("main"),
				attr.Attr("data-slot")("sidebar-inset"),
				attr.Class("v-app-main"),
			)(
				turbo.Frame("main",
					turbo.DataAction("advance"),
				)(
					child...,
				),
			),
		),
		turbo.Frame("dialog"),
		NotePort(),
	)
}

func PageFrame(title, id string, child ...html.Node) html.Node {
	return Group(
		html.Title()(html.Text(title)),
		turbo.Frame(id)(child...),
	)
}

func DialogButton(url string, attrs ...attr.Node) html.Element {
	return Button(
		attr.Href(url),
		attr.Attr("data-turbo-frame")("dialog"),
		attr.Group(attrs...),
	)
}

func Dialog(children ...html.Node) html.Node {
	return turbo.Frame("dialog")(
		html.Div(
			attr.Class("v-dialog"),
			attr.Attr("data-controller")("dialog"),
		)(
			html.Div(
				attr.Class("v-dialog-backdrop"),
				attr.Attr("data-action")("click->dialog#dismiss:self"),
			),
			html.Div(
				attr.Class("v-dialog-positioner"),
			)(
				html.Div(
					attr.Class("v-dialog-panel"),
				)(
					html.Group(children...),
					html.Div(
						attr.Class("v-dialog-close"),
						attr.Attr("data-action")("click->dialog#dismiss"),
					)(
						Icon("line/x-close"),
					),
				),
			),
		),
	)
}
