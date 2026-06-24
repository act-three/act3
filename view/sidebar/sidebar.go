package sidebar

import (
	"fmt"
	"strings"

	"ily.dev/domi"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

type Config struct {
	Path           string // current URL path, used to highlight the active item
	TaskCount      int
	TaskCountError int
	Uploads        []model.Upload
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
	Attr    domi.Attr
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
				{"line/info-circle", "/app/about", "About", "", "", nil, ""},
				{"line/cloud-01", "/app/transmission", "Download Client", "", "", nil, ""},
				{"line/database-01", "/app/tmdb", "TMDB", "", "", nil, ""},
				{"line/hard-drive", "/app/storage", "Storage", "", "", nil, ""},
				{"line/calendar-check-01", "/app/tasks", "Tasks", numeric(c.TaskCount), numeric(c.TaskCountError), nil, TaskStatsID},
			},
		},
	}
}

func Sidebar(c Config) domi.Node {
	return html.Div(
		Class("v-sidebar"),
		Attr("data-state")(""),
		Attr("data-collapsible")(""),
		Attr("data-variant")("inset"),
		Attr("data-side")("left"),
		Attr("data-slot")("sidebar"),
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

func sidebarContent(c Config) domi.Node {
	return html.Div(
		Attr("data-slot")("sidebar-content"),
		Attr("data-sidebar")("content"),
		Class("v-sidebar-content"),
	)(
		html.Div(Class("v-sidebar-heading"))(
			Link("/")(
				FlexRow(Style("align-items:center;gap:0.5rem"))(
					Box(Class("v-wordmark")),
					Box()(domi.Safe("&beta;")),
				),
			),
		),
		rangeNodes(sidebarData(c), func(v MenuSection) domi.Node {
			return sidebarGroup(v, c.Path)
		}),
		sidebarUploadProgress(c.Uploads),
	)
}

func sidebarGroup(v MenuSection, current string) domi.Node {
	return html.Div(
		Attr("data-slot")("sidebar-group"),
		Attr("data-sidebar")("group"),
		Class("v-sidebar-group"),
	)(
		iff(v.Label != "", func() domi.Node {
			return sidebarGroupLabel(v.Label)
		}),

		sidebarGroupContent(v.Items, current),
	)
}

func sidebarGroupLabel(s string) domi.Node {
	return html.Div(
		Attr("data-slot")("sidebar-group-label"),
		Attr("data-sidebar")("group-label"),
		Class("v-sidebar-group-label"),
	)(
		domi.Text(s),
	)
}

func sidebarGroupContent(items []MenuItem, current string) domi.Node {
	return sidebarMenu(items, current)
}

func sidebarMenu(items []MenuItem, current string) domi.Node {
	return html.UL(
		Attr("data-slot")("sidebar-menu"),
		Attr("data-sidebar")("menu"),
		Class("v-sidebar-menu"),
	)(
		rangeNodes(items, func(it MenuItem) domi.Node {
			return sidebarMenuItem(it, current)
		}),
	)
}

func sidebarMenuItem(it MenuItem, current string) domi.Node {
	return html.LI(
		Attr("data-slot")("sidebar-menu-item"),
		Attr("data-sidebar")("menu-item"),
		Class("v-sidebar-menu-item"),
	)(
		sidebarMenuButton(it, current),
	)
}

func sidebarMenuButton(it MenuItem, current string) domi.Node {
	return html.A(
		Class("v-sidebar-menu-button"),
		it.Attr,
		Href(it.Path),
		domi.Bool("data-selected")(isActive(it.Path, current)),
		Attr("data-slot")("sidebar-menu-button"),
		Attr("data-sidebar")("menu-button"),
		Attr("data-size")("default"),
	)(
		Label(it.Icon, it.Label),
		menuStats(it),
	)
}

// isActive reports whether the menu item at itemPath should be highlighted
// for the current path: when the current path is the item's path or sits
// beneath it, so /app/series stays lit on /app/series/spice-and-wolf.
func isActive(itemPath, current string) bool {
	return current == itemPath || strings.HasPrefix(current, itemPath+"/")
}

func menuStats(it MenuItem) domi.Node {
	return FlexRow(Gap2)(
		iff(it.Value != "", func() domi.Node {
			return html.Div(Class("v-sidebar-menu-value"))(domi.Text(it.Value))
		}),

		iff(it.Badge != "", func() domi.Node {
			return html.Div(Class("v-sidebar-menu-badge"))(domi.Text(it.Badge))
		}),
	)
}

// sidebarUploadProgress shows the oldest in-flight upload's name
// and progress at the bottom of the sidebar.
func sidebarUploadProgress(uploads []model.Upload) domi.Node {
	if len(uploads) == 0 {
		return nil
	}
	u := uploads[0]
	return html.Div(
		Class("v-sidebar-upload-progress"),
	)(
		html.Div(Class("v-sidebar-upload-label"))(domi.Text(u.Name)),
		Progress(u.Frac),
	)
}

func numeric(n int) string {
	if n > 0 {
		return fmt.Sprintf("%d", n)
	}
	return ""
}
