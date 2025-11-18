package web

import (
	"bytes"
	"log/slog"
	"net/http"

	"ily.dev/act3/html"
)

func Page(node ...html.Node) http.Handler {
	return handler("text/html", 200, node...)
}

func PageBadRequest(node ...html.Node) http.Handler {
	return handler("text/html", 400, node...)
}

func Stream(node ...html.Node) http.Handler {
	return handler("text/vnd.turbo-stream.html", 200, node...)
}

func StreamBadRequest(node ...html.Node) http.Handler {
	return handler("text/vnd.turbo-stream.html", 400, node...)
}

func StreamNotFound(node ...html.Node) http.Handler {
	return handler("text/vnd.turbo-stream.html", 404, node...)
}

func StreamInternalError(node ...html.Node) http.Handler {
	return handler("text/vnd.turbo-stream.html", 500, node...)
}

func handler(contentType string, code int, node ...html.Node) http.Handler {
	buf := &bytes.Buffer{}
	err := html.Render(buf, html.Group(node...))
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(code)
		_, err := w.Write(buf.Bytes())
		if err != nil {
			slog.WarnContext(ctx, err.Error())
			return
		}
	})
}
