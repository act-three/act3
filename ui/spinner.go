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
	SpinnerSM = attr.Class("u-spinner+size-sm")
	SpinnerMD = attr.Class("u-spinner+size-md")
	SpinnerLG = attr.Class("u-spinner+size-lg")
)
