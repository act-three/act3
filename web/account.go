package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/view"
)

func (c *Config) accountProfile(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return view.EditAccountProfile(), nil
}

func (c *Config) accountSecurity(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return view.EditAccountSecurity(), nil
}
