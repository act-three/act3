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
				{"line/database-01", "/app/tmdb", "TMDB", nil},
				{"line/hard-drive", "/app/storage", "Storage", nil},
				{"line/calendar-check-01", "/app/tasks", "Tasks", nil},
			},
		},
	}
}

func Sidebar() html.Node {
	return html.Div(
		attr.Class("v-sidebar"),
		attr.Attr("data-state")(""),
		attr.Attr("data-collapsible")(""),
		attr.Attr("data-variant")("inset"),
		attr.Attr("data-side")("left"),
		attr.Attr("data-slot")("sidebar"),
		turbo.DataFrame("main"),
		stimulus.Controller("sidebar"),
		stimulus.Action("turbo:visit@document->sidebar#visit"),
	)(
		html.Div(
			attr.Attr("data-slot")("sidebar-gap"),
			attr.Class("v-sidebar-gap"),
		),
		html.Div(
			attr.Attr("data-slot")("sidebar-container"),
			attr.Class("v-sidebar-container"),
		)(
			sidebarContent(),
		),
	)
}

func sidebarContent() html.Node {
	return html.Div(
		attr.Attr("data-slot")("sidebar-content"),
		attr.Attr("data-sidebar")("content"),
		attr.Class("v-sidebar-content"),
	)(
		html.Range(sidebarData(), sidebarGroup),
	)
}

func sidebarGroup(v MenuSection) html.Node {
	return html.Div(
		attr.Attr("data-slot")("sidebar-group"),
		attr.Attr("data-sidebar")("group"),
		attr.Class("v-sidebar-group"),
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
		attr.Class("v-sidebar-group-label"),
	)(
		html.Text(s),
	)
}

func sidebarGroupContent(items []MenuItem) html.Node {
	return sidebarMenu(items)
}

func sidebarMenu(items []MenuItem) html.Node {
	return html.Ul(
		attr.Attr("data-slot")("sidebar-menu"),
		attr.Attr("data-sidebar")("menu"),
		attr.Class("v-sidebar-menu"),
	)(
		html.Range(items, sidebarMenuItem),
	)
}

func sidebarMenuItem(it MenuItem) html.Node {
	return html.Li(
		attr.Attr("data-slot")("sidebar-menu-item"),
		attr.Attr("data-sidebar")("menu-item"),
		attr.Class("v-sidebar-menu-item"),
	)(
		sidebarMenuButton(it),
	)
}

func sidebarMenuButton(it MenuItem) html.Node {
	return html.A(
		attr.Class("v-sidebar-menu-button"),
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
