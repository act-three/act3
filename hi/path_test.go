package hi

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"ily.dev/domi"
)

func renderPathTest(t *testing.T, path string, v View) Page {
	t.Helper()
	u, err := url.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	in := instance[struct{}, App[struct{}]]{theme: defaultTheme, icons: defaultIconSource, path: urlPath(u)}
	return in.render(v, in.path)
}

func TestPathMatching(t *testing.T) {
	cases := []struct {
		url, match   string
		prefix, full bool
	}{
		{"/", "/", true, true},
		{"/", "", true, true},
		{"/app", "/", true, false},
		{"/app", "/app", true, true},
		{"/app", "app", true, true},
		{"/app/profile", "/app", true, false},
		{"/apple", "/app", false, false},
		{"/app", "/app/profile", false, false},
		{"/App", "/app", false, false},
		{"/app/", "/app", true, false},
		{"/app/", "/app/", true, false},
		{"/app", "/app/", true, true},
		{"/app//x", "/app/", true, false},
		{"/a/b", "/a/b", true, true},
		{"/a//b", "/a/b", false, false},
		{"/app/../x", "/app", true, false},
		{"/app/../x", "/x", false, false},
		{"/app/./tasks//", "/app//./", true, false},
		{"/app", "//other/../app//", true, true},
		{"/../../app", "../../app", false, false},
		{"/app", "../../app", true, true},
		{"/", ".", true, true},
		{"/", "app/..", true, true},
		{"/app/%2e%2e/x", "/app", true, false},
		{"/a%2Fb/c", "/a%2Fb", false, false},
		{"/a%2Fb", "/a/b", false, false},
		{"/a/b", "/a%2Fb", false, false},
		{"/a%252Fb", "/a%2Fb", true, true},
		{"/100%25", "/100%", true, true},
		{"/100%25", "/100%25", false, false},
		{"/bad%25zz", "/bad%zz", true, true},
		{"/a+b", "/a+b", true, true},
		{"/a%2Bb", "/a+b", true, true},
		{"/a%20b", "/a b", true, true},
		{"/caf%C3%A9", "/café", true, true},
		{"/app?q=x#anchor", "/app", true, true},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s/%s/methods", tc.url, tc.match), func(t *testing.T) {
			u, err := url.Parse(tc.url)
			if err != nil {
				t.Fatal(err)
			}
			p := RequestPath(urlPath(u))
			if got := p.HasPrefix(tc.match); got != tc.prefix {
				t.Errorf("HasPrefix(%q) = %v, want %v", tc.match, got, tc.prefix)
			}
			if got := p.Equal(tc.match); got != tc.full {
				t.Errorf("Equal(%q) = %v, want %v", tc.match, got, tc.full)
			}
		})
		for _, full := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%s/full=%v", tc.url, tc.match, full), func(t *testing.T) {
				calls := 0
				var gotPath RequestPath
				child := Lazy(func() View {
					calls++
					return PathReader(func(p RequestPath) View {
						gotPath = p
						return Text("matched").Title("matched")
					})
				})
				match := PathPrefix
				want := tc.prefix
				u, err := url.Parse(tc.url)
				if err != nil {
					t.Fatal(err)
				}
				wantPath := urlPath(u)
				if full {
					match, want = Path, tc.full
				}
				v := First(match(tc.match, child), Text("fallback").Title("fallback"))
				if calls != 0 {
					t.Fatal("matched during construction")
				}
				page := renderPathTest(t, tc.url, v)
				if want {
					if calls != 1 || page.title != "matched" || !slices.Equal(gotPath, wantPath) {
						t.Errorf("calls=%d, title=%q, path=%q; want one match with path %q",
							calls, page.title, gotPath, wantPath)
					}
				} else if calls != 0 || page.title != "fallback" {
					t.Errorf("calls=%d, title=%q; want fallback without calling child", calls, page.title)
				}
			})
		}
	}
}

