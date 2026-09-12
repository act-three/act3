package hi

import (
	"context"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"ily.dev/domi"
)

type navigationApp struct {
	stubApp
	update func(context.Context, int) domi.Cmd[int]
}

func (a *navigationApp) Update(ctx context.Context, m int) domi.Cmd[int] {
	return a.update(ctx, m)
}

func (a *navigationApp) Subscriptions(context.Context) domi.Sub[int] { return nil }

func TestInstanceURLChange(t *testing.T) {
	ctx := t.Context()
	app := &navigationApp{}
	in := &instance[int, *navigationApp]{app: app, path: []string{"initial"}}
	want := []string{"next"}
	calls := 0
	app.update = func(gotCtx context.Context, m int) domi.Cmd[int] {
		calls++
		if gotCtx != ctx || m != 7 {
			t.Fatalf("Update received context %v and message %d", gotCtx, m)
		}
		if !slices.Equal(in.path, want) {
			t.Errorf("path before app Update = %q, want %q", in.path, want)
		}
		return nil
	}
	in.Update(ctx, msg[int]{appmsg: 7, path: []string{"next"}})
	in.Update(ctx, wrapMsg(7))
	if !slices.Equal(in.path, want) {
		t.Errorf("ordinary message changed instance path to %q", in.path)
	}
	app.preview = func(_ context.Context, _ *url.URL, render PreviewRenderer) Preview {
		return render("/preview", Text("preview"))
	}
	in.Preview(ctx, &url.URL{Path: "/preview"})
	if !slices.Equal(in.path, want) {
		t.Errorf("preview changed instance path to %q", in.path)
	}
	want = nil
	in.Update(ctx, msg[int]{appmsg: 7, path: []string{}})
	if calls != 3 {
		t.Errorf("app Update called %d times, want 3", calls)
	}
}

// pathProbe checks both stages: receiving the path during resolution does
// not by itself ensure that child containers receive it during lowering.
func pathProbe(t *testing.T, want []string, resolved, lowered *int) View {
	t.Helper()
	return base(func(r resenv) []node {
		*resolved++
		if !slices.Equal(r.path, want) {
			t.Errorf("resolution path = %q, want %q", r.path, want)
		}
		return []node{func(env environment) box {
			*lowered++
			if !slices.Equal(env.renv.path, want) {
				t.Errorf("lowering path = %q, want %q", env.renv.path, want)
			}
			return textLeaf("path").render(env)
		}}
	})
}

func TestInstancePathEnvironment(t *testing.T) {
	app := &navigationApp{update: func(context.Context, int) domi.Cmd[int] { return nil }}
	in := &instance[int, *navigationApp]{app: app, path: []string{"initial"}}
	for _, want := range [][]string{{"next", "a/b", ""}, {}} {
		in.Update(t.Context(), msg[int]{path: want})
		resolved, lowered := 0, 0
		probe := pathProbe(t, want, &resolved, &lowered)
		app.view = VStack(
			Group(probe, Lazy(func() View { return First(Empty(), probe) })),
			Button("/", probe),
			For([]int{1}, func(int) string { return "item" }, func(int) View { return probe }),
		)
		in.View(t.Context())
		if resolved != 4 || lowered != 4 {
			t.Errorf("resolved=%d, lowered=%d; want 4 each", resolved, lowered)
		}
	}
}

func TestHandlerPathEnvironment(t *testing.T) {
	resolved, lowered := 0, 0
	h := Handler(
		func(context.Context, *url.URL) (*stubApp, domi.Cmd[struct{}]) {
			return &stubApp{view: pathProbe(t, []string{"a/b", "edit", ""}, &resolved, &lowered)}, nil
		},
		func(*url.URL) struct{} { return struct{}{} },
		func(*url.URL) struct{} { return struct{}{} },
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/a%2Fb/edit/?q=x", nil))
	if rec.Code != 200 || resolved != 1 || lowered != 1 {
		t.Fatalf("initial request: status=%d, resolved=%d, lowered=%d", rec.Code, resolved, lowered)
	}
}

func TestPreviewPathEnvironment(t *testing.T) {
	for _, tc := range []struct {
		dest string
		want []string
	}{
		{"/canonical/a%2Fb/?q=x#anchor", []string{"canonical", "a/b", ""}},
		{"/", nil},
	} {
		t.Run(tc.dest, func(t *testing.T) {
			current := []string{"current"}
			resolved, lowered := 0, 0
			app := &stubApp{view: pathProbe(t, current, &resolved, &lowered)}
			in := &instance[struct{}, *stubApp]{app: app, path: current}
			app.preview = func(_ context.Context, _ *url.URL, render PreviewRenderer) Preview {
				return render(tc.dest, Lazy(func() View {
					in.View(t.Context())
					return VStack(pathProbe(t, tc.want, &resolved, &lowered))
				}))
			}
			dest, _, _ := in.Preview(t.Context(), &url.URL{Path: "/requested"})
			if dest != tc.dest || !slices.Equal(in.path, current) {
				t.Errorf("preview: dest=%q, instance path=%q", dest, in.path)
			}
			if resolved != 2 || lowered != 2 {
				t.Errorf("resolved=%d, lowered=%d; want 2 each", resolved, lowered)
			}
		})
	}
}

func TestRenderPathEnvironment(t *testing.T) {
	resolved, lowered := 0, 0
	Render(pathProbe(t, nil, &resolved, &lowered))
	if resolved != 1 || lowered != 1 {
		t.Errorf("resolved=%d, lowered=%d; want 1 each", resolved, lowered)
	}
}
