package base

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/web/static"
	"ily.dev/act3/web/web"
)

var (
	styleBundleURL  = static.FS.NameToDigest("/static/bundle.css")
	scriptBundleURL = static.FS.NameToDigest("/static/bundle.js")
)

func Base(title string, head ...html.Node) func(body ...html.Node) http.Handler {
	return func(body ...html.Node) http.Handler {
		return web.Page(
			html.Doctype,
			html.Html(
				attr.Class("group/active-url"),
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
					html.Group(head...),
				),
				html.Body()(body...),
			),
		)
	}
}