func TestPathFuncSelection(t *testing.T) {
	tasks := Button("/app/tasks", Text("Tasks"))
	profile := Button("/app/profile", Text("Profile"))
	v := PathPrefix("/app", PathReader(func(p RequestPath) View {
		return Group(tasks.Selected(p.HasPrefix("/app/tasks")), profile.Selected(p.Equal("/app/profile")))
	}))
	for _, tc := range []struct {
		path           string
		tasks, profile bool
	}{
		{"/app/tasks/edit", true, false},
		{"/app/profile", false, true},
		{"/app/tasks-other", false, false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			got := renderNode(t, renderPathTest(t, tc.path, v).page)
			want := renderNode(t, renderPathTest(t, "/", Group(
				tasks.Selected(tc.tasks), profile.Selected(tc.profile),
			)).page)
			if got != want {
				t.Errorf("sidebar selection:\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func TestPathPreservesScope(t *testing.T) {
	probe := PathReader(func(p RequestPath) View { return Text(fmt.Sprint(p)) })
	v := PathPrefix("/app", Path("/app/tasks", Group(
		probe,
		VStack(probe),
		Path("/app/tasks", probe),
	)))
	got := renderNode(t, renderPathTest(t, "/app/tasks", v).page)
	want := renderNode(t, renderPathTest(t, "/", Group(
		Text("[app tasks]"), VStack(Text("[app tasks]")), Text("[app tasks]"),
	)).page)
	if got != want {
		t.Errorf("Path changed its child's path:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestPathScope(t *testing.T) {
	contexts := []struct {
		name string
		wrap func(View) View
	}{
		{"group", func(v View) View { return v }},
		{"stack", func(v View) View { return VStack(v) }},
		{"grid", func(v View) View { return Grid(Columns(2), v) }},
		{"button", func(v View) View { return Button("/", v) }},
		{"scroll", func(v View) View { return ScrollView(Vertical, v) }},
		{"overlay", func(v View) View { return Text("base").Overlay(TopTrailing, v) }},
		{"underlay", func(v View) View { return Text("base").Underlay(Center, v) }},
		{"modifiers", func(v View) View {
			return v.Background(Red).Padding().Opacity(0.5).Background(Blue).Frame(Width(100))
		}},
		{"keyed", func(v View) View {
			return VStack(ForEach([]int{1}, func(int) string { return "item" }, func(int) View { return v }))
		}},
	}
	for _, tc := range contexts {
		t.Run(tc.name, func(t *testing.T) {
			probe := PathReader(func(p RequestPath) View { return Text(fmt.Sprint(p)) })
			v := Group(PathPrefix("/app", tc.wrap(Group(
				probe,
				PathPrefix("/app/tasks", tc.wrap(Lazy(func() View { return probe }))),
				probe,
			))), probe)
			wantView := Group(tc.wrap(Group(
				Text("[app tasks edit]"), tc.wrap(Text("[app tasks edit]")), Text("[app tasks edit]"),
			)), Text("[app tasks edit]"))
			got := renderNode(t, renderPathTest(t, "/app/tasks/edit", v).page)
			want := renderNode(t, renderPathTest(t, "/", wantView).page)
			if got != want {
				t.Errorf("scoped path:\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func TestPathFirstIsolation(t *testing.T) {
	bad := Lazy(func() View {
		t.Fatal("unselected route resolved")
		return Empty()
	})
	probe := PathReader(func(p RequestPath) View { return Text(fmt.Sprint(p)) })
	v := Group(First(
		PathPrefix("/app", Path("/missing", bad)).Padding().Background(Red),
		PathPrefix("/app", PathPrefix("/app/tasks", Group(probe, probe))),
		bad,
	), probe)
	got := renderNode(t, renderPathTest(t, "/app/tasks/edit", v).page)
	want := renderNode(t, renderPathTest(t, "/", Group(
		Text("[app tasks edit]"), Text("[app tasks edit]"), Text("[app tasks edit]"),
	)).page)
	if got != want {
		t.Errorf("route alternatives changed sibling paths:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestPathCallbackIsolation(t *testing.T) {
	var saved []string
	probe := PathReader(func(p RequestPath) View { return Text(fmt.Sprint(p)) })
	v := Group(PathReader(func(p RequestPath) View {
		saved = p
		p[0] = "changed"
		return probe
	}), probe)
	for range 2 {
		got := renderNode(t, renderPathTest(t, "/app/tasks", v).page)
		want := renderNode(t, renderPathTest(t, "/", Group(Text("[app tasks]"), Text("[app tasks]"))).page)
		if got != want {
			t.Errorf("callback changed the path:\ngot:  %s\nwant: %s", got, want)
		}
		saved[1] = "later"
	}
}

func TestPathModifiers(t *testing.T) {
	probe := PathReader(func(p RequestPath) View { return Text(fmt.Sprint(p)) })
	decorate := func(v View) View {
		return v.Background(Red).Padding().Opacity(0.5).Frame(Width(100)).Overlay(Center, probe)
	}
	shared := Group(probe, probe)
	v := Group(decorate(PathPrefix("/app", shared)), shared)
	got := renderNode(t, renderPathTest(t, "/app/tasks", v).page)
	want := renderNode(t, renderPathTest(t, "/app/tasks", Group(
		decorate(Text("[app tasks]")), decorate(Text("[app tasks]")), shared,
	)).page)
	if got != want {
		t.Errorf("modifiers changed route scope or group distribution:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestPathInitialRequest(t *testing.T) {
	h := Handler(
		func(context.Context, *url.URL) (*stubApp, domi.Cmd[struct{}]) {
			return &stubApp{view: PathPrefix("/café", Path("/café/edit", Text("initial route")))}, nil
		},
		func(*url.URL) struct{} { return struct{}{} },
		func(*url.URL) struct{} { return struct{}{} },
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/caf%C3%A9/edit?q=x", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "initial route") {
		t.Fatalf("initial path: status %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestPathURLChange(t *testing.T) {
	app := &navigationApp{update: func(context.Context, int) domi.Cmd[int] { return nil }}
	app.view = First(Path("/next", Text("next").Title("next")), Path("/", Text("root").Title("root")))
	in := &instance[int, *navigationApp]{app: app, path: []string{"initial"}}
	for _, path := range []string{"/next", "/"} {
		u, err := url.Parse(path)
		if err != nil {
			t.Fatal(err)
		}
		in.Update(t.Context(), msg[int]{path: urlPath(u)})
		title, _ := in.View(t.Context())
		want := "next"
		if path == "/" {
			want = "root"
		}
		if title != want {
			t.Errorf("path %q: title=%q, want %q", path, title, want)
		}
	}
}

func TestPathPreview(t *testing.T) {
	cases := []struct {
		current, dest string
		want          []string
	}{
		{"/current", "/canonical/edit", []string{"canonical", "edit"}},
		{"/current", "/a%2Fb", []string{"a/b"}},
		{"/app/current", "/next/?q=x#anchor", []string{"next", ""}},
		{"/current", "/", nil},
	}
	for _, tc := range cases {
		t.Run(tc.current+"/"+tc.dest, func(t *testing.T) {
			u, err := url.Parse(tc.current)
			if err != nil {
				t.Fatal(err)
			}
			app := &stubApp{view: PathReader(func(p RequestPath) View { return Text("current").Title(fmt.Sprint(p)) })}
			in := &instance[struct{}, *stubApp]{app: app, path: urlPath(u)}
			app.preview = func(_ context.Context, _ *url.URL, render PreviewRenderer) Preview {
				return render(tc.dest, PathReader(func(p RequestPath) View {
					if !slices.Equal(p, tc.want) {
						t.Errorf("preview path=%q, want %q", p, tc.want)
					}
					current, _ := in.View(t.Context())
					if current != fmt.Sprint(urlPath(u)) {
						t.Errorf("preview changed current path to %s", current)
					}
					return Text("preview").Title("preview")
				}))
			}
			dest, title, _ := in.Preview(t.Context(), &url.URL{Path: "/requested"})
			if dest != tc.dest || title != "preview" || !slices.Equal(in.path, urlPath(u)) {
				t.Fatalf("preview: dest=%q, title=%q, instance path=%q", dest, title, in.path)
			}
		})
	}
}
