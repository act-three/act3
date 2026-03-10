package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Spinner(attrs ...attr.Node) html.Node {
	return html.Div(
		attr.Class("u-spinner"),
		group(attrs...),
	)(spinnerLeaves...)
}

var spinnerLeaves = func() (a []html.Node) {
	for range 8 {
		a = append(a, html.Span())
	}
	return a
}()

var (
	SpinnerSM = attr.Attr("data-size")("sm")
	SpinnerMD = attr.Attr("data-size")("md")
	SpinnerLG = attr.Attr("data-size")("lg")
)
