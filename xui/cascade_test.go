package ui_test

import (
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
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

// TestBackgroundShapeOrder pins the shape slot: a shape applied after
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
		{"style wrapper", ui.HStack(ui.Text("x").FixedSize().Foreground("red")), `class="ui-mod ui-rigid`},
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
