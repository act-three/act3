package web

import (
	"net/http"

	"ily.dev/act3/view"
)

func (w *web) accountProfile(req *http.Request) (http.Handler, error) {
	return page(view.EditAccountProfile()), nil
}

func (w *web) accountSecurity(req *http.Request) (http.Handler, error) {
	return page(view.EditAccountSecurity()), nil
}
