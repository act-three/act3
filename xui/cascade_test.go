package ui_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"
)

// TestForegroundInnermostWins pins the wrapper model for inherited
// modifiers: the Foreground closest to the content styles it, with or
// without structure in between.
func TestForegroundInnermostWins(t *testing.T) {
	for _, tt := range []struct {
		name string
		v    ui.View
	}{
		{"adjacent", ui.Text("hi").Foreground(ui.CSSColor("rgb(255, 0, 0)")).Foreground(ui.CSSColor("rgb(0, 0, 255)"))},
		{"frame between", ui.Text("hi").Foreground(ui.CSSColor("rgb(255, 0, 0)")).Frame(ui.Width(100)).Foreground(ui.CSSColor("rgb(0, 0, 255)"))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, tt.v, func(s *uitest.Session) {
				var color string
				s.Eval(`getComputedStyle(document.querySelector("ui-text")).color`, &color)
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
	html := render(t, ui.VStack(ui.Text("a"), ui.Text("b")).Foreground(ui.CSSColor("red")))
	if strings.Contains(html, "ui-box") {
		t.Fatalf("Foreground should not produce a wrapper:\n%s", html)
	}
	if got := classRule(t, html, `<ui-vstack class="(ui-\w+)"`); got != "align-items:center;color:red;display:inline-flex;flex-direction:column;gap:8px" {
		t.Errorf("stack box rule = %q, want the consumed color in the stack's own set", got)
	}
	m := regexp.MustCompile(`\.(ui-\w+)\{align-items:center;color:red;display:inline-flex;flex-direction:column;gap:8px\}`).FindStringSubmatch(html)
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
	v := ui.Button(struct{}{}, ui.Text("x")).Font(ui.Title).Foreground(ui.CSSColor("rgb(255, 0, 0)"))
	stage(t, v, func(s *uitest.Session) {
		var size, color string
		s.Eval(`getComputedStyle(document.querySelector("button")).fontSize`, &size)
		s.Eval(`getComputedStyle(document.querySelector("button")).color`, &color)
		if size != "24px" { // Title, 1.5rem
			t.Errorf("button font-size = %s, want the modifier's 24px", size)
		}
		if color != "rgb(255, 0, 0)" {
			t.Errorf("button color = %s, want the modifier's rgb(255, 0, 0)", color)
		}
	})
}

// TestDisabledStateMatchesARIA pins that Disabled is active on a
// control disabled either natively or by aria-disabled, so views
// lowered to elements without a disabled attribute take part.
func TestDisabledStateMatchesARIA(t *testing.T) {
	red := ui.Foreground(ui.CSSColor("rgb(255, 0, 0)"))
	for name, v := range map[string]ui.View{
		"native": ui.Text("x").Tag("button").Attr(attr.Disabled(true)).WhileDisabled(red),
		"aria":   ui.Text("x").Attr(domi.Name("aria-disabled", "true")).WhileDisabled(red),
		"none":   ui.Text("x").Tag("button").WhileDisabled(red),
	} {
		t.Run(name, func(t *testing.T) {
			stage(t, v, func(s *uitest.Session) {
				var color string
				s.Eval(`getComputedStyle(document.querySelector("ui-text, button")).color`, &color)
				if want := name != "none"; (color == "rgb(255, 0, 0)") != want {
					t.Errorf("color = %s, want disabled styling = %v", color, want)
				}
			})
		})
	}
}

// TestOpacityMultiplies pins the collapse: opacity applications
// multiply into one product, consumed as a single declaration by the
// first box below — no wrapper elements, and a product of 1 is free.
func TestOpacityMultiplies(t *testing.T) {
	stage(t, ui.Text("hi").Opacity(0.5).Opacity(0.5), func(s *uitest.Session) {
		var mods int
		s.Eval(`document.querySelectorAll("ui-box").length`, &mods)
		var op string
		s.Eval(`getComputedStyle(document.querySelector("ui-text")).opacity`, &op)
		if mods != 0 || op != "0.25" {
			t.Errorf("wrappers = %d, text opacity = %s, want none at 0.25", mods, op)
		}
	})

	if html := render(t, ui.Text("x").Opacity(1)); strings.Contains(html, "opacity") || strings.Contains(html, "ui-box") {
		t.Errorf("Opacity(1) should render nothing extra:\n%s", html)
	}
}

// TestButtonDisabledState pins that a disabled button enters the
// Disabled state whatever its action, and that an enabled one does not.
func TestButtonDisabledState(t *testing.T) {
	red := ui.Foreground(ui.CSSColor("rgb(255, 0, 0)"))
	for name, tc := range map[string]struct {
		v    ui.View
		want bool
	}{
		"send":              {ui.Button(struct{}{}, ui.Text("x")).WhileDisabled(red), false},
		"send disabled":     {ui.Button(struct{}{}, ui.Text("x")).Disabled(true).WhileDisabled(red), true},
		"navigate":          {ui.Button("/x", ui.Text("x")).WhileDisabled(red), false},
		"navigate disabled": {ui.Button("/x", ui.Text("x")).Disabled(true).WhileDisabled(red), true},
	} {
		t.Run(name, func(t *testing.T) {
			stage(t, tc.v, func(s *uitest.Session) {
				var color string
				s.Eval(`getComputedStyle(document.querySelector("button, a")).color`, &color)
				if got := color == "rgb(255, 0, 0)"; got != tc.want {
					t.Errorf("color = %s, want disabled styling = %v", color, tc.want)
				}
			})
		})
	}
}

// TestLinkDisabledState pins that a link used as a box is the
// element that performs its action, so a disabled one enters the
// Disabled state whatever its action, and an enabled one does not.
func TestLinkDisabledState(t *testing.T) {
	red := ui.Background(ui.CSSColor("rgb(255, 0, 0)"))
	for name, tc := range map[string]struct {
		v    ui.View
		want bool
	}{
		"send":              {ui.Link(struct{}{}, ui.Text("x")).WhileDisabled(red), false},
		"send disabled":     {ui.Link(struct{}{}, ui.Text("x")).Disabled(true).WhileDisabled(red), true},
		"navigate":          {ui.Link("/x", ui.Text("x")).WhileDisabled(red), false},
		"navigate disabled": {ui.Link("/x", ui.Text("x")).Disabled(true).WhileDisabled(red), true},
	} {
		t.Run(name, func(t *testing.T) {
			stage(t, tc.v, func(s *uitest.Session) {
				var color string
				s.Eval(`getComputedStyle(document.querySelector("button, a")).backgroundColor`, &color)
				if got := color == "rgb(255, 0, 0)"; got != tc.want {
					t.Errorf("background = %s, want disabled styling = %v", color, tc.want)
				}
			})
		})
	}
}

// TestOpacityComposesWithDisabled pins that the button's disabled
// fade is an ordinary opacity application: server-known, it joins the
// pending product, so a modifier opacity and the component fade
// multiply onto the one button element.
func TestOpacityComposesWithDisabled(t *testing.T) {
	v := ui.Button(struct{}{}, ui.Text("x")).Disabled(true).Opacity(0.1)
	stage(t, v, func(s *uitest.Session) {
		var mods int
		s.Eval(`document.querySelectorAll("ui-box").length`, &mods)
		var btn string
		s.Eval(`getComputedStyle(document.querySelector("button")).opacity`, &btn)
		if mods != 0 || btn != "0.05" {
			t.Errorf("wrappers = %d, button opacity = %s, want none at 0.05", mods, btn)
		}
	})

	html := render(t, ui.Button(struct{}{}, ui.Text("x")).Disabled(true).Class("inner").Opacity(0.1))
	if got := strings.Count(html, "inner"); got != 1 || !strings.Contains(html, `<button class="inner `) {
		t.Errorf("modifiers should land on the button element:\n%s", html)
	}
}

// TestElementReset pins the normalize tier against the UA
// stylesheet: an element under the root has no styling of its own,
// whatever its tag, so a view's appearance is whatever its lowering
// sets. HTML view content is the exception and keeps the browser's
// styling.
func TestElementReset(t *testing.T) {
	const props = `["borderTopWidth","paddingTop","marginTop","textAlign","appearance","textDecorationLine","color","fontSize","fontWeight","display","cursor"]`
	styleOf := func(s *uitest.Session, sel string) string {
		var out string
		s.Eval(`(() => { const c = getComputedStyle(document.querySelector(`+strconv.Quote(sel)+`)); return `+props+`.map(k => c[k]).join(";") })()`, &out)
		return out
	}
	for _, tag := range []string{"button", "a", "h1", "ul", "pre", "code", "fieldset"} {
		t.Run(tag, func(t *testing.T) {
			v := ui.HStack(ui.Text("x"), ui.Text("x").Tag(tag).Attr(attr.Href("/")))
			stage(t, v, func(s *uitest.Session) {
				if plain, tagged := styleOf(s, "ui-text"), styleOf(s, tag); plain != tagged {
					t.Errorf("%s = %s\nui-text = %s", tag, tagged, plain)
				}
			})
		})
	}
	t.Run("html", func(t *testing.T) {
		stage(t, ui.HTML(html.Button()(domi.Text("x"))), func(s *uitest.Session) {
			var w string
			s.Eval(`getComputedStyle(document.querySelector("button")).borderTopWidth`, &w)
			if w == "0px" {
				t.Errorf("HTML content button border-width = %s, want the browser's", w)
			}
		})
	})
}

// TestBackgroundStacks pins the paint stack: an outer Background
// paints behind an inner one on the same element, visible where the
// inner is translucent — the outermost color as background-color,
// the inner colors as image layers listed innermost first.
func TestBackgroundStacks(t *testing.T) {
	html := render(t, ui.Text("x").Background(ui.CSSColor("#0008")).Background(ui.CSSColor("#fff")))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	if got != "background-color:#fff;background-image:linear-gradient(#0008,#0008);display:block;overflow-wrap:break-word" {
		t.Errorf("paint stack = %q, want the outer color under the inner layer:\n%s", got, html)
	}
}

// TestBackgroundShapeOrder pins the shape's write order: a shape applied after
// paint shapes it; paint applied after a shape lands outside it.
func TestBackgroundShapeOrder(t *testing.T) {
	// Background then shape: shape and paint share the element —
	// a red capsule.
	shaped := render(t, ui.Text("x").Background(ui.CSSColor("red")).BorderShape(ui.Capsule))
	if got := classRule(t, shaped, `<ui-text class="(ui-\w+)"`); got != "background-color:red;border-radius:9999px;display:block;overflow-wrap:break-word" {
		t.Errorf("shape after paint should shape the paint, got %q:\n%s", got, shaped)
	}

	// Shape then background: the shape stays on the text element and
	// the paint boxes out around it, unshaped — a red rectangle.
	square := render(t, ui.Text("x").BorderShape(ui.Capsule).Background(ui.CSSColor("red")))
	if got := classRule(t, square, `<ui-box class="(ui-\w+)"`); got != "background-color:red;display:grid;grid-template-columns:100%;grid-template-rows:100%;place-items:center" {
		t.Errorf("paint after shape should land on a wrapper, got %q:\n%s", got, square)
	}
	if got := classRule(t, square, `<ui-text class="(ui-\w+)"`); got != "border-radius:9999px;display:block;overflow-wrap:break-word" {
		t.Errorf("the shape should stay on the inner element, got %q:\n%s", got, square)
	}
}

// TestBorderShapeRepetition pins shape inheritance: the shape descends
// to the first box, so the innermost of two shapes wins and the outer
// one is inert — no wrapper, no declaration.
func TestBorderShapeRepetition(t *testing.T) {
	html := render(t, ui.Text("x").BorderShape(ui.RoundedRectangle).BorderShape(ui.Capsule))
	if got := classRule(t, html, `<ui-text class="(ui-\w+)"`); got != "border-radius:var(--ui-radius);display:block;overflow-wrap:break-word" {
		t.Errorf("innermost shape should land on the text element, got %q:\n%s", got, html)
	}
	if strings.Contains(html, "9999px") || strings.Contains(html, "ui-box") {
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
	html := render(t, ui.Text("x").BorderStroke(2, ui.CSSColor("red")))
	if strings.Contains(html, "ui-box") {
		t.Fatalf("BorderStroke should not produce a wrapper:\n%s", html)
	}
	if got := classRule(t, html, `<ui-text class="(ui-\w+)"`); got != "display:block;overflow-wrap:break-word;position:relative;"+carrier("inset 0 0 0 2px red") {
		t.Errorf("stroke rule = %q:\n%s", got, html)
	}
}

// TestBorderStrokeStacks pins the stroke stack: strokes merge onto one
// element as a shadow list, the outer stroke listed first, painting
// over the inner one.
func TestBorderStrokeStacks(t *testing.T) {
	html := render(t, ui.Text("x").BorderStroke(2, ui.CSSColor("red")).BorderStroke(4, ui.CSSColor("blue")))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	if got != "display:block;overflow-wrap:break-word;position:relative;"+carrier("inset 0 0 0 4px blue,inset 0 0 0 2px red") {
		t.Errorf("stroke stack = %q, want the outer stroke over the inner:\n%s", got, html)
	}
}

// TestBorderStrokeShapeOrder pins the shape's write order against the
// stroke: a shape applied after a stroke shapes its ring; a stroke
// applied after a shape rings the shaped box, unshaped.
func TestBorderStrokeShapeOrder(t *testing.T) {
	shaped := render(t, ui.Text("x").BorderStroke(2, ui.CSSColor("red")).BorderShape(ui.Capsule))
	if got := classRule(t, shaped, `<ui-text class="(ui-\w+)"`); got != "border-radius:9999px;display:block;overflow-wrap:break-word;position:relative;"+carrier("inset 0 0 0 2px red") {
		t.Errorf("shape after stroke should shape the stroke, got %q:\n%s", got, shaped)
	}

	square := render(t, ui.Text("x").BorderShape(ui.Capsule).BorderStroke(2, ui.CSSColor("red")))
	if got := classRule(t, square, `<ui-box class="(ui-\w+)"`); got != "display:grid;grid-template-columns:100%;grid-template-rows:100%;place-items:center;position:relative;"+carrier("inset 0 0 0 2px red") {
		t.Errorf("stroke after shape should land on a wrapper, got %q:\n%s", got, square)
	}
	if got := classRule(t, square, `<ui-text class="(ui-\w+)"`); got != "border-radius:9999px;display:block;overflow-wrap:break-word" {
		t.Errorf("the shape should stay on the inner element, got %q:\n%s", got, square)
	}
}

// TestBorderStrokeDoesNotInterceptClicks pins the carrier's
// hit-test transparency: a stroked container's ring covers its
// content without stealing its pointer events.
func TestBorderStrokeDoesNotInterceptClicks(t *testing.T) {
	v := ui.HStack(ui.Button(struct{}{}, ui.Text("click"))).BorderStroke(4, ui.CSSColor("red"))
	stage(t, v, func(s *uitest.Session) {
		var inButton bool
		s.Eval(`(() => {
			const r = document.querySelector("button").getBoundingClientRect();
			return !!document.elementFromPoint(r.x + r.width/2, r.y + r.height/2).closest("button");
		})()`, &inButton)
		if !inButton {
			t.Error("the element under the pointer is outside the button")
		}
	})
}

// TestBorderStrokeOnImage pins the replaced-element accommodation: an
// img cannot host the carrier, so the strokes box out around it.
func TestBorderStrokeOnImage(t *testing.T) {
	html := render(t, ui.Image("/x.png").BorderStroke(2, ui.CSSColor("red")))
	if got := classRule(t, html, `<ui-box class="(ui-\w+)"`); got != "display:grid;grid-template-columns:100%;grid-template-rows:100%;place-items:center;position:relative;"+carrier("inset 0 0 0 2px red") {
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
	html := render(t, ui.ScrollView(ui.Vertical, ui.Text("x")).BorderStroke(2, ui.CSSColor("red")))
	if got := classRule(t, html, `<ui-box class="[^"]*(ui-\w+)"`); got != "align-self:stretch;display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-self:stretch;place-items:center;position:relative;"+carrier("inset 0 0 0 2px red") {
		t.Errorf("scroll strokes should land on a wrapper, got %q:\n%s", got, html)
	}
	if !strings.Contains(html, `<ui-scroll `) {
		t.Errorf("the viewport should render inside:\n%s", html)
	}
}

// TestZStackPaintsInOrder pins the ZStack's paint order in the
// browser: a later subview paints over an earlier one, even when the
// earlier one forms a stacking context, as a translucent one does,
// and the later one does not.
func TestZStackPaintsInOrder(t *testing.T) {
	v := ui.ZStack(
		ui.CSSColor("blue").Frame(ui.Width(120), ui.Height(60)).Opacity(0.5),
		ui.Text("over").Class("over"),
	)
	stage(t, v, func(s *uitest.Session) {
		var onTop bool
		s.Eval(`(() => {
			const r = document.querySelector(".over").getBoundingClientRect();
			return !!document.elementFromPoint(r.x + r.width/2, r.y + r.height/2).closest(".over");
		})()`, &onTop)
		if !onTop {
			t.Error("the earlier subview paints over the later text")
		}
	})
}

// TestLayerIsolatesSubview pins the subview's stacking isolation:
// the subview forms its own stacking context, so no z-index inside
// it — app CSS included — can climb the composite's z ladder past
// the layers.
func TestLayerIsolatesSubview(t *testing.T) {
	html := render(t, ui.Text("x").Overlay(ui.Center, ui.Text("o")))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	if got != "display:block;isolation:isolate;overflow-wrap:break-word" {
		t.Errorf("layered subview rule = %q, want isolation", got)
	}
}

// TestBorderStrokeOverLayers pins the ring against the layer
// composite: on a layered box the carrier joins the z ladder above
// the overlay, where its tree position alone would lose to the
// layers' indexes.
func TestBorderStrokeOverLayers(t *testing.T) {
	html := render(t, ui.Text("x").Overlay(ui.Center, ui.Text("o")).BorderStroke(2, ui.CSSColor("red")))
	got := classRule(t, html, `<ui-layer class="(ui-\w+)"`)
	want := "display:grid;grid-template-columns:100%;grid-template-rows:100%;" +
		"isolation:isolate;overflow:visible;place-items:center;position:relative;" +
		strings.Replace(carrier("inset 0 0 0 2px red"), "position:absolute}", "position:absolute;z-index:3}", 1)
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
		ui.Text("hello").Class("stroked").BorderStroke(4, ui.CSSColor("red")),
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
	html := render(t, ui.CSSColor("red").BorderShape(ui.Ellipse))
	if got := classRule(t, html, `<ui-color class="[^"]*(ui-\w+)"`); got != "align-self:stretch;background-color:red;border-radius:50%;justify-self:stretch" {
		t.Errorf("shape should land on the color's element, got %q:\n%s", got, html)
	}
}

// TestWrapperKeepsRigidity pins the layout transparency of wrappers:
// a wrapper forwards its subview's rigid axes, so it resists flex
// compression on the subview's behalf. A frame does the same on an
// auto axis, which takes the subview's sizing.
func TestWrapperKeepsRigidity(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       ui.View
		pattern string
	}{
		{"transform box", ui.HStack(ui.Text("x").FixedSize().Opacity(0.5).Background(ui.CSSColor("red"))), `<ui-box class="(ui-\w+)"`},
		{"frame auto axis", ui.HStack(ui.Text("x").FixedSize().Frame(ui.Height(40))), `<ui-frame class="(ui-\w+)"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			if got := classRule(t, html, tt.pattern); !strings.Contains(got, "flex-shrink:0") {
				t.Errorf("wrapper should carry its subview's rigidity, got %q:\n%s", got, html)
			}
		})
	}
}

// TestTextStyleInnermostWins pins the whole-text modifiers to the same
// rule as their generic counterparts: the first (innermost) value wins.
func TestTextStyleInnermostWins(t *testing.T) {
	html := render(t, ui.Text("x").TextForeground(ui.CSSColor("#111")).TextForeground(ui.CSSColor("#222")))
	if !strings.Contains(html, "color:#111") || strings.Contains(html, "#222") {
		t.Errorf("repeated TextForeground should keep the first color:\n%s", html)
	}
	html = render(t, ui.Text("x").TextFont(ui.Title).TextFont(ui.Caption))
	if !strings.Contains(html, "font-size:1.5rem") || strings.Contains(html, "font-size:0.75rem") {
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
	_, page := new(ui.Renderer).Render(v)
	if err := domi.RenderTo(&sb, page); err != nil {
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
		s.Eval(`getComputedStyle(document.querySelector("ui-html")).alignItems`, &align)
		s.Eval(`getComputedStyle(document.querySelector("ui-html")).justifyItems`, &justify)
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
