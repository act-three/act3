package hi_test

import (
	"slices"
	"testing"

	"ily.dev/act3/hi"
)

func TestFirst(t *testing.T) {
	a, b := hi.Text("a"), hi.Text("b")
	cases := []struct {
		name string
		vs   []hi.View
		want hi.View
	}{
		{"no alternatives", nil, hi.Empty()},
		{"all empty", []hi.View{hi.Empty(), hi.Group(), hi.Lazy(hi.Empty)}, hi.Empty()},
		{"first wins", []hi.View{a, b}, a},
		{"skip empty", []hi.View{hi.Group(hi.Empty(), hi.Lazy(hi.Empty)), b}, b},
		{"whole group", []hi.View{hi.Group(hi.Empty(), a, hi.Group(b)), hi.Text("later")}, hi.Group(a, b)},
		{"empty container", []hi.View{hi.VStack(), a}, hi.VStack()},
		{"empty text", []hi.View{hi.Text(""), a}, hi.Text("")},
		{"nested first", []hi.View{hi.First(hi.Empty()), hi.First(hi.Empty(), a, b)}, a},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := render(t, hi.First(tc.vs...))
			want := render(t, tc.want)
			if got != want {
				t.Errorf("First:\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func TestFirstLazyResolution(t *testing.T) {
	selected := -1
	var calls []int
	var vs []hi.View
	content := hi.Group(hi.Text("first"), hi.Text("second"))
	for i := range 3 {
		vs = append(vs, hi.Lazy(func() hi.View {
			calls = append(calls, i)
			return hi.If(i == selected, content)
		}))
	}
	v := hi.First(vs...)
	if len(calls) != 0 {
		t.Fatal("First resolved alternatives during construction")
	}
	for _, choice := range []int{1, 0, 2, -1} {
		selected = choice
		calls = nil
		got := render(t, v)
		want := render(t, hi.If(choice >= 0, content))
		if got != want {
			t.Errorf("choice %d:\ngot:  %s\nwant: %s", choice, got, want)
		}
		wantCalls := []int{0, 1, 2}
		if choice >= 0 {
			wantCalls = wantCalls[:choice+1]
		}
		if !slices.Equal(calls, wantCalls) {
			t.Errorf("choice %d: calls = %v, want %v", choice, calls, wantCalls)
		}
	}
}

func TestFirstModifiers(t *testing.T) {
	decorate := func(v hi.View) hi.View {
		return v.Background(hi.Red).Padding().Opacity(0.5).Background(hi.Blue).
			Frame(hi.Width(100)).Overlay(hi.Center, hi.Text("overlay"))
	}
	a, b := hi.Text("a"), hi.Text("b")
	shared := hi.Group(a, b)
	v := hi.First(hi.Empty(), hi.Lazy(func() hi.View { return shared }))
	got := render(t, hi.HStack(decorate(v), shared))
	want := render(t, hi.HStack(decorate(a), decorate(b), shared))
	if got != want {
		t.Errorf("modifiers did not distribute over the selected group:\ngot:  %s\nwant: %s", got, want)
	}
	got = render(t, decorate(hi.First(hi.Empty())))
	want = render(t, hi.Empty())
	if got != want {
		t.Errorf("modifiers gave empty First content:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestFirstCopiesViews(t *testing.T) {
	vs := []hi.View{hi.Text("original")}
	v := hi.First(vs...)
	vs[0] = hi.Text("replacement")
	if got, want := render(t, v), render(t, hi.Text("original")); got != want {
		t.Errorf("mutating arguments changed First:\ngot:  %s\nwant: %s", got, want)
	}
}
