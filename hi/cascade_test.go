package hi_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/hi"
	"ily.dev/act3/hi/internal/uitest"
)

// TestForegroundInnermostWins pins the wrapper model for inherited
// modifiers: the Foreground closest to the content styles it, with or
// without structure in between.
func TestForegroundInnermostWins(t *testing.T) {
	for _, tt := range []struct {
		name string
		v    hi.View
	}{
		{"adjacent", hi.Text("hi").Foreground(hi.Red).Foreground(hi.Blue)},
		{"frame between", hi.Text("hi").Foreground(hi.Red).Frame(hi.Width(100)).Foreground(hi.Blue)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, tt.v, func(s *uitest.Session) {
				var color string
				s.Eval(`getComputedStyle(document.querySelector("hi-text")).color`, &color)
				if color != redCSS {
					t.Errorf("text color = %s, want %s", color, redCSS)
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
	html := render(t, hi.VStack(hi.Text("a"), hi.Text("b")).Foreground(hi.Red))
	if strings.Contains(html, "hi-box") {
		t.Fatalf("Foreground should not produce a wrapper:\n%s", html)
	}
	if got := classRule(t, html, `<hi-vstack class="(hi-\w+)"`); got != "align-items:center;color:"+redCSS+";column-gap:8px;display:inline-flex;flex-direction:column;row-gap:8px" {
		t.Errorf("stack box rule = %q, want the consumed color in the stack's own set", got)
	}
	m := regexp.MustCompile(`\.(hi-\w+)\{align-items:center;color:` + regexp.QuoteMeta(redCSS) + `;column-gap:8px;display:inline-flex;flex-direction:column;row-gap:8px\}`).FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no color rule in the sheet:\n%s", html)
	}
	if n := strings.Count(html, m[1]); n != 2 { // the rule and one use
		t.Errorf("consumed color class appears %d times, want 2:\n%s", n, html)
	}
}

// Explicit label paint is preserved inside a button.
func TestButtonLabelForeground(t *testing.T) {
	v := hi.Button(struct{}{}, hi.Text("x").Foreground(hi.Red))
	stage(t, v, func(s *uitest.Session) {
		var color string
		s.Eval(`getComputedStyle(document.querySelector("button hi-text")).color`, &color)
		if color != redCSS {
			t.Errorf("label color = %s, want the label's %s", color, redCSS)
		}
	})
}

// TestDisabledStateMatchesARIA pins that Disabled is active on a
// control disabled either natively or by aria-disabled, so views
// lowered to elements without a disabled attribute take part.
func TestDisabledStateMatchesARIA(t *testing.T) {
	red := hi.Foreground(hi.Red)
	for name, v := range map[string]hi.View{
		"native": hi.Text("x").Tag("button").Attr(attr.Disabled(true)).WhileDisabled(red),
		"aria":   hi.Text("x").Attr(domi.Name("aria-disabled", "true")).WhileDisabled(red),
		"none":   hi.Text("x").Tag("button").WhileDisabled(red),
	} {
		t.Run(name, func(t *testing.T) {
			stage(t, v, func(s *uitest.Session) {
				var color string
				s.Eval(`getComputedStyle(document.querySelector("hi-text, button")).color`, &color)
				if want := name != "none"; (color == redCSS) != want {
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
	stage(t, hi.Text("hi").Opacity(0.5).Opacity(0.5), func(s *uitest.Session) {
		var mods int
		s.Eval(`document.querySelectorAll("hi-box").length`, &mods)
		var op string
		s.Eval(`getComputedStyle(document.querySelector("hi-text")).opacity`, &op)
		if mods != 0 || op != "0.25" {
			t.Errorf("wrappers = %d, text opacity = %s, want none at 0.25", mods, op)
		}
	})

	if html := render(t, hi.Text("x").Opacity(1)); strings.Contains(html, "opacity") || strings.Contains(html, "hi-box") {
		t.Errorf("Opacity(1) should render nothing extra:\n%s", html)
	}
}

// TestButtonDisabledState pins that a disabled button enters the
// Disabled state whatever its action, and that an enabled one does not.
func TestButtonDisabledState(t *testing.T) {
	paint := hi.Background(hi.Red)
	for name, tc := range map[string]struct {
		v    hi.View
		want bool
	}{
		"send":              {hi.Button(struct{}{}, hi.Text("x")).WhileDisabled(paint), false},
		"send disabled":     {hi.Button(struct{}{}, hi.Text("x")).Disabled(true).WhileDisabled(paint), true},
		"navigate":          {hi.Button("/x", hi.Text("x")).WhileDisabled(paint), false},
		"navigate disabled": {hi.Button("/x", hi.Text("x")).Disabled(true).WhileDisabled(paint), true},
	} {
		t.Run(name, func(t *testing.T) {
			stage(t, tc.v, func(s *uitest.Session) {
				var color string
				s.Eval(`getComputedStyle(document.querySelector("button, a")).backgroundColor`, &color)
				if got := color == redCSS; got != tc.want {
					t.Errorf("background = %s, want disabled styling = %v", color, tc.want)
				}
			})
		})
	}
}

// TestLinkDisabledState pins that a link used as a box is the
// element that performs its action, so a disabled one enters the
// Disabled state whatever its action, and an enabled one does not.
func TestLinkDisabledState(t *testing.T) {
	red := hi.Background(hi.Red)
	for name, tc := range map[string]struct {
		v    hi.View
		want bool
	}{
		"send":              {hi.Link(struct{}{}, hi.Text("x")).WhileDisabled(red), false},
		"send disabled":     {hi.Link(struct{}{}, hi.Text("x")).Disabled(true).WhileDisabled(red), true},
		"navigate":          {hi.Link("/x", hi.Text("x")).WhileDisabled(red), false},
		"navigate disabled": {hi.Link("/x", hi.Text("x")).Disabled(true).WhileDisabled(red), true},
	} {
		t.Run(name, func(t *testing.T) {
			stage(t, tc.v, func(s *uitest.Session) {
				var color string
				s.Eval(`getComputedStyle(document.querySelector("button, a")).backgroundColor`, &color)
				if got := color == redCSS; got != tc.want {
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
	button := hi.Button(struct{}{}, hi.Text("x")).Disabled(true)
	stage(t, hi.HStack(button, button.Opacity(0.1)), func(s *uitest.Session) {
		var mods int
		s.Eval(`document.querySelectorAll("hi-box").length`, &mods)
		var opacities []float64
		s.Eval(`Array.from(document.querySelectorAll("button"), e => Number(getComputedStyle(e).opacity))`, &opacities)
		within(t, "authored opacity composes with the component's", opacities[1], opacities[0]*0.1, 0.0001)
		if mods != 0 {
			t.Errorf("wrappers = %d, want none", mods)
		}
	})

	html := render(t, hi.Button(struct{}{}, hi.Text("x")).Disabled(true).Class("inner").Opacity(0.1))
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
			v := hi.HStack(hi.Text("x"), hi.Text("x").Tag(tag).Attr(attr.Href("/")))
			stage(t, v, func(s *uitest.Session) {
				if plain, tagged := styleOf(s, "hi-text"), styleOf(s, tag); plain != tagged {
					t.Errorf("%s = %s\nhi-text = %s", tag, tagged, plain)
				}
			})
		})
	}
	t.Run("html", func(t *testing.T) {
		stage(t, hi.HTML(html.Button()(domi.Text("x"))), func(s *uitest.Session) {
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
	html := render(t, hi.Text("x").Background(hi.OKLCHA(0, 0, 0, 0.5)).Background(hi.White))
	got := classRule(t, html, `<hi-text class="(hi-\w+)"`)
	if got != "background-color:"+whiteCSS+";background-image:linear-gradient(oklch(0 0 0 / 0.5),oklch(0 0 0 / 0.5));display:block;overflow-wrap:break-word" {
		t.Errorf("paint stack = %q, want the outer color under the inner layer:\n%s", got, html)
	}
}

// TestBackgroundShapeOrder pins the shape's write order: a shape applied after
// paint shapes it; paint applied after a shape lands outside it.
func TestBackgroundShapeOrder(t *testing.T) {
	// Background then shape: shape and paint share the element —
	// a red capsule.
	shaped := render(t, hi.Text("x").Background(hi.Red).BorderShape(hi.Capsule))
	if got := classRule(t, shaped, `<hi-text class="(hi-\w+)"`); got != "background-color:"+redCSS+";border-radius:9999px;display:block;overflow-wrap:break-word" {
		t.Errorf("shape after paint should shape the paint, got %q:\n%s", got, shaped)
	}

	// Shape then background: the shape stays on the text element and
	// the paint boxes out around it, unshaped — a red rectangle.
	square := render(t, hi.Text("x").BorderShape(hi.Capsule).Background(hi.Red))
	if got := classRule(t, square, `<hi-box class="(hi-\w+)"`); got != "align-items:center;background-color:"+redCSS+";display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-items:center" {
		t.Errorf("paint after shape should land on a wrapper, got %q:\n%s", got, square)
	}
	if got := classRule(t, square, `<hi-text class="(hi-\w+)"`); got != "border-radius:9999px;display:block;overflow-wrap:break-word" {
		t.Errorf("the shape should stay on the inner element, got %q:\n%s", got, square)
	}
}

// TestBorderShapeRepetition pins shape inheritance: the shape descends
// to the first box, so the innermost of two shapes wins and the outer
// one is inert — no wrapper, no declaration.
func TestBorderShapeRepetition(t *testing.T) {
	html := render(t, hi.Text("x").BorderShape(hi.RoundedRectangle).BorderShape(hi.Capsule))
	if got := classRule(t, html, `<hi-text class="(hi-\w+)"`); got != "border-radius:var(--hi-radius);display:block;overflow-wrap:break-word" {
		t.Errorf("innermost shape should land on the text element, got %q:\n%s", got, html)
	}
	if strings.Contains(html, "9999px") || strings.Contains(html, "hi-box") {
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
	html := render(t, hi.Text("x").BorderStroke(2, hi.Red))
	if strings.Contains(html, "hi-box") {
		t.Fatalf("BorderStroke should not produce a wrapper:\n%s", html)
	}
	if got := classRule(t, html, `<hi-text class="(hi-\w+)"`); got != "display:block;overflow-wrap:break-word;position:relative;"+carrier("inset 0 0 0 2px "+redCSS) {
		t.Errorf("stroke rule = %q:\n%s", got, html)
	}
}

// TestBorderStrokeStacks pins the stroke stack: strokes merge onto one
// element as a shadow list, the outer stroke listed first, painting
// over the inner one.
func TestBorderStrokeStacks(t *testing.T) {
	html := render(t, hi.Text("x").BorderStroke(2, hi.Red).BorderStroke(4, hi.Blue))
	got := classRule(t, html, `<hi-text class="(hi-\w+)"`)
	if got != "display:block;overflow-wrap:break-word;position:relative;"+carrier("inset 0 0 0 4px "+blueCSS+",inset 0 0 0 2px "+redCSS) {
		t.Errorf("stroke stack = %q, want the outer stroke over the inner:\n%s", got, html)
	}
}

// TestBorderStrokeShapeOrder pins the shape's write order against the
// stroke: a shape applied after a stroke shapes its ring; a stroke
// applied after a shape rings the shaped box, unshaped.
func TestBorderStrokeShapeOrder(t *testing.T) {
	shaped := render(t, hi.Text("x").BorderStroke(2, hi.Red).BorderShape(hi.Capsule))
	if got := classRule(t, shaped, `<hi-text class="(hi-\w+)"`); got != "border-radius:9999px;display:block;overflow-wrap:break-word;position:relative;"+carrier("inset 0 0 0 2px "+redCSS) {
		t.Errorf("shape after stroke should shape the stroke, got %q:\n%s", got, shaped)
	}

	square := render(t, hi.Text("x").BorderShape(hi.Capsule).BorderStroke(2, hi.Red))
	if got := classRule(t, square, `<hi-box class="(hi-\w+)"`); got != "align-items:center;display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-items:center;position:relative;"+carrier("inset 0 0 0 2px "+redCSS) {
		t.Errorf("stroke after shape should land on a wrapper, got %q:\n%s", got, square)
	}
	if got := classRule(t, square, `<hi-text class="(hi-\w+)"`); got != "border-radius:9999px;display:block;overflow-wrap:break-word" {
		t.Errorf("the shape should stay on the inner element, got %q:\n%s", got, square)
	}
}

// TestBorderStrokeDoesNotInterceptClicks pins the carrier's
// hit-test transparency: a stroked container's ring covers its
// content without stealing its pointer events.
func TestBorderStrokeDoesNotInterceptClicks(t *testing.T) {
	v := hi.HStack(hi.Button(struct{}{}, hi.Text("click"))).BorderStroke(4, hi.Red)
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
	html := render(t, hi.Image("/x.png").BorderStroke(2, hi.Red))
	if got := classRule(t, html, `<hi-box class="(hi-\w+)"`); got != "align-items:center;display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-items:center;position:relative;"+carrier("inset 0 0 0 2px "+redCSS) {
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
	html := render(t, hi.ScrollView(hi.Vertical, hi.Text("x")).BorderStroke(2, hi.Red))
	if got := classRule(t, html, `<hi-box class="[^"]*(hi-\w+)"`); got != "align-items:center;align-self:stretch;display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-items:center;justify-self:stretch;position:relative;"+carrier("inset 0 0 0 2px "+redCSS) {
		t.Errorf("scroll strokes should land on a wrapper, got %q:\n%s", got, html)
	}
	if !strings.Contains(html, `<hi-scroll `) {
		t.Errorf("the viewport should render inside:\n%s", html)
	}
}

// TestZStackPaintsInOrder pins the ZStack's paint order in the
// browser: a later subview paints over an earlier one, even when the
// earlier one forms a stacking context, as a translucent one does,
// and the later one does not.
func TestZStackPaintsInOrder(t *testing.T) {
	v := hi.ZStack(
		hi.Blue.Frame(hi.Width(120), hi.Height(60)).Opacity(0.5),
		hi.Text("over").Class("over"),
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
	html := render(t, hi.Text("x").Overlay(hi.Center, hi.Text("o")))
	got := classRule(t, html, `<hi-text class="(hi-\w+)"`)
	if got != "display:block;isolation:isolate;overflow-wrap:break-word" {
		t.Errorf("layered subview rule = %q, want isolation", got)
	}
}

// TestBorderStrokeOverLayers pins the ring against the layer
// composite: on a layered box the carrier joins the z ladder above
// the overlay, where its tree position alone would lose to the
// layers' indexes.
func TestBorderStrokeOverLayers(t *testing.T) {
	html := render(t, hi.Text("x").Overlay(hi.Center, hi.Text("o")).BorderStroke(2, hi.Red))
	got := classRule(t, html, `<hi-layer class="(hi-\w+)"`)
	want := "align-items:center;display:grid;grid-template-columns:100%;grid-template-rows:100%;" +
		"isolation:isolate;justify-items:center;position:relative;" +
		strings.Replace(carrier("inset 0 0 0 2px "+redCSS), "position:absolute}", "position:absolute;z-index:3}", 1)
	if got != want {
		t.Errorf("layered stroke rule = %q, want the ring on the z ladder:\n%s", got, html)
	}
}

// TestBorderStrokeTakesNoSpace pins the stroke's layout contract in
// the browser: the carrier draws the ring, and the stroked box is the
// same size as an unstroked one.
func TestBorderStrokeTakesNoSpace(t *testing.T) {
	v := hi.VStack(
		hi.Text("hello").Class("plain"),
		hi.Text("hello").Class("stroked").BorderStroke(4, hi.Red),
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
	html := render(t, hi.Red.BorderShape(hi.Ellipse))
	if got := classRule(t, html, `<hi-color class="[^"]*(hi-\w+)"`); got != "align-self:stretch;background-color:"+redCSS+";border-radius:50%;justify-self:stretch" {
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
		v       hi.View
		pattern string
	}{
		{"transform box", hi.HStack(hi.Text("x").FixedSize().Opacity(0.5).Background(hi.Red)), `<hi-box class="(hi-\w+)"`},
		{"frame auto axis", hi.HStack(hi.Text("x").FixedSize().Frame(hi.Height(40))), `<hi-frame class="(hi-\w+)"`},
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
	html := render(t, hi.Text("x").TextForeground(hi.OKLCH(0.1, 0, 0)).TextForeground(hi.OKLCH(0.2, 0, 0)))
	if !strings.Contains(html, "color:oklch(0.1 0 0)") || strings.Contains(html, "oklch(0.2 0 0)") {
		t.Errorf("repeated TextForeground should keep the first color:\n%s", html)
	}
}

// stageApp is stage with unlayered app CSS placed before the hi stylesheet in the
// document, so an app rule can win only through the hi cascade layer,
// never through source order.
func stageApp(t *testing.T, appCSS string, v hi.View, fn func(*uitest.Session)) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("<!doctype html><meta charset=utf-8><style>")
	sb.WriteString(appCSS)
	sb.WriteString("</style><style>")
	sb.WriteString(staticCSS)
	sb.WriteString(`</style><body>`)
	_, page := hi.Render(v)
	if err := domi.RenderTo(&sb, page); err != nil {
		t.Fatalf("render: %v", err)
	}
	uitest.Run(t, 600, 400, sb.String(), fn)
}

// TestAppCSSBeatsStaticSheet pins the app-vs-hi contract for hi.css:
// every rule it emits sits in the hi layer, so an unlayered app class
// overrides it at equal specificity regardless of source order. The
// fixture is the one HTML's documentation invites: an app class
// restyling the host adapter's interior layout.
func TestAppCSSBeatsStaticSheet(t *testing.T) {
	v := hi.HTML(domi.Text("hi")).Class("app-host")
	stageApp(t, ".app-host{place-items:stretch}", v, func(s *uitest.Session) {
		var align, justify string
		s.Eval(`getComputedStyle(document.querySelector("hi-html")).alignItems`, &align)
		s.Eval(`getComputedStyle(document.querySelector("hi-html")).justifyItems`, &justify)
		if align != "stretch" || justify != "stretch" {
			t.Errorf("place-items = %s %s, want the app's stretch stretch", align, justify)
		}
	})
}

// TestAppCSSBeatsDynamicSheet pins the same contract for the
// render-time hashed sheet, whose style element follows the app's in
// the document and would otherwise win by source order.
func TestAppCSSBeatsDynamicSheet(t *testing.T) {
	v := hi.Text("hi").Padding(hi.Edges(16)).Class("app-pad")
	stageApp(t, ".app-pad{padding:0}", v, func(s *uitest.Session) {
		var pad string
		s.Eval(`getComputedStyle(document.querySelector(".app-pad")).paddingTop`, &pad)
		if pad != "0px" {
			t.Errorf("padding-top = %s, want the app's 0px", pad)
		}
	})
}

// TestBorderStrokeZeroWidthKeepsStructure pins the stability rule
// against the stroke accommodations: a stroke that draws nothing
// still boxes out around an image, so the lowering does not depend
// on the width.
func TestBorderStrokeZeroWidthKeepsStructure(t *testing.T) {
	for _, px := range []float64{0, -1} {
		html := render(t, hi.Image("/x.png").BorderStroke(complex(px, 0), hi.Red))
		if !strings.Contains(html, "<hi-box ") {
			t.Errorf("BorderStroke(%g) should keep the wrapper:\n%s", px, html)
		}
	}
}

// TestBorderClippedTransforms pins the clip as a transform: applied
// outside a stroke, it lands on the view's own element with the stroke
// and the shape, in either order, with no wrapper.
func TestBorderClippedTransforms(t *testing.T) {
	want := "border-radius:9999px;display:block;overflow-wrap:break-word;overflow-x:clip;overflow-y:clip;position:relative;" + carrier("inset 0 0 0 2px "+redCSS)
	for name, v := range map[string]hi.View{
		"clip then shape": hi.Text("x").BorderStroke(2, hi.Red).BorderClipped().BorderShape(hi.Capsule),
		"shape then clip": hi.Text("x").BorderStroke(2, hi.Red).BorderShape(hi.Capsule).BorderClipped(),
	} {
		html := render(t, v)
		if strings.Contains(html, "hi-box") {
			t.Errorf("%s: should not produce a wrapper:\n%s", name, html)
		}
		if got := classRule(t, html, `<hi-text class="(hi-\w+)"`); got != want {
			t.Errorf("%s: rule = %q:\n%s", name, got, html)
		}
	}
}

// TestBorderClippedStrokeOrder pins the clip's write order against
// the stroke, as for the shape: a stroke outside a clip lands on a
// wrapper, and rings that wrapper, which is not clipped.
func TestBorderClippedStrokeOrder(t *testing.T) {
	html := render(t, hi.Text("x").BorderClipped().BorderStroke(2, hi.Red))
	if got := classRule(t, html, `<hi-box class="(hi-\w+)"`); got != "align-items:center;display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-items:center;position:relative;"+carrier("inset 0 0 0 2px "+redCSS) {
		t.Errorf("stroke outside clip should land on a wrapper, got %q:\n%s", got, html)
	}
	if got := classRule(t, html, `<hi-text class="(hi-\w+)"`); got != "display:block;overflow-wrap:break-word;overflow-x:clip;overflow-y:clip" {
		t.Errorf("the clip should stay on the inner element, got %q:\n%s", got, html)
	}
}

// TestBorderClippedOnScroll pins the scroll viewport against a clip:
// the viewport already confines its content, so its own overflow
// wins, and nothing boxes out.
func TestBorderClippedOnScroll(t *testing.T) {
	plain := render(t, hi.ScrollView(hi.Vertical, hi.Text("x")).Padding(hi.Edges(0)))
	clipped := render(t, hi.ScrollView(hi.Vertical, hi.Text("x")).BorderClipped().Padding(hi.Edges(0)))
	if clipped != plain {
		t.Errorf("clip changed the scroll lowering:\nwant %s\ngot  %s", plain, clipped)
	}
}

// TestBorderClippedOverLayers pins the clip against the layer
// composite: a clip outside the layers confines them too, and one
// inside confines only the base.
func TestBorderClippedOverLayers(t *testing.T) {
	outside := render(t, hi.Text("x").Overlay(hi.Center, hi.Text("o")).BorderClipped())
	if got := classRule(t, outside, `<hi-layer class="(hi-\w+)"`); !strings.Contains(got, "overflow-x:clip;overflow-y:clip") {
		t.Errorf("clip outside overlay should clip the composite, got %q:\n%s", got, outside)
	}

	inside := render(t, hi.Text("x").BorderClipped().Overlay(hi.Center, hi.Text("o")).Padding(hi.Edges(0)))
	if got := classRule(t, inside, `<hi-layer class="(hi-\w+)"`); strings.Contains(got, "clip") {
		t.Errorf("clip inside overlay should leave the composite unclipped, got %q:\n%s", got, inside)
	}
	if got := classRule(t, inside, `<hi-text class="(hi-\w+)"`); !strings.Contains(got, "overflow-x:clip;overflow-y:clip") {
		t.Errorf("clip inside overlay should clip the base, got %q:\n%s", got, inside)
	}
}
