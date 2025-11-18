package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func (w *web) home(req *http.Request) (http.Handler, error) {
	return media("Act Three",
		html.Div()(html.A(attr.Href("/movies"))(html.Text("All Movies"))),
		html.Div()(html.A(attr.Href("/series"))(html.Text("All Series"))),
	), nil
}
