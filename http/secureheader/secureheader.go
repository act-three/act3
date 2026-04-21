// Package secureheader adds conservative security-related response
// headers to every HTTP response.
//
// The middleware sets headers before invoking the wrapped handler,
// so handlers may override any of them (e.g. a route that serves
// static blobs can replace the default CSP with a tighter one).
package secureheader

import "net/http"

// DefaultCSP is the default Content-Security-Policy applied to every
// response. It is sized to fit the HTML pages rendered by this app
// (inline style attributes, Google Fonts, same-origin scripts and
// media) while remaining restrictive enough to block common XSS and
// exfiltration paths. Routes that serve non-HTML blobs (video,
// downloads, images) override this with something tighter.
const DefaultCSP = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"img-src 'self' data: https://image.tmdb.org https://static.tvmaze.com; " +
	"media-src 'self'; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"form-action 'self'; " +
	"base-uri 'self'"

// Handler wraps h with a middleware that sets a fixed set of
// security headers on every response. Handlers may override any
// individual header by calling w.Header().Set before the response
// is flushed.
func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hdr := w.Header()
		hdr.Set("X-Content-Type-Options", "nosniff")
		hdr.Set("Referrer-Policy", "no-referrer")
		hdr.Set("X-Frame-Options", "DENY")
		hdr.Set("Content-Security-Policy", DefaultCSP)
		h.ServeHTTP(w, req)
	})
}
