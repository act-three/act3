package web

import (
	"fmt"
	"net/http"

	"ily.dev/act3/html"
)

func (c *Config) status(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "tag: %s\n", tag)
	return nil, nil
}
