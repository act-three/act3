package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"ily.dev/act3/hi"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	"ily.dev/act3/ui"
	"ily.dev/domi"
)

func TestHiRoot(t *testing.T) {
	mux := http.NewServeMux()
	Handle(mux, &Config{Model: newTestModel(t)})
	for _, tt := range []struct {
		path, title, content string
		scroll               bool
	}{
		{"/", "Act Three", `data-controller="home"`, true},
		{"/collections", "Collections — Act Three", "Collections", true},
		{"/missing", "Not Found — Act Three", "Not Found", true},
		{"/app/profile", "Profile — Act Three", "Change Name", false},
		{"/app/profile/", "Act Three", "Not Found", false},
		{"/app/about", "About — Act Three", `class="v-about`, false},
		{"/app/security", "Security — Act Three", "Change Password", false},
		{"/app/tasks", "Tasks — Act Three", "Tasks", false},
		{"/app/tmdb", "TMDB — Act Three", "TMDB", false},
		{"/app/transmission", "Transmission — Act Three", "Transmission", false},
		{"/app/collections", "Collections — Act Three", "Collections", false},
		{"/app/movies", "All Movies — Act Three", "All Movies", false},
		{"/app/series", "Edit Series — Act Three", "Series", false},
		{"/app/downloads", "Downloads — Act Three", "No Download Selected", false},
		{"/app/downloads/missing", "Downloads — Act Three", "Not Found", false},
		{"/app/downloads/missing/extra", "Act Three", "Not Found", false},
		{"/app/trash", "Trash — Act Three", "No Item Selected", false},
		{"/app/trash/missing", "Trash — Act Three", "Not Found", false},
		{"/app/trash/missing/extra", "Act Three", "Not Found", false},
		{"/app/profile/extra", "Act Three", "Not Found", false},
		{"/app/movies-other", "Act Three", "Not Found", false},
		{"/app/movies/missing", "All Movies — Act Three", "Not Found", false},
		{"/app/series/missing", "Edit Series — Act Three", "Not Found", false},
		{"/app/collections/missing", "Collections — Act Three", "Not Found", false},
		{"/app/missing", "Act Three", "Not Found", false},
		{"/apple", "Not Found — Act Three", "Not Found", true},
		{"/application/profile", "Not Found — Act Three", "Not Found", true},
	} {
		t.Run(tt.path, func(t *testing.T) {
			r := httptest.NewRecorder()
			mux.ServeHTTP(r, httptest.NewRequest("GET", tt.path, nil))
			if r.Code != http.StatusOK {
				t.Fatalf("status = %d", r.Code)
			}
			for _, want := range []string{
				"<title>" + tt.title + "</title>",
				"<hi-root", "<hi-html", "v-domi-root",
				"@layer hi{", tt.content, `id="player"`, `id="note-port"`,
			} {
				if !strings.Contains(r.Body.String(), want) {
					t.Errorf("response missing %q", want)
				}
			}
			body := r.Body.String()
			if got := strings.Contains(body, `data-slot="sidebar-wrapper"`); got == tt.scroll {
				t.Errorf("editor shell = %v, want %v", got, !tt.scroll)
			}
			if got := strings.Contains(body, `scroll="y"`); got != tt.scroll {
				t.Errorf("document scrolling = %v, want %v", got, tt.scroll)
			}
			if strings.Contains(body, "<hi-scroll") {
				t.Error("page scrolling must use the document viewport")
			}
			for _, id := range []string{"player", "note-port"} {
				if count := strings.Count(body, `id="`+id+`"`); count != 1 {
					t.Errorf("%s containers = %d, want 1", id, count)
				}
			}
		})
	}
}

func TestEditorRoutesAreLazy(t *testing.T) {
	// A static page must not construct any of the database-backed pages.
	v := viewEditorPage(nil, []string{"app", "profile"}, nil)
	if title, _ := renderViewAt(t, "/app/profile", v); title != "Profile" {
		t.Fatalf("title = %q, want Profile", title)
	}
}

