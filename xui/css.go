package ui

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"net/http"
	"time"
)

// staticCSS is the static stylesheet.
//
//go:embed ui.css
var staticCSS []byte

var cssDigest = func() string {
	h := sha256.Sum256(staticCSS)
	return hex.EncodeToString(h[:3])
}()

// Stylesheet provides access
// to the static stylesheet asset.
//
// The returned handler serves the stylesheet
// at a URL path chosen by the app.
//
// The returned digest is a content hash of the stylesheet.
// Include it in the request path
// to ensure the browser never uses
// a stylesheet from the wrong version of this package.
//
//	digest, handler := ui.Stylesheet()
//	path := "/ui." + digest + ".css"
//	mux := &http.ServeMux{}
//	mux.Handle(path, handler)
//
// The stylesheet can then be linked in the document head.
// See [domi.Document].
//
//	html.Link(attr.Rel("stylesheet"), attr.Href(path))
func Stylesheet() (digest string, h http.Handler) {
	return cssDigest, http.HandlerFunc(serveCSS)
}

func serveCSS(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=31536000, immutable")
	http.ServeContent(w, req, "ui.css", time.Time{}, bytes.NewReader(staticCSS))
}
