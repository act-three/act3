package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Spinner(attrs ...attr.Node) html.Node {
	return html.Div(
		Class("u-spinner"),
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
	SpinnerSM = Attr("data-spinner-size")("sm")
	SpinnerMD = Attr("data-spinner-size")("md")
	SpinnerLG = Attr("data-spinner-size")("lg")
)
