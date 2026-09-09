package ui

import (
	"strings"
	"testing"
)

func TestBorderShadowLayers(t *testing.T) {
	var got environment
	v := base{func(env environment) box { got = env; return box{} }}.
		BorderShadow(1+2i, -3, -2, 4, Red).
		BorderShadow(0, 1, 0, 0, Blue)
	v.nodes()[0](environment{})
	if !got.hasPaint {
		t.Fatal("shadow did not establish a paint boundary")
	}
	css := borderShadowList(defaultTheme, got.paintUnder(0).shadow)
	want := "calc(1px + 0.125rem) -3px 4px -2px " + Red.color().colorCoords(defaultTheme).css() +
		",0px 1px 0px 0px " + Blue.color().colorCoords(defaultTheme).css()
	if css != want {
		t.Fatalf("layers = %s, want inner shadow first %s", css, want)
	}
}

func TestBorderShadowNegativeBlur(t *testing.T) {
	got := borderShadowList(defaultTheme, []shadow{{blur: 2 - 4i, color: oklch{a: 1}}})
	if !strings.Contains(got, "max(0px,calc(2px - 0.25rem))") {
		t.Errorf("negative resolved blur must remain valid: %s", got)
	}
}