func TestPreviewResolvesObjectPath(t *testing.T) {
	m := newTestModel(t)
	if err := m.WithTxRW(t.Context(), func(tx *model.TxRW) error {
		_, err := tx.MovieCreate("Dune", "")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, m, "/app/about")
	for _, path := range []string{"/dune", "/app/movies/dune"} {
		t.Run(path, func(t *testing.T) {
			a.Preview(t.Context(), &url.URL{Path: path}, func(dest string, v hi.View) hi.Preview {
				if title, _ := renderViewAt(t, dest, v); title != "Dune" {
					t.Fatalf("preview title = %q, want Dune", title)
				}
				return hi.Preview{}
			})
			if title, _ := renderApp(t, a); title != "About" {
				t.Fatalf("live title after preview = %q, want About", title)
			}
		})
	}
}

func TestPreviewPreservesState(t *testing.T) {
	a := newTestApp(t, newTestModel(t), "/app/about")
	a.notes = []ui.Note{{ID: "note-1", Title: "Preview must not replay this"}}
	for _, tt := range []struct {
		path, dest, title, content string
	}{
		{"/app", "/app/profile", "Profile", "Change Name"},
		{"/", "/", "", `data-controller="home"`},
		{"/apple", "/apple", "Not Found", "Not Found"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			u := &url.URL{Path: tt.path, RawQuery: "q=kept", Fragment: "section"}
			var dest, title, body string
			a.Preview(t.Context(), u, func(d string, v hi.View) hi.Preview {
				dest = d
				var n domi.Node
				title, n = renderViewAt(t, d, v)
				var b strings.Builder
				if err := domi.RenderTo(&b, n); err != nil {
					t.Fatal(err)
				}
				body = b.String()
				return hi.Preview{}
			})
			if dest != tt.dest+"?q=kept#section" || title != tt.title {
				t.Fatalf("preview = %q, %q", dest, title)
			}
			if !strings.Contains(body, tt.content) || strings.Contains(body, a.notes[0].Title) {
				t.Error("preview must render the destination without replaying notifications")
			}
			if a.path != "/app/about" || len(a.notes) != 1 || u.Path != tt.path {
				t.Error("preview changed the live app or the requested URL")
			}
		})
	}
}

func TestRendererTransactionScope(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(map[bool]string{false: "View", true: "Preview"}[preview], func(t *testing.T) {
			m, db := newTestModelDB(t)
			a := newTestApp(t, m, "/app/profile")
			calls := 0
			render := func(path string, v hi.View) {
				renderViewAt(t, path, hi.Lazy(func() hi.View {
					calls++
					if got := db.Stats().InUse; got != 1 {
						t.Errorf("connections in use during lazy rendering = %d, want 1", got)
					}
					return v
				}))
			}
			if preview {
				a.Preview(t.Context(), &url.URL{Path: "/app/about"}, func(dest string, v hi.View) hi.Preview {
					render(dest, v)
					return hi.Preview{}
				})
			} else {
				a.View(t.Context(), func(v hi.View) hi.Page {
					render(a.path, v)
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
		title, n = renderViewAt(t, a.path, v)
		return hi.Page{}
	})
	return title, n
}

// Standalone hi.Render has no request path. Use the handler's renderer
// so route-dependent views see the same environment as a real request.
func renderViewAt(t *testing.T, path string, v hi.View) (title string, n domi.Node) {
	t.Helper()
	h := hi.Handler(
		func(context.Context, *url.URL) (*testViewApp, cmd) {
			return &testViewApp{v: v}, nil
		},
		msg.OnURLRequest, msg.OnURLChange,
		domi.Document(func(s string, body domi.Node) domi.Node {
			title, n = s, body
			return body
		}),
	)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", path, nil))
	if r.Code != http.StatusOK {
		t.Fatalf("render status = %d", r.Code)
	}
	return title, n
}

type testViewApp struct {
	hi.App[msg.Msg]
	v hi.View
}

func (a *testViewApp) View(ctx context.Context, render hi.PageRenderer) hi.Page {
	return render(a.v)
}

func (a *testViewApp) Subscriptions(context.Context) domi.Sub[msg.Msg] { return nil }
