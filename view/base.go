package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
	"ily.dev/act3/web/static"
)

var (
	styleBundleURL  = static.Path("/static/bundle.css")
	scriptBundleURL = static.Path("/static/bundle.js")
	PlyrIconURL     = static.Path("/static/plyr.svg")
)

func base(title string, head ...html.Node) func(...attr.Node) html.Element {
	return func(attrs ...attr.Node) html.Element {
		return func(body ...html.Node) html.Node {
			return Group(
				html.Doctype,
				html.Html(
					Class("group/active-url"),
				)(
					html.Head()(
						html.Meta(attr.Charset("utf-8")),
						html.Meta(attr.Name("viewport"), attr.Content("width=device-width,initial-scale=1")),
						html.Link(attr.Rel("preconnect"), attr.Href("https://fonts.googleapis.com")),
						html.Link(attr.Rel("preconnect"), attr.Href("https://fonts.gstatic.com"), attr.Crossorigin),
						html.Link(attr.Rel("stylesheet"), attr.Href("https://fonts.googleapis.com/css2?family=Atkinson+Hyperlegible+Next:ital,wght@0,200..800;1,200..800&display=swap")),
						html.Link(attr.Rel("stylesheet"), attr.Href(styleBundleURL)),
						html.Script(attr.Src(scriptBundleURL)),
						html.Title()(html.Text(title)),
						Group(head...),
					),
					html.Body(
						Class("dark bg-gray-0 text-gray-12"),
						group(attrs...),
					)(body...),
				),
			)
		}
	}
}
