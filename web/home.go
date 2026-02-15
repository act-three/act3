package web

import (
	"net/http"

	"ily.dev/act3/view"
)

func (w *web) home(req *http.Request) (http.Handler, error) {
	return page(view.Home()), nil
}
