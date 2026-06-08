package web

import (
	"fmt"
	"net/http"
)

func (c *Config) status(w http.ResponseWriter, req *http.Request) (node, error) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "tag: %s\n", tag)
	return nil, nil
}
