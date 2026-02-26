package ui

import (
	"fmt"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Textf(format string, arg ...any) html.Node {
	return Text(fmt.Sprintf(format, arg...))
}

func Text(s string, attrs ...attr.Node) html.Node {
	return html.Div(
		attr.EnvAttr("class", textSelectKey, "a$text+select-none"),
		attr.EnvAttr("class", lineClampKey, ""),
		attr.EnvAttr("class", fontWeightKey, ""),
		group(attrs...),
	)(
		html.Text(s),
	)
}

var (
	FontNormal = html.WithValue(fontWeightKey, "a$text+weight-normal")
	FontMedium = html.WithValue(fontWeightKey, "a$text+weight-medium")
	FontBold   = html.WithValue(fontWeightKey, "a$text+weight-bold")
)

var (
	LineClamp1 = html.WithValue(lineClampKey, "a$text+clamp-1")
	LineClamp2 = html.WithValue(lineClampKey, "a$text+clamp-2")
	LineClamp3 = html.WithValue(lineClampKey, "a$text+clamp-3")
	LineClamp4 = html.WithValue(lineClampKey, "a$text+clamp-4")
	LineClamp5 = html.WithValue(lineClampKey, "a$text+clamp-5")
)

var (
	TextSelectAuto = html.WithValue(textSelectKey, "a$text+select-auto")
	TextSelectNone = html.WithValue(textSelectKey, "a$text+select-none")
)
