// Package jassub serves the optional libass-via-WebAssembly
// subtitle renderer.
//
// Dir dist holds a single placeholder Readme in git; the real
// vendor artifacts (host module, worker, wasm, default font)
// are written there by gen.go and gitignored. With the bundle
// absent, Path("jassub.js") returns "" and the player view
// treats that as "feature off", staying on the WebVTT
// downconversion path for ASS sources. To enable jassub: run
//
//	go run web/jassub/gen.go
//
// once (network required) before building.
package jassub

import (
	"embed"
	"io/fs"
	"net/http"

	"ily.dev/act3/http/digest"
)

//go:embed dist
var distFS embed.FS

var jassubFS = func() *digest.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	h, err := digest.New(sub)
	if err != nil {
		panic(err)
	}
	return h
}()

// Handler serves the vendored jassub artifacts at digest-cache-
// busted paths under /-/jassub/.
func Handler() http.Handler {
	return http.StripPrefix("/-/jassub", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// jassub's emscripten runtime uses eval() / Function()
		// internally for emval bindings, so the worker needs
		// 'unsafe-eval' on its CSP. Workers respect their own
		// response CSP, so we override here without weakening
		// the document CSP set by the secureheader middleware.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-eval' 'wasm-unsafe-eval'; "+
				"connect-src 'self' blob:")
		jassubFS.ServeHTTP(w, req)
	}))
}

// Path returns /-/jassub/<digest-name> for name,
// or "" if name isn't found.
func Path(name string) string {
	d := jassubFS.NameToDigest(name)
	if d == "" {
		return ""
	}
	return "/-/jassub" + d
}
