package sidebar

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

type MenuSection struct {
	Label string
	Items []MenuItem
}

type MenuItem struct {
	Icon  string
	Path  string
	Label string
	Attr  attr.Node
}

func sidebarData() []MenuSection {
	return []MenuSection{
		{
			Items: []MenuItem{
				{"line/spotlight", "/", "Act Three", attr.Attr("data-turbo-frame")("_top")},
			},
		},
		{
			Label: "Account",
			Items: []MenuItem{
				{"line/user-circle", "/app/profile", "Profile", nil},
				{"line/fingerprint-04", "/app/security", "Security", nil},
			},
		},
		{
			Label: "Edit Media",
			Items: []MenuItem{
				{"line/arrow-circle-down", "/app/downloads", "Downloads", nil},
				{"line/file-question-03", "/app/missing", "Missing Media", nil},
				{"line/trash-01", "/app/trash", "Trash", nil},
				{"line/film-01", "/app/movies", "All Movies", nil},
				{"line/tv-03", "/app/series", "All Series", nil},
			},
		},
		{
			Label: "System",
			Items: []MenuItem{
				{"line/cloud-01", "/app/transmission", "Download Client", nil},
				{"line/hard-drive", "/app/storage", "Storage", nil},
				{"line/calendar-check-01", "/app/tasks", "Tasks", nil},
			},
		},
	}
}

func Sidebar() html.Node {
	return turbo.Frame("sidebar",
		attr.Target("main"),
		attr.Class("text-gray-12 group peer md:block"),
		attr.Attr("data-state")(""),
		attr.Attr("data-collapsible")(""),
		attr.Attr("data-variant")("inset"),
		attr.Attr("data-side")("left"),
		attr.Attr("data-slot")("sidebar"),
		stimulus.Controller("sidebar"),
		stimulus.Action("turbo:visit@document->sidebar#visit"),
	)(
		html.Div(
			attr.Attr("data-slot")("sidebar-gap"),
			attr.Class(`
				w-(--sidebar-width)
				relative
				bg-transparent
				duration-200
			`),
		),
		html.Div(
			attr.Attr("data-slot")("sidebar-container"),
			attr.Class(`
				w-(--sidebar-width)
				fixed
				inset-y-0
				z-10
				h-svh
				duration-200
				md:flex
				start-0
				p-2
			`),
		)(
			html.Div(
				attr.Attr("data-sidebar")("sidebar"),
				attr.Attr("data-slot")("sidebar-inner"),
				attr.Class(`
					bg-gray-2
					flex
					h-full
					w-full
					flex-col
				`),
			)(
				sidebarContent(),
			),
		),
	)
}

func sidebarContent() html.Node {
	return html.Div(
		attr.Attr("data-slot")("sidebar-content"),
		attr.Attr("data-sidebar")("content"),
		attr.Class(`
			flex
			min-h-0
			flex-1
			flex-col
			gap-2
			overflow-auto
		`),
	)(
		html.Range(sidebarData(), sidebarGroup),
	)
}

func sidebarGroup(v MenuSection) html.Node {
	return html.Div(
		attr.Attr("data-slot")("sidebar-group"),
		attr.Attr("data-sidebar")("group"),
		attr.Class("relative flex w-full min-w-0 flex-col p-2"),
	)(
		html.If(v.Label != "", func() html.Node {
			return sidebarGroupLabel(v.Label)
		}),
		sidebarGroupContent(v.Items),
	)
}

func sidebarGroupLabel(s string) html.Node {
	return html.Div(
		attr.Attr("data-slot")("sidebar-group-label"),
		attr.Attr("data-sidebar")("group-label"),
		attr.Class(`
				text-gray-11/70
				ring-accent-6
				outline-hidden
				flex
				h-8
				shrink-0
				items-center
				rounded-md
				px-2
				text-xs
				font-medium
				duration-200
				focus-visible:ring-2
				[&>svg]:size-4
				[&>svg]:shrink-0
			`),
	)(
		html.Text(s),
	)
}

func sidebarGroupContent(items []MenuItem) html.Node {
	return html.Div(
		attr.Attr("data-slot")("sidebar-group-content"),
		attr.Attr("data-sidebar")("group-content"),
		attr.Class("w-full text-sm"),
	)(
		sidebarMenu(items),
	)
}

func sidebarMenu(items []MenuItem) html.Node {
	return html.Ul(
		attr.Attr("data-slot")("sidebar-menu"),
		attr.Attr("data-sidebar")("menu"),
		attr.Class("flex w-full min-w-0 flex-col gap-1 gap-0"),
	)(
		html.Range(items, sidebarMenuItem),
	)
}

func sidebarMenuItem(it MenuItem) html.Node {
	return html.Li(
		attr.Attr("data-slot")("sidebar-menu-item"),
		attr.Attr("data-sidebar")("menu-item"),
		attr.Class("group/menu-item relative"),
	)(
		sidebarMenuButton(it),
	)
}

func sidebarMenuButton(it MenuItem) html.Node {
	return html.A(
		attr.Class(`
			peer/menu-button
			outline-hidden
			ring-accent-6
			flex
			w-full
			items-center
			gap-2
			overflow-hidden
			rounded-md
			p-2
			text-start
			text-sm
			focus-visible:ring-2
			disabled:pointer-events-none
			disabled:opacity-50
			aria-disabled:pointer-events-none
			aria-disabled:opacity-50
			[&>span:last-child]:truncate
			[&>svg]:size-4
			[&>svg]:shrink-0
			h-8
			text-sm
			highlight-link
			data-selected:bg-gray-5
			data-selected:text-accent-10
		`),
		it.Attr,
		attr.Href(it.Path),
		stimulus.Target("sidebar", "link"),
		attr.Attr("data-slot")("sidebar-menu-button"),
		attr.Attr("data-sidebar")("menu-button"),
		attr.Attr("data-size")("default"),
	)(
		Icon(it.Icon),
		html.Span()(html.Text(it.Label)),
	)
}
