package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Spinner() html.Node {
	return html.Div(
		attr.Class("a$spinner"),
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
	spinnerSM = "a$spinner+size-sm"
	spinnerMD = "a$spinner+size-md"
	spinnerLG = "a$spinner+size-lg"
)
