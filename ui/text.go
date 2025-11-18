package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	FontNormal = Class("font-normal")
	FontMedium = Class("font-medium")
	FontBold   = Class("font-bold")
)

func Text(s string, attrs ...attr.Node) html.Node {
	return html.Div(
		attr.FuncAttr("class", func(get func(any) any) string {
			s, _ := get(lineClampKey).(string)
			return s
		}),
		group(attrs...),
	)(
		html.Text(s),
	)
}

var (
	LineClamp1 = html.WithValue(lineClampKey, "line-clamp-1")
	LineClamp2 = html.WithValue(lineClampKey, "line-clamp-2")
	LineClamp3 = html.WithValue(lineClampKey, "line-clamp-3")
	LineClamp4 = html.WithValue(lineClampKey, "line-clamp-4")
	LineClamp5 = html.WithValue(lineClampKey, "line-clamp-5")
)
