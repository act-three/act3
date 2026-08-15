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
	if got := classRule(t, html, `class="ui-vstack (ui-\w+)"`); got != "align-items:center;color:red;gap:8px" {
		t.Errorf("stack box rule = %q, want the consumed color in the stack's own set", got)
	}
	m := regexp.MustCompile(`\.(ui-\w+)\{align-items:center;color:red;gap:8px\}`).FindStringSubmatch(html)
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

// TestOpacityMultiplies pins the collapse: opacity applications
// multiply into one product, consumed as a single declaration by the
// first box below — no wrapper elements, and a product of 1 is free.
func TestOpacityMultiplies(t *testing.T) {
	stage(t, ui.Text("hi").Opacity(0.5).Opacity(0.5), func(s *uitest.Session) {
		var mods int
		s.Eval(`document.querySelectorAll(".ui-mod").length`, &mods)
		var op string
		s.Eval(`getComputedStyle(document.querySelector(".ui-text")).opacity`, &op)
		if mods != 0 || op != "0.25" {
			t.Errorf("wrappers = %d, text opacity = %s, want none at 0.25", mods, op)
		}
	})

	if html := render(t, ui.Text("x").Opacity(1)); strings.Contains(html, "opacity") || strings.Contains(html, "ui-mod") {
		t.Errorf("Opacity(1) should render nothing extra:\n%s", html)
	}
}

// TestOpacityComposesWithDisabled pins that the button's disabled
// fade is an ordinary opacity application: server-known, it joins the
// pending product, so a modifier opacity and the component fade
// multiply onto the one button element.
func TestOpacityComposesWithDisabled(t *testing.T) {
	v := ui.Button(ui.Text("x"), struct{}{}).Disabled(true).Opacity(0.1)
	stage(t, v, func(s *uitest.Session) {
		var mods int
		s.Eval(`document.querySelectorAll(".ui-mod").length`, &mods)
		var btn string
		s.Eval(`getComputedStyle(document.querySelector(".ui-button")).opacity`, &btn)
		if mods != 0 || btn != "0.05" {
			t.Errorf("wrappers = %d, button opacity = %s, want none at 0.05", mods, btn)
		}
	})

	html := render(t, ui.Button(ui.Text("x"), struct{}{}).Disabled(true).Class("inner").Opacity(0.1))
	if got := strings.Count(html, "inner"); got != 1 || !strings.Contains(html, `<button class="ui-padding ui-button inner `) {
		t.Errorf("modifiers should land on the button element:\n%s", html)
	}
}

// TestButtonBorderReset pins the normalize tier against the UA
// stylesheet: the native button border is reset, leaving the stroke
// channel as the border's only painter.
func TestButtonBorderReset(t *testing.T) {
	stage(t, ui.Button(ui.Text("x"), struct{}{}), func(s *uitest.Session) {
		var w string
		s.Eval(`getComputedStyle(document.querySelector(".ui-button")).borderTopWidth`, &w)
		if w != "0px" {
			t.Errorf("button border-width = %s, want 0px", w)
		}
	})
}

// TestBackgroundStacks pins the paint stack: an outer Background
// paints behind an inner one on the same element, visible where the
// inner is translucent — the outermost color as background-color,
// the inner colors as image layers listed innermost first.
func TestBackgroundStacks(t *testing.T) {
	html := render(t, ui.Text("x").Background("#0008").Background("#fff"))
	got := classRule(t, html, `class="ui-text (ui-\w+)"`)
	if got != "background-color:#fff;background-image:linear-gradient(#0008,#0008)" {
		t.Errorf("paint stack = %q, want the outer color under the inner layer:\n%s", got, html)
	}
}

// TestBackgroundShapeOrder pins the shape's write order: a shape applied after
// paint shapes it; paint applied after a shape lands outside it.
func TestBackgroundShapeOrder(t *testing.T) {
	// Background then shape: shape and paint share the element —
	// a red capsule.
	shaped := render(t, ui.Text("x").Background("red").BorderShape(ui.Capsule))
	if got := classRule(t, shaped, `class="ui-text (ui-\w+)"`); got != "background-color:red;border-radius:9999px" {
		t.Errorf("shape after paint should shape the paint, got %q:\n%s", got, shaped)
	}

	// Shape then background: the shape stays on the text element and
	// the paint boxes out around it, unshaped — a red rectangle.
	square := render(t, ui.Text("x").BorderShape(ui.Capsule).Background("red"))
	if got := classRule(t, square, `class="ui-mod (ui-\w+)"`); got != "background-color:red;place-items:center" {
		t.Errorf("paint after shape should land on a wrapper, got %q:\n%s", got, square)
	}
	if got := classRule(t, square, `class="ui-text (ui-\w+)"`); got != "border-radius:9999px" {
		t.Errorf("the shape should stay on the inner element, got %q:\n%s", got, square)
	}
}

