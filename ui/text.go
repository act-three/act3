package ui

import (
	"fmt"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Textf(format string, arg ...any) html.Node {
	return Text(fmt.Sprintf(format, arg...))
}

func Text(s string) html.Node {
	return html.Div(
		attr.EnvAttr("class", lineClampKey, ""),
		attr.EnvAttr("class", fontWeightKey, ""),
	)(
		html.Text(s),
	)
}

var (
	FontNormal = html.WithValue(fontWeightKey, "font-normal")
	FontMedium = html.WithValue(fontWeightKey, "font-medium")
	FontBold   = html.WithValue(fontWeightKey, "font-bold")
)

var (
	LineClamp1 = html.WithValue(lineClampKey, "line-clamp-1")
	LineClamp2 = html.WithValue(lineClampKey, "line-clamp-2")
	LineClamp3 = html.WithValue(lineClampKey, "line-clamp-3")
	LineClamp4 = html.WithValue(lineClampKey, "line-clamp-4")
	LineClamp5 = html.WithValue(lineClampKey, "line-clamp-5")
)
