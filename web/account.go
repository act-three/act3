package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/web/app"
)

func (w *web) accountProfile(req *http.Request) (http.Handler, error) {
	return app.Page("Profile",
		html.Div()(html.Text("Change Name")),
	), nil
}

func (w *web) accountSecurity(req *http.Request) (http.Handler, error) {
	return app.Page("Security",
		html.Div()(html.Text("Change Password")),
	), nil
}
