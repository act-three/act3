package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"ily.dev/domi"
)

// stubApp is an App whose views are supplied by the test.
type stubApp struct {
	view    View
	preview func(*url.URL) (string, View)
}

func (a *stubApp) Update(context.Context, struct{}) domi.Cmd[struct{}] { return nil }
func (a *stubApp) View(context.Context) View                           { return a.view }
func (a *stubApp) Subscriptions(context.Context) domi.Sub[struct{}]    { return nil }
func (a *stubApp) Preview(_ context.Context, u *url.URL) (string, View) {
	if a.preview == nil {
		return "", nil
	}
	return a.preview(u)
}

func renderNode(t *testing.T, n domi.Node) string {
	t.Helper()
	var sb strings.Builder
	if err := domi.RenderTo(&sb, n); err != nil {
		t.Fatalf("render: %v", err)
	}
	return sb.String()
}

// TestInstanceAccumulatesRules verifies that rules remain in the stylesheet after their views disappear.
// It also verifies that those rules are reused when the views return.
func TestInstanceAccumulatesRules(t *testing.T) {
	ctx := context.Background()
	app := &stubApp{view: Text("a").Padding(Edges(16))}
	in := &instance[struct{}, *stubApp]{app: app}
	view := func() string {
		_, n := in.View(ctx)
		return renderNode(t, n)
	}
	first := view()
	if !strings.Contains(first, "padding-block-start:16px") {
		t.Fatalf("first render missing its rule:\n%s", first)
	}
	app.view = Text("a")
	gone := view()
	if !strings.Contains(gone, "padding-block-start:16px") {
		t.Errorf("rule dropped when its view went away:\n%s", gone)
	}
	app.view = Text("a").Padding(Edges(16))
	back := view()
	if back != first {
		t.Errorf("revisit is not byte-identical to the first render:\n%s\nvs:\n%s", back, first)
	}
}

// TestInstancePreview verifies that a preview is rendered only when the App
// offers one, and that its rules join the instance's stylesheet.
func TestInstancePreview(t *testing.T) {
	ctx := context.Background()
	app := &stubApp{view: Text("a")}
	in := &instance[struct{}, *stubApp]{app: app}
	u := &url.URL{Path: "/x"}

	if dest, _, n := in.Preview(ctx, u); dest != "" || n != nil {
		t.Errorf("declined preview rendered: dest=%q n=%v", dest, n)
	}

	app.preview = func(*url.URL) (string, View) {
		return "/x", Text("b").Title("x").Padding(Edges(16))
	}
	dest, title, n := in.Preview(ctx, u)
	if dest != "/x" || title != "x" {
		t.Errorf("preview = %q, %q; want /x, x", dest, title)
	}
	if got := renderNode(t, n); !strings.Contains(got, "padding-block-start:16px") {
		t.Errorf("preview missing its rule:\n%s", got)
	}
	_, n = in.View(ctx)
	if got := renderNode(t, n); !strings.Contains(got, "padding-block-start:16px") {
		t.Errorf("preview's rule absent from the next view:\n%s", got)
	}
}

// TestHandlerNonce verifies that the nonce reaches the style element of a served page.
func TestHandlerNonce(t *testing.T) {
	h := Handler(
		func(context.Context, *url.URL) (*stubApp, domi.Cmd[struct{}]) {
			return &stubApp{view: Image("/x.png")}, nil
		},
		func(*url.URL) struct{} { return struct{}{} },
		func(*url.URL) struct{} { return struct{}{} },
		StyleNonce(func(context.Context) string { return "abc123" }),
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 {
		t.Fatalf("status = %d, body:\n%s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `<style nonce="abc123">`) {
		t.Errorf("style element does not carry the nonce:\n%s", rec.Body)
	}
}

// TestHandlerStylesheet verifies that the page of a default-document app
// loads the static stylesheet from the Handler under the internal URL prefix,
// and that an app with a custom document is left to link it itself.
func TestHandlerStylesheet(t *testing.T) {
	newApp := func(context.Context, *url.URL) (*stubApp, domi.Cmd[struct{}]) {
		return &stubApp{view: Text("a")}, nil
	}
	onURL := func(*url.URL) struct{} { return struct{}{} }
	onChange := func(*url.URL) struct{} { return struct{}{} }
	get := func(t *testing.T, h http.Handler, p string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		if rec.Code != 200 {
			t.Fatalf("GET %s: status = %d, body:\n%s", p, rec.Code, rec.Body)
		}
		return rec
	}

	t.Run("default", func(t *testing.T) {
		h := Handler(newApp, onURL, onChange, domi.InternalURLPrefix("/-/x"))
		body := get(t, h, "/").Body.String()
		m := regexp.MustCompile(`<ui-root><link href="([^"]+)" rel="stylesheet">`).FindStringSubmatch(body)
		if m == nil {
			t.Fatalf("no stylesheet link in ui-root:\n%s", body)
		}
		cssPath := m[1]
		if !strings.HasPrefix(cssPath, "/-/x/ui.") || !strings.HasSuffix(cssPath, ".css") {
			t.Errorf("stylesheet path = %q", cssPath)
		}
		if rec := get(t, h, cssPath); rec.Body.String() != string(staticCSS) {
			t.Errorf("GET %s served something other than the stylesheet:\n%s", cssPath, rec.Body)
		} else if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
			t.Errorf("GET %s: Cache-Control = %q", cssPath, cc)
		}
	})

	t.Run("custom", func(t *testing.T) {
		h := Handler(newApp, onURL, onChange, domi.Document(func(title string, body domi.Node) domi.Node {
			return domi.Tag("html")(domi.Tag("head")(domi.Tag("title")(domi.Text("custom "+title))), body)
		}))
		body := get(t, h, "/").Body.String()
		if !strings.Contains(body, "<title>custom </title>") {
			t.Errorf("custom document not used:\n%s", body)
		}
		if strings.Contains(body, "stylesheet") {
			t.Errorf("stylesheet link emitted despite custom document:\n%s", body)
		}
	})
}

// TestStylesheet verifies that Stylesheet serves CSS.
func TestStylesheet(t *testing.T) {
	digest, h := Stylesheet()
	if digest == "" {
		t.Error("empty digest")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/xui."+digest+".css", nil))
	if rec.Code != 200 || rec.Body.String() != string(staticCSS) {
		t.Errorf("status = %d, body:\n%s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Errorf("Content-Type = %q", ct)
	}
}
