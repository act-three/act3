package hi

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"golang.org/x/net/html"
	"ily.dev/domi"
	"ily.dev/domi/attr"
)

//go:embed hi.js
var rawClientJS []byte

var clientJS = fmt.Sprintf("import %q;\n%s\nrun();\n",
	"./"+domiClientModueName,
	rawClientJS,
)

var clientJSDigest = func() string {
	h := sha256.Sum256([]byte(clientJS))
	return hex.EncodeToString(h[:3])
}()

// ClientModule returns an HTML fragment
// that loads the JavaScript modules used by Hi and Domi.
// The returned Node can be placed into the head element
// of the application's custom document, if any.
// See [domi.Document].
//
// The given prefix must match the value
// used in [domi.InternalURLPrefix].
//
// Applications that do not provide a custom document
// do not need to use ClientModule.
// Applications that bundle the Hi and Domi modules
// with additional JavaScript do not need to use ClientModule.
// See the “Serving Client Assets” section in the package documentation for details.
func ClientModule(prefix string) domi.Node {
	return domi.Fragment(
		domi.ClientModule(prefix),
		domi.Tag("script", attr.Type("module"), attr.Src(clientJSPath(prefix))),
	)
}

func clientJSPath(prefix string) string {
	return path.Join("/", prefix, "hi."+clientJSDigest+".js")
}

func clientJSHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=31536000, immutable")
	http.ServeContent(w, req, "hi.js", time.Time{}, strings.NewReader(clientJS))
}

// Parse the filename out of Domi's script tag.
var domiClientModueName = func() string {
	var buf bytes.Buffer
	if err := domi.RenderTo(&buf, domi.ClientModule("")); err != nil {
		panic(err)
	}
	z := html.NewTokenizer(&buf)
	for tt := z.Next(); tt != html.ErrorToken; tt = z.Next() {
		if tt != html.StartTagToken {
			continue
		}
		token := z.Token()
		if token.Data != "script" {
			continue
		}
		for _, a := range token.Attr {
			if a.Key == "src" && a.Val != "" {
				return path.Base(a.Val)
			}
		}
	}
	panic(fmt.Errorf("hi: domi.ClientModule returned no script source"))
}()
