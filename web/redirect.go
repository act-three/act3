package web

import (
	"path"
	"strings"
)

var redirects = map[string]string{
	"/app": "/app/profile",
}

func redirect(req string) string {
	if p := redirects[strings.TrimRight(path.Clean(req), "/")]; p != "" {
		return p
	}
	return req
}
