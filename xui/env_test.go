package ui

import (
	"reflect"
	"strings"
	"testing"

	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// TestBoxIsInnermostWriter pins the write-order rule for contended
// values: a box contributes its own values through the same
// environment fields as the modifiers wrapping it, writing last,
// so its value wins and build applies unconditionally.
func TestBoxIsInnermostWriter(t *testing.T) {
	var env environment
	env.tag = "picture" // an outer Tag modifier
	b := imageNode{src: "x.png"}.render(env)
	var sb strings.Builder
	if err := domi.RenderTo(&sb, b.node); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := sb.String(); !strings.Contains(got, "<img ") || strings.Contains(got, "picture") {
		t.Errorf("image should keep its img tag:\n%s", got)
	}
}

// envProbe records the environment its render receives.
type envProbe struct{ got *environment }

func (n envProbe) render(env environment) box {
	*n.got = env
	return box{}
}

// TestSubviewHelpersStrip pins that the subview helpers strip the
// environment's box values before a subview renders: the env a
// subview sees is the given env, stripped, so each value stops at
// the first box under its modifier.
func TestSubviewHelpersStrip(t *testing.T) {
	pending := func() environment {
		c, s := Color("red"), Capsule
		return environment{
			shape: &s,
			attrs: attr.Class("x"),
			tag:   "b",
			fg:    &c,
			font:  Title,
		}
	}
	subview := func(t *testing.T, render func(environment, node)) {
		t.Helper()
		var got environment
		render(pending(), envProbe{&got})
		want := pending()
		want.boxenv = boxenv{}
		want.container = got.container // not a box value
		if !reflect.DeepEqual(got, want) {
			t.Errorf("subview env = %+v, want %+v", got, want)
		}
	}
	subview(t, func(env environment, n node) { wrapSubview(env, n) })
	subview(t, func(env environment, n node) { subviewsRendered(env, base{n}) })
}
