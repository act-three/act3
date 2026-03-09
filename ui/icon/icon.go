package icon

import (
	"crypto/sha3"
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"ily.dev/act3/encoding/base32c"
	"ily.dev/act3/http/digest"
)

//go:embed svg
var svgFS embed.FS

var subFS, _ = fs.Sub(svgFS, "svg")

var handler, _ = digest.New(subFS)

// Handler returns an http.Handler serving icon SVGs at
// digest-based paths.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.URL.Path, "/") == missingPath[1:] {
			w.Header().Set("Cache-Control", "max-age=31536000, immutable")
			w.Header().Set("Content-Type", "image/svg+xml")
			io.WriteString(w, missingSVG)
			return
		}
		if handler != nil {
			handler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

// Path returns the digest-based URL path for the named
// icon. Returns the missing icon path if not found.
func Path(name string) string {
	if handler != nil {
		if p := handler.NameToDigest(name + ".svg"); p != "" {
			return p
		}
	}
	return missingPath
}

// SVG returns the icon as an inline html.Node.
// Returns the missing icon if not found.
func SVG(name string) string {
	b, err := fs.ReadFile(subFS, name+".svg")
	if err != nil {
		return missingSVG
	}
	return string(b)
}

var missingPath = func() string {
	sum := sha3.Sum256([]byte(missingSVG))
	d := strings.ToLower(base32c.EncodeToString(sum[:])[:6])
	return "/missing." + d + ".svg"
}()

// missingSVG is the square-dashed icon from Lucide (MIT license),
// used as a fallback when the requested icon is not found.
const missingSVG = `<svg class="u-icon" viewBox="0 0 24 24">` +
	`<path d="M5 3a2 2 0 0 0-2 2"/>` +
	`<path d="M19 3a2 2 0 0 1 2 2"/>` +
	`<path d="M21 19a2 2 0 0 1-2 2"/>` +
	`<path d="M5 21a2 2 0 0 1-2-2"/>` +
	`<path d="M9 3h1"/>` +
	`<path d="M9 21h1"/>` +
	`<path d="M14 3h1"/>` +
	`<path d="M14 21h1"/>` +
	`<path d="M3 9v1"/>` +
	`<path d="M21 9v1"/>` +
	`<path d="M3 14v1"/>` +
	`<path d="M21 14v1"/>` +
	`</svg>`
