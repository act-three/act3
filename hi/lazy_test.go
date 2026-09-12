package hi_test

import (
	"strings"
	"testing"

	"ily.dev/act3/hi"
)

func TestLazyResolution(t *testing.T) {
	contexts := []struct {
		name string
		wrap func(hi.View) hi.View
	}{
		{"root", func(v hi.View) hi.View { return v }},
		{"group", func(v hi.View) hi.View { return hi.Group(hi.Text("before"), v, hi.Text("after")) }},
		{"stack", func(v hi.View) hi.View { return hi.HStack(v, hi.Spacer()) }},
		{"zstack", func(v hi.View) hi.View { return hi.ZStack(v, hi.Text("above")) }},
		{"grid", func(v hi.View) hi.View { return hi.Grid(hi.Columns(2), v) }},
		{"button", func(v hi.View) hi.View { return hi.Button("/", v) }},
		{"scroll", func(v hi.View) hi.View { return hi.ScrollView(hi.Vertical, v) }},
		{"overlay", func(v hi.View) hi.View { return hi.Text("base").Overlay(hi.TopTrailing, v) }},
		{"underlay", func(v hi.View) hi.View { return hi.Text("base").Underlay(hi.Center, v) }},
		{"modifiers", func(v hi.View) hi.View {
			return v.Background(hi.Red).Padding().Opacity(0.5).Background(hi.Blue).
				Frame(hi.Width(100)).FrameBounds(hi.MinWidth(80)).
				FrameRatio(2, 1, hi.Horizontal).Sticky(hi.Edges(0)).
				Overlay(hi.Center, hi.Text("overlay")).Title("page")
		}},
	}
	contents := []struct {
		name string
		view hi.View
	}{
		{"empty", hi.Empty()},
		{"one", hi.Text("one")},
		{"many", hi.Group(hi.Text("first"), hi.Text("second"))},
		{"nested", hi.Group(hi.Empty(), hi.Lazy(func() hi.View {
			return hi.Group(hi.Text("nested"), hi.Empty(), hi.Text("siblings"))
		}))},
		{"root scroll", hi.ScrollView(hi.Vertical, hi.Text("scrolling"))},
		{"empty again", hi.Empty()},
	}
	for _, ctx := range contexts {
		t.Run(ctx.name, func(t *testing.T) {
			var current hi.View
			calls := 0
			v := ctx.wrap(hi.Lazy(func() hi.View {
				calls++
				return current
			}))
			if calls != 0 {
				t.Fatal("Lazy called during construction")
			}
			for i, content := range contents {
				t.Run(content.name, func(t *testing.T) {
					current = content.view
					got := render(t, v)
					want := render(t, ctx.wrap(current))
					if got != want {
						t.Errorf("lazy content differs from direct content:\ngot:  %s\nwant: %s", got, want)
					}
					if calls != i+1 {
						t.Errorf("callback calls = %d, want %d", calls, i+1)
					}
				})
			}
		})
	}
}

func TestLazyUnselected(t *testing.T) {
	v := hi.Lazy(func() hi.View {
		t.Fatal("unselected Lazy called")
		return hi.Empty()
	}).Padding().Background(hi.Red)
	got := render(t, hi.Group(
		hi.If(false, v),
		hi.IfElse(true, hi.Text("selected"), v),
		hi.IfElse(false, v, hi.Text("also selected")),
	))
	want := render(t, hi.Group(hi.Text("selected"), hi.Text("also selected")))
	if got != want {
		t.Errorf("unselected branches affected output:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestLazyForKeys(t *testing.T) {
	shared := hi.Text("shared")
	plain := render(t, shared)
	current := hi.View(shared)
	items := []string{"a", "b"}
	buildCalls, keyCalls, lazyCalls := 0, 0, 0
	v := hi.For(items, func(s string) string {
		keyCalls++
		return s
	}, func(string) hi.View {
		buildCalls++
		return hi.Lazy(func() hi.View {
			lazyCalls++
			return current
		})
	})
	if buildCalls != 2 || keyCalls != 2 || lazyCalls != 0 {
		t.Fatalf("construction calls: build=%d, key=%d, lazy=%d", buildCalls, keyCalls, lazyCalls)
	}
	for _, content := range []hi.View{
		hi.Group(hi.Empty(), shared, hi.Text("second")),
		hi.Empty(),
		shared,
	} {
		current = content
		got := render(t, v)
		want := render(t, hi.For(items, func(s string) string { return s }, func(string) hi.View {
			return current
		}))
		if got != want {
			t.Errorf("lazy item keys differ from direct items:\ngot:  %s\nwant: %s", got, want)
		}
		if strings.Contains(got, "shared") {
			for _, key := range items {
				if n := strings.Count(got, `domi-key="`+key+`"`); n != 1 {
					t.Errorf("key %q occurs %d times, want 1", key, n)
				}
			}
		} else if strings.Contains(got, "domi-key") {
			t.Error("empty items contributed keys")
		}
		if got := render(t, shared); got != plain {
			t.Error("key assignment modified the shared view")
		}
	}
	if buildCalls != 2 || keyCalls != 2 || lazyCalls != 6 {
		t.Errorf("render calls: build=%d, key=%d, lazy=%d", buildCalls, keyCalls, lazyCalls)
	}
}

func TestLazySharedModifiers(t *testing.T) {
	shared := hi.Text("shared")
	v := hi.Lazy(func() hi.View { return shared })
	got := render(t, hi.Group(v.Padding().Background(hi.Red), v, shared))
	want := render(t, hi.Group(shared.Padding().Background(hi.Red), shared, shared))
	if got != want {
		t.Errorf("lazy modifiers affected shared content:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestGroupCopiesViews(t *testing.T) {
	vs := []hi.View{hi.Text("original")}
	v := hi.Group(vs...)
	vs[0] = hi.Text("replacement")
	if got, want := render(t, v), render(t, hi.Text("original")); got != want {
		t.Errorf("mutating arguments changed Group:\ngot:  %s\nwant: %s", got, want)
	}
}
