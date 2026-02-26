package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Spinner() html.Node {
	return html.Div(
		attr.Class("u-spinner"),
		attr.EnvAttr("class", spinnerSizeKey, spinnerMD),
	)(spinnerLeaves...)
}

var spinnerLeaves = func() (a []html.Node) {
	for range 8 {
		a = append(a, html.Span())
	}
	return a
}()

var (
	SpinnerSM = html.WithValue(spinnerSizeKey, spinnerSM)
	SpinnerMD = html.WithValue(spinnerSizeKey, spinnerMD) // default
	SpinnerLG = html.WithValue(spinnerSizeKey, spinnerLG)
)

const (
	spinnerSM = "u-spinner+size-sm"
	spinnerMD = "u-spinner+size-md"
	spinnerLG = "u-spinner+size-lg"
)
