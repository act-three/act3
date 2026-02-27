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
		html.Div(
			attr.Attr("data-slot")("sidebar-wrapper"),
			attr.Class(`
				group/sidebar-wrapper
				isolate
				bg-gray-2
				flex
				min-h-svh
				w-full
				app
				h-svh
			`),
			attr.Style("--sidebar-width: 200px; --sidebar-width-mobile: 20rem;"),
		)(
			sidebar.Sidebar(),
			turbo.Frame("main",
				attr.Role("main"),
				turbo.DataAction("advance"),
				attr.Attr("data-slot")("sidebar-inset"),
				attr.Class(`
					bg-gray-1
					relative
					w-full
					flex-1
					flex-col
					grid
					place-items-center
					grid-cols-1
					grid-rows-1
					m-2
					ms-0
					rounded-xl
					shadow-sm
				`),
			)(
				child...,
			),
		),
		turbo.Frame("dialog"),
		NoteViewport(),
	)
}

func PageFrame(title, id string, child ...html.Node) html.Node {
	return Group(
		html.Title()(html.Text(title)),
		turbo.Frame(id)(child...),
	)
}

func DialogButton(url string) html.Element {
	return Button(
		attr.Href(url),
		attr.Attr("data-turbo-frame")("dialog"),
	)
}

func Dialog(children ...html.Node) html.Node {
	return turbo.Frame("dialog")(
		html.Div(
			attr.Class(`fixed inset-[0]`),
			attr.Attr("data-controller")("dialog"),
		)(
			html.Div(
				attr.Class(`
					absolute
					inset-[0]
					bg-[rgba(180,180,180,0.6)]
				`),
				attr.Attr("data-action")("click->dialog#dismiss:self"),
			),
			html.Div(
				attr.Class(`
					flex
					absolute
					top-1/2
					left-1/2
					-translate-1/2
					max-h-dvh
					max-w-dvw
					p-12
					pointer-events-none
				`),
			)(
				html.Div(
					attr.Class(`
						relative
						max-h-full
						flex
						shadow-xl
						rounded-sm
						bg-gray-1
						p-6
						pointer-events-auto
					`),
				)(
					html.Group(children...),
					html.Div(
						attr.Class(`
							absolute
							top-2
							right-2
							size-6
							p-1
							rounded-full
							bg-gray-2
							flex
						`),
						attr.Attr("data-action")("click->dialog#dismiss"),
					)(
						Icon("x"),
					),
				),
			),
		),
	)
}
