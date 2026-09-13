package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"ily.dev/act3/hi"
	"ily.dev/act3/ui"
	"ily.dev/domi"
)

func TestHiRoot(t *testing.T) {
	mux := http.NewServeMux()
	Handle(mux, &Config{Model: newTestModel(t)})
	for _, tt := range []struct {
		path, title, content string
	}{
		{"/", "Act Three", `data-controller="home"`},
		{"/app/profile", "Profile — Act Three", "Change Name"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			r := httptest.NewRecorder()
			mux.ServeHTTP(r, httptest.NewRequest("GET", tt.path, nil))
			if r.Code != http.StatusOK {
				t.Fatalf("status = %d", r.Code)
			}
			for _, want := range []string{
				"<title>" + tt.title + "</title>",
				"<hi-root", `scroll="y"`, "<hi-html", "v-domi-root",
				"@layer hi{", tt.content, `id="player"`, `id="note-port"`,
			} {
				if !strings.Contains(r.Body.String(), want) {
					t.Errorf("response missing %q", want)
				}
			}
		})
	}
}

func TestPreviewPreservesState(t *testing.T) {
	a := newTestApp(t, newTestModel(t), "/app/about")
	a.notes = []ui.Note{{ID: "note-1", Title: "Preview must not replay this"}}
	u := &url.URL{Path: "/app", RawQuery: "q=kept", Fragment: "section"}
	var dest, title, body string
	a.Preview(t.Context(), u, func(d string, v hi.View) hi.Preview {
		dest = d
		var n domi.Node
		title, n = hi.Render(v)
		var b strings.Builder
		if err := domi.RenderTo(&b, n); err != nil {
			t.Fatal(err)
		}
		body = b.String()
		return hi.Preview{}
	})
	if dest != "/app/profile?q=kept#section" || title != "Profile — Act Three" {
		t.Fatalf("preview = %q, %q", dest, title)
	}
	if !strings.Contains(body, "Change Name") || strings.Contains(body, a.notes[0].Title) {
		t.Error("preview must render the destination without replaying notifications")
	}
	if a.path != "/app/about" || len(a.notes) != 1 || u.Path != "/app" {
		t.Error("preview changed the live app or the requested URL")
	}
}

func TestRendererTransactionScope(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(map[bool]string{false: "View", true: "Preview"}[preview], func(t *testing.T) {
			m, db := newTestModelDB(t)
			a := newTestApp(t, m, "/app/profile")
			calls := 0
			render := func(v hi.View) {
				hi.Render(hi.Lazy(func() hi.View {
					calls++
					if got := db.Stats().InUse; got != 1 {
						t.Errorf("connections in use during lazy rendering = %d, want 1", got)
					}
					return v
				}))
			}
			if preview {
				a.Preview(t.Context(), &url.URL{Path: "/app/about"}, func(_ string, v hi.View) hi.Preview {
					render(v)
					return hi.Preview{}
				})
			} else {
				a.View(t.Context(), func(v hi.View) hi.Page {
					render(v)
					return hi.Page{}
				})
			}
			if calls != 1 {
				t.Errorf("lazy render calls = %d, want 1", calls)
			}
			if got := db.Stats().InUse; got != 0 {
				t.Errorf("connections in use after rendering = %d, want 0", got)
			}
		})
	}
}

func renderApp(t *testing.T, a *app) (title string, n domi.Node) {
	t.Helper()
	a.View(t.Context(), func(v hi.View) hi.Page {
		title, n = hi.Render(v)
		return hi.Page{}
	})
	return title, n
}
