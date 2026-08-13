package ui_test

import (
	"regexp"
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
	"ily.dev/domi"
)

// TestForegroundInnermostWins pins the wrapper model for inherited
// modifiers: the Foreground closest to the content styles it, with or
// without structure in between.
func TestForegroundInnermostWins(t *testing.T) {
	for _, tt := range []struct {
		name string
		v    ui.View
	}{
		{"adjacent", ui.Text("hi").Foreground("rgb(255, 0, 0)").Foreground("rgb(0, 0, 255)")},
		{"frame between", ui.Text("hi").Foreground("rgb(255, 0, 0)").Frame(ui.Width(100)).Foreground("rgb(0, 0, 255)")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, tt.v, func(s *uitest.Session) {
				var color string
				s.Eval(`getComputedStyle(document.querySelector(".ui-text")).color`, &color)
				if color != "rgb(255, 0, 0)" {
					t.Errorf("text color = %s, want rgb(255, 0, 0)", color)
				}
			})
		})
	}
}

// TestInheritedModifierCollapses pins the collapsed lowering for
// inherited modifiers: no wrapper element, with the consumed
// declaration landing exactly once — on the first element boundary
// under the modifier — and descendants styled by CSS inheritance.
func TestInheritedModifierCollapses(t *testing.T) {
	html := render(t, ui.VStack(ui.Text("a"), ui.Text("b")).Foreground("red"))
	if strings.Contains(html, "ui-mod") {
		t.Fatalf("Foreground should not produce a wrapper:\n%s", html)
	}
	if got := classRule(t, html, `class="ui-vstack (ui-\w+)"`); got != "color:red" {
		t.Errorf("stack box rule = %q, want the consumed color", got)
	}
	m := regexp.MustCompile(`\.(ui-\w+)\{color:red\}`).FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no color rule in the sheet:\n%s", html)
	}
	if n := strings.Count(html, m[1]); n != 2 { // the rule and one use
		t.Errorf("consumed color class appears %d times, want 2:\n%s", n, html)
	}
}

// TestModifierBeatsComponentChrome pins the collapse against
// component chrome: consumed Font and Foreground values land on the
// button element itself, where its font:inherit and color:inherit —
// the absence of an opinion — must yield to them.
func TestModifierBeatsComponentChrome(t *testing.T) {
	v := ui.Button(ui.Text("x"), struct{}{}).Font(ui.Title).Foreground("rgb(255, 0, 0)")
	stage(t, v, func(s *uitest.Session) {
		var size, color string
		s.Eval(`getComputedStyle(document.querySelector(".ui-button")).fontSize`, &size)
		s.Eval(`getComputedStyle(document.querySelector(".ui-button")).color`, &color)
		if size != "24px" { // Title, 1.5rem
			t.Errorf("button font-size = %s, want the modifier's 24px", size)
		}
		if color != "rgb(255, 0, 0)" {
			t.Errorf("button color = %s, want the modifier's rgb(255, 0, 0)", color)
		}
	})
}

// TestOpacityNests pins the wrapper model for opacity: each
// application is its own wrapper, so opacities multiply.
func TestOpacityNests(t *testing.T) {
	stage(t, ui.Text("hi").Opacity(0.5).Opacity(0.5), func(s *uitest.Session) {
		var ops []string
		s.Eval(`[...document.querySelectorAll(".ui-mod")].map(e => getComputedStyle(e).opacity)`, &ops)
		if len(ops) != 2 || ops[0] != "0.5" || ops[1] != "0.5" {
			t.Errorf("nested wrapper opacities = %v, want [0.5 0.5]", ops)
		}
	})
}

// TestOpacityComposesWithDisabled pins that a modifier opacity
// multiplies with a component's own state opacity rather than either
// clobbering the other.
func TestOpacityComposesWithDisabled(t *testing.T) {
	v := ui.Button(ui.Text("x"), struct{}{}).Disabled(true).Opacity(0.1)
	stage(t, v, func(s *uitest.Session) {
		var mod, btn string
		s.Eval(`getComputedStyle(document.querySelector(".ui-mod")).opacity`, &mod)
		s.Eval(`getComputedStyle(document.querySelector(".ui-button")).opacity`, &btn)
		if mod != "0.1" || btn != "0.5" {
			t.Errorf("opacities = %s x %s, want 0.1 x 0.5", mod, btn)
		}
	})
}

// TestBackgroundStacks pins the paint stack: an outer Background
// paints behind an inner one, visible where the inner is translucent.
func TestBackgroundStacks(t *testing.T) {
	html := render(t, ui.Text("x").Background("#0008").Background("#fff"))
	inner := classRule(t, html, `class="ui-mod (ui-\w+)"><div class="ui-text"`)
	if inner != "background-color:#0008" {
		t.Errorf("inner paint = %q, want the first Background", inner)
	}
	outer := classRule(t, html, `class="ui-mod (ui-\w+)"><div class="ui-mod`)
	if outer != "background-color:#fff" {
		t.Errorf("outer paint = %q, want the second Background", outer)
	}
}