// TestBorderShapeRepetition pins shape inheritance: the shape descends
// to the first box, so the innermost of two shapes wins and the outer
// one is inert — no wrapper, no declaration.
func TestBorderShapeRepetition(t *testing.T) {
	html := render(t, ui.Text("x").BorderShape(ui.RoundedRectangle).BorderShape(ui.Capsule))
	if got := classRule(t, html, `class="ui-text (ui-\w+)"`); got != "border-radius:var(--ui-radius)" {
		t.Errorf("innermost shape should land on the text element, got %q:\n%s", got, html)
	}
	if strings.Contains(html, "9999px") || strings.Contains(html, "ui-mod") {
		t.Errorf("outer shape should be inert:\n%s", html)
	}
}

// carrier returns the ::after block drawing the given stroke shadows.
func carrier(shadows string) string {
	return `&::after{border-radius:inherit;box-shadow:` + shadows + `;content:"";inset:0;pointer-events:none;position:absolute}`
}

// TestBorderStrokePaints pins the stroke lowering: a stroke is a ring
// an ::after block in the element's own rule paints over the element —
// no wrapper element.
func TestBorderStrokePaints(t *testing.T) {
	html := render(t, ui.Text("x").BorderStroke(2, "red"))
	if strings.Contains(html, "ui-mod") {
		t.Fatalf("BorderStroke should not produce a wrapper:\n%s", html)
	}
	if got := classRule(t, html, `class="ui-text (ui-\w+)"`); got != "position:relative;"+carrier("inset 0 0 0 2px red") {
		t.Errorf("stroke rule = %q:\n%s", got, html)
	}
}

// TestBorderStrokeStacks pins the stroke stack: strokes merge onto one
// element as a shadow list, the outer stroke listed first, painting
// over the inner one.
func TestBorderStrokeStacks(t *testing.T) {
	html := render(t, ui.Text("x").BorderStroke(2, "red").BorderStroke(4, "blue"))
	got := classRule(t, html, `class="ui-text (ui-\w+)"`)
	if got != "position:relative;"+carrier("inset 0 0 0 4px blue,inset 0 0 0 2px red") {
		t.Errorf("stroke stack = %q, want the outer stroke over the inner:\n%s", got, html)
	}
}

// TestBorderStrokeShapeOrder pins the shape's write order against the
// stroke: a shape applied after a stroke shapes its ring; a stroke
// applied after a shape rings the shaped box, unshaped.
func TestBorderStrokeShapeOrder(t *testing.T) {
	shaped := render(t, ui.Text("x").BorderStroke(2, "red").BorderShape(ui.Capsule))
	if got := classRule(t, shaped, `class="ui-text (ui-\w+)"`); got != "border-radius:9999px;position:relative;"+carrier("inset 0 0 0 2px red") {
		t.Errorf("shape after stroke should shape the stroke, got %q:\n%s", got, shaped)
	}

	square := render(t, ui.Text("x").BorderShape(ui.Capsule).BorderStroke(2, "red"))
	if got := classRule(t, square, `class="ui-mod (ui-\w+)"`); got != "place-items:center;position:relative;"+carrier("inset 0 0 0 2px red") {
		t.Errorf("stroke after shape should land on a wrapper, got %q:\n%s", got, square)
	}
	if got := classRule(t, square, `class="ui-text (ui-\w+)"`); got != "border-radius:9999px" {
		t.Errorf("the shape should stay on the inner element, got %q:\n%s", got, square)
	}
}

