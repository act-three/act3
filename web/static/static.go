package static

import (
	"embed"
	"net/http"

	"ily.dev/act3/http/digest"
)

//go:embed static
var staticFS embed.FS

var fs, err = digest.New(staticFS)

func init() {
	if err != nil {
		panic(err)
	}
}

func Handler() http.Handler {
	return http.StripPrefix("/-", fs)
}

func Path(name string) string {
	return "/-" + fs.NameToDigest(name)
}