// TestBackgroundShapeOrder pins the shape's write order: a shape applied after
// paint shapes it; paint applied after a shape lands outside it.
func TestBackgroundShapeOrder(t *testing.T) {
	// Background then shape: the shape merges onto the paint wrapper —
	// a red capsule.
	shaped := render(t, ui.Text("x").Background("red").BorderShape(ui.Capsule))
	if got := classRule(t, shaped, `class="ui-mod ui-border-capsule (ui-\w+)"`); got != "background-color:red" {
		t.Errorf("shape after paint should shape the paint element, got %q:\n%s", got, shaped)
	}

	// Shape then background: the shape stays on the text element and
	// the paint wraps it, unshaped — a red rectangle.
	square := render(t, ui.Text("x").BorderShape(ui.Capsule).Background("red"))
	if got := classRule(t, square, `class="ui-mod (ui-\w+)"`); got != "background-color:red" {
		t.Errorf("paint after shape should land on a wrapper, got %q:\n%s", got, square)
	}
	if !strings.Contains(square, `class="ui-text ui-border-capsule"`) {
		t.Errorf("the shape should stay on the inner element:\n%s", square)
	}
}

// TestBorderShapeRepetition pins shape inheritance: the shape descends
// to the first box, so the innermost of two shapes wins and the outer
// one is inert — no wrapper, no class.
func TestBorderShapeRepetition(t *testing.T) {
	html := render(t, ui.Text("x").BorderShape(ui.RoundedRectangle).BorderShape(ui.Capsule))
	if !strings.Contains(html, `class="ui-text ui-border-rounded"`) {
		t.Errorf("innermost shape should land on the text element:\n%s", html)
	}
	if strings.Contains(html, "ui-border-capsule") || strings.Contains(html, "ui-mod") {
		t.Errorf("outer shape should be inert:\n%s", html)
	}
}

// TestBorderShapeShapesColorPaint pins render-time consumption: a
// Color draws on an inner paint element, so the shape it consumes is
// realized there rather than on its own (unpainted) element.
func TestBorderShapeShapesColorPaint(t *testing.T) {
	html := render(t, ui.Color("red").BorderShape(ui.Ellipse))
	if !strings.Contains(html, `class="ui-color-paint ui-border-ellipse`) {
		t.Errorf("shape should land on the color's paint element:\n%s", html)
	}
}

// TestWrapperKeepsRigidity pins the layout transparency of wrappers:
// a wrapper forwards its subview's rigid axes, so it resists flex
// compression on the subview's behalf. A frame does the same on an
// auto axis, which takes the subview's sizing.
func TestWrapperKeepsRigidity(t *testing.T) {
	for _, tt := range []struct {
		name string
		v    ui.View
		want string
	}{
		{"style wrapper", ui.HStack(ui.Text("x").FixedSize().Background("red")), `class="ui-mod ui-rigid`},
		{"frame auto axis", ui.HStack(ui.Text("x").FixedSize().Frame(ui.Height(40))), `class="ui-frame ui-rigid`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			if !strings.Contains(html, tt.want) {
				t.Errorf("wrapper should carry its subview's rigidity, want %q:\n%s", tt.want, html)
			}
		})
	}
}

// TestTextStyleInnermostWins pins the whole-text modifiers to the same
// rule as their generic counterparts: the first (innermost) value wins.
func TestTextStyleInnermostWins(t *testing.T) {
	html := render(t, ui.Text("x").TextForeground("#111").TextForeground("#222"))
	if !strings.Contains(html, "color:#111") || strings.Contains(html, "#222") {
		t.Errorf("repeated TextForeground should keep the first color:\n%s", html)
	}
	html = render(t, ui.Text("x").TextFont(ui.Title).TextFont(ui.Caption))
	if !strings.Contains(html, "ui-font-title") || strings.Contains(html, "ui-font-caption") {
		t.Errorf("repeated TextFont should keep the first size:\n%s", html)
	}
}

// stageApp is stage with unlayered app CSS placed before ui.CSS in the
// document, so an app rule can win only through the xui cascade layer,
// never through source order.
func stageApp(t *testing.T, appCSS string, v ui.View, fn func(*uitest.Session)) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("<!doctype html><meta charset=utf-8><style>")
	sb.WriteString(appCSS)
	sb.WriteString("</style><style>")
	sb.WriteString(ui.CSS)
	sb.WriteString(`</style><body>`)
	if err := domi.RenderTo(&sb, new(ui.Renderer).Render(v)); err != nil {
		t.Fatalf("render: %v", err)
	}
	uitest.Run(t, 600, 400, sb.String(), fn)
}

// TestAppCSSBeatsStaticSheet pins the app-vs-xui contract for ui.css:
// every rule it emits sits in the xui layer, so an unlayered app class
// overrides it at equal specificity regardless of source order. The
// fixture is the one HTML's documentation invites: an app class
// restyling the host adapter's interior layout.
func TestAppCSSBeatsStaticSheet(t *testing.T) {
	v := ui.HTML(domi.Text("hi")).Class("app-host")
	stageApp(t, ".app-host{place-items:stretch}", v, func(s *uitest.Session) {
		var align, justify string
		s.Eval(`getComputedStyle(document.querySelector(".ui-html")).alignItems`, &align)
		s.Eval(`getComputedStyle(document.querySelector(".ui-html")).justifyItems`, &justify)
		if align != "stretch" || justify != "stretch" {
			t.Errorf("place-items = %s %s, want the app's stretch stretch", align, justify)
		}
	})
}

// TestAppCSSBeatsDynamicSheet pins the same contract for the
// render-time hashed sheet, whose style element follows the app's in
// the document and would otherwise win by source order.
func TestAppCSSBeatsDynamicSheet(t *testing.T) {
	v := ui.Text("hi").Padding(ui.Edges(16)).Class("app-pad")
	stageApp(t, ".app-pad{padding:0}", v, func(s *uitest.Session) {
		var pad string
		s.Eval(`getComputedStyle(document.querySelector(".app-pad")).paddingTop`, &pad)
		if pad != "0px" {
			t.Errorf("padding-top = %s, want the app's 0px", pad)
		}
	})
}
