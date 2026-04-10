// Package assets embeds shared static assets that need to be
// available at runtime to multiple packages. The same files are
// also copied into web/static/static at build time by go generate
// (see main.go) so they're served by the digest FS for direct use
// in HTML templates.
package assets

import _ "embed"

//go:embed poster-fallback.png
var PosterFallbackPNG []byte

//go:embed thumbnail-fallback.png
var ThumbnailFallbackPNG []byte

//go:embed banner-fallback.png
var BannerFallbackPNG []byte
