package sidebar

import (
	"fmt"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

type Config struct {
	TaskCount      int
	TaskCountError int
}

type MenuSection struct {
	Label string
	Items []MenuItem
}

const TaskStatsID = "sidebar-task-stats"

type MenuItem struct {
	Icon    string
	Path    string
	Label   string
	Value   string // shown in muted text on right
	Badge   string // shown in bright capsule on right
	Attr    attr.Node
	StatsID string // if set, wraps value/badge in a stream target
}

func sidebarData(c Config) []MenuSection {
	return []MenuSection{
		{
			Label: "Account",
			Items: []MenuItem{
				{"line/user-circle", "/app/profile", "Profile", "", "", nil, ""},
				{"line/fingerprint-04", "/app/security", "Security", "", "", nil, ""},
			},
		},
		{
			Label: "Edit Media",
			Items: []MenuItem{
				{"line/arrow-circle-down", "/app/downloads", "Downloads", "", "", nil, ""},
				{"line/file-question-03", "/app/missing", "Missing Media", "", "", nil, ""},
				{"line/trash-01", "/app/trash", "Trash", "", "", nil, ""},
				{"line/layers-three-01", "/app/collections", "Collections", "", "", nil, ""},
				{"line/film-01", "/app/movies", "All Movies", "", "", nil, ""},
				{"line/tv-03", "/app/series", "All Series", "", "", nil, ""},
			},
		},
		{
			Label: "System",
			Items: []MenuItem{
				{"line/cloud-01", "/app/transmission", "Download Client", "", "", nil, ""},
				{"line/database-01", "/app/tmdb", "TMDB", "", "", nil, ""},
				{"line/hard-drive", "/app/storage", "Storage", "", "", nil, ""},
				{"line/calendar-check-01", "/app/tasks", "Tasks", numeric(c.TaskCount), numeric(c.TaskCountError), nil, TaskStatsID},
			},
		},
	}
}

func Sidebar(c Config) html.Node {
	return html.Div(
		Class("v-sidebar"),
		Attr("data-state")(""),
		Attr("data-collapsible")(""),
		Attr("data-variant")("inset"),
		Attr("data-side")("left"),
		Attr("data-slot")("sidebar"),
		turbo.DataFrame("main"),
		stimulus.Controller("sidebar"),
		stimulus.Action("turbo:visit@document->sidebar#visit"),
	)(
		html.Div(
			Attr("data-slot")("sidebar-gap"),
			Class("v-sidebar-gap"),
		),
		html.Div(
			Attr("data-slot")("sidebar-container"),
			Class("v-sidebar-container"),
		)(
			sidebarContent(c),
		),
	)
}

func sidebarContent(c Config) html.Node {
	return html.Div(
		Attr("data-slot")("sidebar-content"),
		Attr("data-sidebar")("content"),
		Class("v-sidebar-content"),
	)(
		html.Div(Class("v-sidebar-heading"))(
			Link("/")(Box(Class("v-wordmark"))),
		),
		html.Range(sidebarData(c), sidebarGroup),
	)
}

func sidebarGroup(v MenuSection) html.Node {
	return html.Div(
		Attr("data-slot")("sidebar-group"),
		Attr("data-sidebar")("group"),
		Class("v-sidebar-group"),
	)(
		html.If(v.Label != "", func() html.Node {
			return sidebarGroupLabel(v.Label)
		}),
		sidebarGroupContent(v.Items),
	)
}

func sidebarGroupLabel(s string) html.Node {
	return html.Div(
		Attr("data-slot")("sidebar-group-label"),
		Attr("data-sidebar")("group-label"),
		Class("v-sidebar-group-label"),
	)(
		html.Text(s),
	)
}

func sidebarGroupContent(items []MenuItem) html.Node {
	return sidebarMenu(items)
}

func sidebarMenu(items []MenuItem) html.Node {
	return html.Ul(
		Attr("data-slot")("sidebar-menu"),
		Attr("data-sidebar")("menu"),
		Class("v-sidebar-menu"),
	)(
		html.Range(items, sidebarMenuItem),
	)
}

func sidebarMenuItem(it MenuItem) html.Node {
	return html.Li(
		Attr("data-slot")("sidebar-menu-item"),
		Attr("data-sidebar")("menu-item"),
		Class("v-sidebar-menu-item"),
	)(
		sidebarMenuButton(it),
	)
}

func sidebarMenuButton(it MenuItem) html.Node {
	return html.A(
		Class("v-sidebar-menu-button"),
		it.Attr,
		Href(it.Path),
		stimulus.Target("sidebar", "link"),
		Attr("data-slot")("sidebar-menu-button"),
		Attr("data-sidebar")("menu-button"),
		Attr("data-size")("default"),
	)(
		Label(it.Icon, it.Label),
		menuStats(it),
	)
}

func menuStats(it MenuItem) html.Node {
	inner := menuStatsInner(it.Value, it.Badge)
	if it.StatsID != "" {
		return turbo.StreamTarget(it.StatsID)(inner)
	}
	return inner
}

func menuStatsInner(value, badge string) html.Node {
	return FlexRow(Gap2)(
		html.If(value != "", func() html.Node {
			return html.Div(Class("v-sidebar-menu-value"))(html.Text(value))
		}),
		html.If(badge != "", func() html.Node {
			return html.Div(Class("v-sidebar-menu-badge"))(html.Text(badge))
		}),
	)
}

func TaskStats(count, countError int) html.Node {
	return turbo.Update(TaskStatsID)(
		menuStatsInner(numeric(count), numeric(countError)),
	)
}

func numeric(n int) string {
	if n > 0 {
		return fmt.Sprintf("%d", n)
	}
	return ""
}
