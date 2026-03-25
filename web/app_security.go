package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/view"
)

func (c *Config) appSecurity(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return view.AppSecurity(), nil
}
