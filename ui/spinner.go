package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func Spinner(attrs ...domi.Attr) domi.Node {
	return html.Div(
		Class("u-spinner"),
		group(attrs...),
	)(spinnerLeaves...)
}

var spinnerLeaves = func() (a []domi.Node) {
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
