package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	. "ily.dev/act3/ui"
	"ily.dev/act3/web/static"
)

var (
	styleBundleURL  = static.Path("/static/bundle.css")
	scriptBundleURL = static.Path("/static/bundle.js")
)

func Document(title string, body domi.Node) domi.Node {
	return Group(
		html.HTML()(
			html.Head()(
				html.Meta(attr.Charset("utf-8")),
				html.Meta(attr.Name("viewport"), attr.Content("width=device-width,initial-scale=1")),
				html.Link(attr.Rel("preconnect"), Href("https://fonts.googleapis.com")),
				html.Link(attr.Rel("preconnect"), Href("https://fonts.gstatic.com"), attr.CrossOrigin("")),
				html.Link(attr.Rel("stylesheet"), Href("https://fonts.googleapis.com/css2?family=Atkinson+Hyperlegible+Next:ital,wght@0,200..800;1,200..800&display=swap")),
				html.Link(attr.Rel("icon"), attr.Type("image/svg+xml"), Href(logoURL)),
				html.Link(attr.Rel("stylesheet"), Href(styleBundleURL)),
				html.Script(attr.Type("module"), attr.Src(scriptBundleURL)),
				html.Title()(domi.Text(title)),
			),
			body,
		),
	)
}
