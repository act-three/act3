package static

import (
	"embed"

	"ily.dev/act3/http/digest"
)

//go:embed static
var staticFS embed.FS

var FS = newFS()

func newFS() *digest.Handler {
	fs, err := digest.New(staticFS)
	if err != nil {
		panic(err)
	}
	return fs
}