// TestBorderStrokeDoesNotInterceptClicks pins the carrier's
// hit-test transparency: a stroked container's ring covers its
// content without stealing its pointer events.
func TestBorderStrokeDoesNotInterceptClicks(t *testing.T) {
	v := ui.HStack(ui.Button(ui.Text("click"), struct{}{})).BorderStroke(4, "red")
	stage(t, v, func(s *uitest.Session) {
		var inButton bool
		s.Eval(`(() => {
			const r = document.querySelector(".ui-button").getBoundingClientRect();
			return !!document.elementFromPoint(r.x + r.width/2, r.y + r.height/2).closest(".ui-button");
		})()`, &inButton)
		if !inButton {
			t.Error("the element under the pointer is outside the button")
		}
	})
}

// TestBorderStrokeOnImage pins the replaced-element accommodation: an
// img cannot host the carrier, so the strokes box out around it.
func TestBorderStrokeOnImage(t *testing.T) {
	html := render(t, ui.Image("/x.png").BorderStroke(2, "red"))
	if got := classRule(t, html, `class="ui-mod (ui-\w+)"`); got != "place-items:center;position:relative;"+carrier("inset 0 0 0 2px red") {
		t.Errorf("image strokes should land on a wrapper, got %q:\n%s", got, html)
	}
	if !strings.Contains(html, `<img `) {
		t.Errorf("the image should render inside:\n%s", html)
	}
}

// TestBorderStrokeOnScroll pins the scroll accommodation: a carrier
// on the viewport would scroll away with the content, so the strokes
// box out around it.
func TestBorderStrokeOnScroll(t *testing.T) {
	html := render(t, ui.ScrollView(ui.Vertical, ui.Text("x")).BorderStroke(2, "red"))
	if got := classRule(t, html, `class="ui-mod [^"]*(ui-\w+)"`); got != "place-items:center;position:relative;"+carrier("inset 0 0 0 2px red") {
		t.Errorf("scroll strokes should land on a wrapper, got %q:\n%s", got, html)
	}
	if !strings.Contains(html, `class="ui-scroll ui-scroll-y`) {
		t.Errorf("the viewport should render inside:\n%s", html)
	}
}

// TestBorderStrokeOverLayers pins the ring against the layer
// composite: on a layered box the carrier joins the z ladder above
// the overlay, where its tree position alone would lose to the
// layers' indexes.
func TestBorderStrokeOverLayers(t *testing.T) {
	html := render(t, ui.Text("x").LayerOver(ui.Center, ui.Text("o")).BorderStroke(2, "red"))
	got := classRule(t, html, `class="ui-layers (ui-\w+)"`)
	want := "position:relative;" + strings.Replace(carrier("inset 0 0 0 2px red"), "position:absolute}", "position:absolute;z-index:3}", 1)
	if got != want {
		t.Errorf("layered stroke rule = %q, want the ring on the z ladder:\n%s", got, html)
	}
}

// TestBorderStrokeTakesNoSpace pins the stroke's layout contract in
// the browser: the carrier draws the ring, and the stroked box is the
// same size as an unstroked one.
func TestBorderStrokeTakesNoSpace(t *testing.T) {
	v := ui.VStack(
		ui.Text("hello").Class("plain"),
		ui.Text("hello").Class("stroked").BorderStroke(4, "red"),
	)
	stage(t, v, func(s *uitest.Session) {
		var shadow string
		s.Eval(`getComputedStyle(document.querySelector(".stroked"), "::after").boxShadow`, &shadow)
		if !strings.Contains(shadow, "inset") || !strings.Contains(shadow, "4px") {
			t.Errorf("carrier shadow = %q, want the 4px inset ring", shadow)
		}
		var pw, sw float64
		s.Eval(`document.querySelector(".plain").getBoundingClientRect().width`, &pw)
		s.Eval(`document.querySelector(".stroked").getBoundingClientRect().width`, &sw)
		if pw == 0 || pw != sw {
			t.Errorf("stroked width = %g, plain width = %g; a stroke must not affect layout", sw, pw)
		}
	})
}

// TestBorderShapeShapesColor pins render-time consumption: a Color
// paints its own box, so the shape it consumes is realized there.
func TestBorderShapeShapesColor(t *testing.T) {
	html := render(t, ui.Color("red").BorderShape(ui.Ellipse))
	if got := classRule(t, html, `class="ui-color [^"]*(ui-\w+)"`); got != "background-color:red;border-radius:50%" {
		t.Errorf("shape should land on the color's element, got %q:\n%s", got, html)
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
		{"transform box", ui.HStack(ui.Text("x").FixedSize().Opacity(0.5).Background("red")), `class="ui-mod ui-rigid`},
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
