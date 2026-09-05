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
	b := nodeImage{src: "x.png"}.render(env)
	var sb strings.Builder
	if err := domi.RenderTo(&sb, b.node); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := sb.String(); !strings.Contains(got, "<img ") || strings.Contains(got, "picture") {
		t.Errorf("image should keep its img tag:\n%s", got)
	}
}

// envProbe returns a node that records the environment it receives.
func envProbe(got *environment) node {
	return func(env environment) box {
		*got = env
		return box{}
	}
}

// TestSubviewHelpersStrip pins that the subview helpers strip the
// environment's box values before a subview renders: the env a
// subview sees is the given env, stripped, so each value stops at
// the first box under its modifier.
func TestSubviewHelpersStrip(t *testing.T) {
	pending := func() environment {
		env := environment{
			shape:      []term[Shape]{{value: Capsule}},
			attrs:      attr.Class("x"),
			tag:        "b",
			fg:         []term[color]{{value: oklch{a: 1}}},
			fontWeight: []term[string]{{value: "700"}},
			opacity:    []term[float64]{{value: 0.5}},
		}
		env.root.atRoot = true
		env.root.style.Set("isolation", "isolate")
		return env
	}
	subview := func(t *testing.T, render func(environment, node)) {
		t.Helper()
		var got environment
		render(pending(), envProbe(&got))
		want := pending()
		want.nextenv = nextenv{}
		want.root = rootenv{}
		want.container = got.container // not a box value
		if !reflect.DeepEqual(got, want) {
			t.Errorf("subview env = %+v, want %+v", got, want)
		}
	}
	subview(t, func(env environment, n node) { wrapSubview(env, n) })
	subview(t, func(env environment, n node) { renderSubviewList(env, base{n}) })
}

// TestEnvironmentModifiersPreserveAtRoot pins ownership of root-specialized
// lowering: modifiers describe the environment without deciding whether a
// particular node can accommodate it at the root.
func TestEnvironmentModifiersPreserveAtRoot(t *testing.T) {
	cases := []struct {
		name string
		mod  modifier
	}{
		{"environment", modEnv(func(env environment) environment {
			env.tag = "sentinel"
			return env
		})},
		{"transform", modTransform(func(env environment) environment {
			env.style.Set("overflow-x", "clip")
			return env
		})},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var got environment
			tt.mod(envProbe(&got))(environment{root: rootenv{atRoot: true}})
			if !got.root.atRoot {
				t.Error("modifier consumed atRoot before the node could inspect it")
			}
		})
	}
}

// TestCanScrollDocumentEnvironment pins ScrollView's own allowlist for its
// hoisted root lowering. Persistent subtree context remains valid, while any
// one-shot value intended for the removed viewport box rejects the lowering.
func TestCanScrollDocumentEnvironment(t *testing.T) {
	rootWithStyle := rootenv{atRoot: true}
	rootWithStyle.style.Set("isolation", "isolate")
	cases := []struct {
		name string
		env  environment
		want bool
	}{
		{"root marker", environment{root: rootenv{atRoot: true}}, true},
		{"not root", environment{}, false},
		{"persistent subtree context", environment{
			disabled:  true,
			lineLimit: 2,
			root:      rootenv{atRoot: true},
		}, true},
		{"root base style", environment{root: rootWithStyle}, true},
		{"tag", environment{root: rootenv{atRoot: true}, nextenv: nextenv{tag: "sentinel"}}, false},
		{"paint", environment{nextenv: nextenv{
			bg: []term[color]{{value: oklch{a: 1}}},
		}, root: rootenv{atRoot: true}}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := canScrollDocument(tt.env); got != tt.want {
				t.Errorf("canScrollDocument = %t, want %t", got, tt.want)
			}
		})
	}
}

// TestCanOverlayRootEnvironment pins Overlay's own allowlist for its hoisted
// root lowering. Root context passes through the fixed layer lowering, while
// values belonging to the ordinary composite box reject it.
func TestCanOverlayRootEnvironment(t *testing.T) {
	rootWithStyle := rootenv{atRoot: true}
	rootWithStyle.style.Set("isolation", "isolate")
	cases := []struct {
		name string
		env  environment
		want bool
	}{
		{"root marker", environment{root: rootenv{atRoot: true}}, true},
		{"root base style", environment{root: rootWithStyle}, true},
		{"not root", environment{}, false},
		{"persistent subtree context", environment{
			disabled:  true,
			lineLimit: 2,
			root:      rootenv{atRoot: true},
		}, true},
		{"tag", environment{root: rootenv{atRoot: true}, nextenv: nextenv{tag: "sentinel"}}, false},
		{"paint", environment{nextenv: nextenv{
			bg: []term[color]{{value: oklch{a: 1}}},
		}, root: rootenv{atRoot: true}}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := canOverlayRoot(tt.env); got != tt.want {
				t.Errorf("canOverlayRoot = %t, want %t", got, tt.want)
			}
		})
	}
}
