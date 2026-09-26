package hi

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"

	"ily.dev/act3/hi/internal/sheet"
	"ily.dev/act3/hi/internal/uitest"
)

func TestPresentationSurfaceTheme(t *testing.T) {
	for _, tc := range []struct {
		name        string
		root, local Color
	}{
		{"light root dark local", OKLCH(.95, .02, 80), OKLCH(.2, .03, 215)},
		{"dark root light local", OKLCH(.2, .03, 215), OKLCH(.95, .02, 80)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rootDialog, localNote, localPopover, localDialog, nestedPopover environment
			probe := func(env *environment) View { return view(envProbe(env)) }

			v := VStack(
				Text("Trigger").Dialog(Present(false, 7), probe(&rootDialog)).Class("root-dialog"),
				VStack(
					note{Note: Note{Message: "Note", Action: probe(&localNote)}}.view().Class("note-reference"),
					Text("Trigger").Menu(Present(false, 7), probe(&localPopover)).Class("local-popover"),
					Text("Trigger").Dialog(Present(false, 7), VStack(probe(&localDialog),
						Text("Nested trigger").Popover(Present(false, 7), probe(&nestedPopover)))).
						Class("local-dialog"),
				).ThemeBackground(tc.local),
			)
			_, page := Render(v, Theme(tc.root, OKLCH(.6, .2, 10), 45))
			if localPopover.theme != localNote.theme {
				t.Errorf("popover content theme differs from local note: %+v != %+v", localPopover.theme, localNote.theme)
			}
			if localDialog.theme != rootDialog.theme {
				t.Errorf("local ThemeBackground changed dialog theme: %+v != %+v", localDialog.theme, rootDialog.theme)
			}
			if localDialog.theme.bgbase.isLight() != tc.root.color().colorCoords(defaultTheme).isLight() {
				t.Error("dialog did not inherit the page's light/dark mode")
			}
			if localPopover.theme.bgbase.isLight() != tc.local.color().colorCoords(defaultTheme).isLight() {
				t.Error("popover did not inherit the local light/dark mode")
			}
			wantNested := localDialog.theme
			wantNested.bgbase = menuColor.color().colorCoords(wantNested)
			if nestedPopover.theme != wantNested {
				t.Errorf("nested popover theme = %+v, want derived from dialog %+v", nestedPopover.theme, wantNested)
			}
			fixture := "<!doctype html><style>" + string(staticCSS) + "</style><body>" + renderNode(t, page)
			uitest.Run(t, 800, 600, fixture, func(s *uitest.Session) {
				var paints map[string]map[string]string
				s.Eval(`(() => {
					const paint = selector => {
						if (selector.endsWith('-dialog')) selector += ' + dialog > hi-zstack';
						if (selector === '.local-popover') selector += ' [popover] hi-zstack';
						const e = document.querySelector(selector), s = getComputedStyle(e);
						return {background: s.backgroundColor, color: s.color, scheme: s.colorScheme,
							radius: s.borderRadius, shadow: s.boxShadow,
							before: getComputedStyle(e, '::before').boxShadow,
							after: getComputedStyle(e, '::after').boxShadow};
					};
					return Object.fromEntries(['root-dialog', 'local-dialog', 'note-reference',
						'local-popover'].
						map(name => [name, paint('.' + name)]));
				})()`, &paints)
				if radius := paints["local-popover"]["radius"]; radius != "11px" {
					t.Errorf("popover radius = %q, want 11px", radius)
				}
				delete(paints["local-popover"], "radius")
				delete(paints["note-reference"], "radius")
				for _, pair := range [][2]string{{"local-popover", "note-reference"}, {"local-dialog", "root-dialog"}} {
					if !reflect.DeepEqual(paints[pair[0]], paints[pair[1]]) {
						t.Errorf("%s paint differs from %s: %+v != %+v", pair[0], pair[1], paints[pair[0]], paints[pair[1]])
					}
				}
			})
		})
	}
}

func TestPresentationAttachmentLayout(t *testing.T) {
	for _, tc := range []struct {
		name string
		view View
	}{
		{"rigid", Text("Receiver").Frame(Width(80), Height(40))},
		{"filling", Blue},
		{"scrolling", ScrollView(Vertical, Text("Content").Frame(Height(1000)))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sh sheet.Sheet
			env := environment{sheet: &sh, theme: defaultTheme, canPresent: true}
			before := unary(VStack, tc.view)(env)
			for _, attached := range []View{tc.view.Menu(Present(true, 7), Red), tc.view.Dialog(Present(true, 7), Red)} {
				after := unary(VStack, attached)(env)
				if after.fills != before.fills || after.rigid != before.rigid {
					t.Errorf("attachment changed receiver layout: fills=%v rigid=%v, want fills=%v rigid=%v",
						after.fills, after.rigid, before.fills, before.rigid)
				}
			}
			if tc.name == "scrolling" {
				_, page := Render(tc.view.Menu(Present(false, 7), Text("Menu")))
				html := renderNode(t, page)
				if !strings.Contains(html, "<hi-scroll ") || strings.Contains(html, ` scroll="y"`) {
					t.Error("root attachment should retain an ordinary ScrollView viewport")
				}
			}
		})
	}
}

func TestPresentationAttachmentGeometry(t *testing.T) {
	points := []Alignment{TopLeading, Top, TopTrailing, Leading, Center, Trailing,
		BottomLeading, Bottom, BottomTrailing}
	for _, direction := range []string{"ltr", "rtl"} {
		t.Run(direction, func(t *testing.T) {
			var sh sheet.Sheet
			var content strings.Builder
			for _, fallback := range []bool{false, true} {
				for _, at := range points {
					for _, anchor := range points {
						decls := popoverAttachmentStyle(at, anchor).Decls()
						if fallback {
							name := sh.PositionTryFor(decls)
							decls.Set("position-try-fallbacks", name)
							decls.Set("inset-block-start", "10000px")
							decls.Set("inset-block-end", "0px")
						}
						class := sh.ClassFor(decls)
						fmt.Fprintf(&content, `<div popover="manual" class="%s"></div>`, class)
					}
				}
			}
			fixture := `<style>
				.scope {position:fixed;left:340px;top:260px;width:120px;height:80px;
					anchor-name:--hi-attachment;anchor-scope:--hi-attachment;overflow:hidden}
				[popover] {position:fixed;position-anchor:--hi-attachment;margin:0;padding:0;border:0;
					width:60px;height:40px;max-width:none;max-height:none}
				` + sh.CSS() + `</style><div dir="` + direction + `" class="scope"><div>` +
				content.String() + `</div></div>`
			uitest.Run(t, 800, 600, fixture, func(s *uitest.Session) {
				s.Eval(`document.querySelectorAll('[popover]').forEach(e => e.showPopover())`, nil)
				var rects []uitest.Rect
				s.Eval(`Array.from(document.querySelectorAll('[popover]'), e => {
					const r=e.getBoundingClientRect();return {X:r.x,Y:r.y,W:r.width,H:r.height};})`, &rects)
				for k, r := range rects {
					i, j := k/len(points)%len(points), k%len(points)
					at, anchor := points[i], points[j]
					x, ownX := float64(at.horizontal().point())/100, float64(anchor.horizontal().point())/100
					if direction == "rtl" {
						x, ownX = 1-x, 1-ownX
					}
					wantX := 340 + 120*x - 60*ownX
					wantY := 260 + .8*float64(at.vertical().point()) - .4*float64(anchor.vertical().point())
					if math.Abs(r.X-wantX) > .1 || math.Abs(r.Y-wantY) > .1 || r.W != 60 || r.H != 40 {
						t.Errorf("attachment (%v,%v), fallback=%v: got %+v, want x=%v y=%v w=60 h=40", at, anchor, k >= 81, r, wantX, wantY)
					}
				}
			})
		})
	}
}

func TestPresentationAvailableSpace(t *testing.T) {
	for _, test := range []struct {
		name         string
		top          int
		fills        AxisSet
		wantY, wantH float64
	}{
		{"more above", 400, Vertical, 0, 400},
		{"more below", 100, Vertical, 140, 460},
		{"both axes prefer height", 400, Horizontal | Vertical, 0, 400},
		{"equal space prefers authored", 280, Vertical, 320, 280},
		{"nonfill preserves authored when it fits", 350, 0, 390, 180},
		{"nonfill flips when authored overflows", 400, 0, 220, 180},
	} {
		t.Run(test.name, func(t *testing.T) {
			var sh sheet.Sheet
			class := sh.ClassFor(popoverPositionStyle(BottomLeading, TopLeading, test.fills, &sh).Decls())
			height := "180px"
			if test.fills.hasAny(Vertical) {
				height = "stretch"
			}
			fixture := fmt.Sprintf(`<style>
				.scope {position:fixed;left:300px;top:%dpx;width:80px;height:40px;
					anchor-name:--hi-attachment;anchor-scope:--hi-attachment;overflow:hidden}
				[popover] {position:fixed;margin:0;padding:0;border:0;width:180px;height:%s;
					max-width:none;max-height:none;overflow:auto}
				.content {height:1000px}
				%s</style><div class="scope"><div popover="manual" class="%s"><div class="content"></div></div></div>`,
				test.top, height, sh.CSS(), class)
			uitest.Run(t, 800, 600, fixture, func(s *uitest.Session) {
				s.Eval(`document.querySelector('[popover]').showPopover()`, nil)
				r := s.Rect("[popover]", 0)
				if math.Abs(r.Y-test.wantY) > .1 || math.Abs(r.H-test.wantH) > .1 {
					t.Errorf("got %+v, want y=%v h=%v", r, test.wantY, test.wantH)
				}
				var scrollHeight float64
				s.Eval(`document.querySelector('[popover]').scrollHeight`, &scrollHeight)
				if scrollHeight != 1000 {
					t.Errorf("scroll height = %v, want 1000", scrollHeight)
				}
			})
		})
	}
}

func TestPresentationHorizontalSpace(t *testing.T) {
	for _, test := range []struct {
		name         string
		left         int
		wantX, wantW float64
	}{
		{"more left", 600, 0, 600},
		{"more right", 100, 140, 660},
		{"equal space prefers authored", 380, 420, 380},
	} {
		t.Run(test.name, func(t *testing.T) {
			var sh sheet.Sheet
			class := sh.ClassFor(popoverPositionStyle(TopTrailing, TopLeading, Horizontal, &sh).Decls())
			fixture := fmt.Sprintf(`<style>
				.scope {position:fixed;left:%dpx;top:260px;width:40px;height:40px;
					anchor-name:--hi-attachment;anchor-scope:--hi-attachment;overflow:hidden}
				[popover] {position:fixed;margin:0;padding:0;border:0;width:stretch;height:100px;
					max-width:none;max-height:none;overflow:auto}
				.content {width:1000px}
				%s</style><div class="scope"><div popover="manual" class="%s"><div class="content"></div></div></div>`,
				test.left, sh.CSS(), class)
			uitest.Run(t, 800, 600, fixture, func(s *uitest.Session) {
				s.Eval(`document.querySelector('[popover]').showPopover()`, nil)
				r := s.Rect("[popover]", 0)
				if math.Abs(r.X-test.wantX) > .1 || math.Abs(r.W-test.wantW) > .1 {
					t.Errorf("got %+v, want x=%v w=%v", r, test.wantX, test.wantW)
				}
			})
		})
	}
}

func presentationLayoutStage(t *testing.T, v View, css string, fn func(*uitest.Session)) {
	t.Helper()
	_, page := Render(v)
	fixture := "<!doctype html><meta charset=utf-8><style>" + string(staticCSS) +
		"</style><body>" + renderNode(t, page) + "<style>" + css + "</style>"
	uitest.Run(t, 800, 600, fixture, func(s *uitest.Session) {
		s.Eval(`document.querySelectorAll('[data-hi-open="true"]').forEach(e => {
			if (e.localName === 'dialog') e.showModal(); else e.showPopover();
		})`, nil)
		fn(s)
	})
}

func TestPresentationScrollLayout(t *testing.T) {
	for _, test := range []struct {
		name         string
		top          int
		wantY, wantH float64
	}{
		{"above", 400, 4, 392},
		{"below", 100, 144, 452},
	} {
		t.Run(test.name, func(t *testing.T) {
			var entries []View
			for i := range 40 {
				entries = append(entries, Text(fmt.Sprintf("Menu item %d", i)).
					Frame(Height(32)).Class("entry"))
			}
			menu := ScrollView(Vertical, VStack(entries...).Gap(0)).
				Class("menu-scroll").Frame(Width(240))
			v := Text("Trigger").Frame(Width(80), Height(40)).
				Menu(Present(true, 1), menu).
				Class("trigger")
			css := fmt.Sprintf(".trigger {position:fixed;left:300px;top:%dpx;overflow:hidden}", test.top)
			presentationLayoutStage(t, v, css, func(s *uitest.Session) {
				for _, selector := range []string{"[data-hi-presentation]", ".menu-scroll"} {
					r := s.Rect(selector, 0)
					wantY, wantH := test.wantY, test.wantH
					if selector == "[data-hi-presentation]" {
						wantY, wantH = wantY-4, wantH+8
					}
					if math.Abs(r.X-300) > .1 || math.Abs(r.Y-wantY) > .1 ||
						math.Abs(r.W-240) > .1 || math.Abs(r.H-wantH) > .1 {
						t.Errorf("%s geometry = %+v, want x=300 y=%v w=240 h=%v", selector, r, wantY, wantH)
					}
				}
				var size struct{ Client, Scroll float64 }
				s.Eval(`(() => {const e=document.querySelector('.menu-scroll');
					return {Client:e.clientHeight,Scroll:e.scrollHeight}})()`, &size)
				if size.Client != test.wantH || size.Scroll != 1280 {
					t.Errorf("menu scroll sizes = %+v, want client=%v scroll=1280", size, test.wantH)
				}
				s.Eval(`document.querySelector('.menu-scroll').scrollTop = 10000`, nil)
				last := s.Rect(".entry", len(entries)-1)
				viewport := s.Rect(".menu-scroll", 0)
				if last.Y < viewport.Y || math.Abs(last.Bottom()-viewport.Bottom()) > .1 {
					t.Errorf("last entry not fully reachable: entry=%+v viewport=%+v", last, viewport)
				}
				var documentFits bool
				s.Eval(`document.scrollingElement.scrollHeight <= innerHeight &&
					document.scrollingElement.scrollWidth <= innerWidth`, &documentFits)
				if !documentFits {
					t.Error("presentation caused document overflow")
				}
			})
		})
	}
}

// The browser harness runs Chrome; Safari's compositor still needs a
// separate check for painted content escaping the scroll viewport.
func TestPresentationPageScrollLayout(t *testing.T) {
	for _, tc := range []struct {
		name         string
		scroll, move int
		above        bool
		wantHeight   float64
	}{
		{"above", 600, -50, true, 400},
		{"below", 900, 50, false, 460},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var entries []View
			for i := range 40 {
				entries = append(entries, Text(fmt.Sprintf("Item %d", i)).Frame(Height(32)).Class("entry"))
			}
			menu := ScrollView(Vertical, VStack(entries...).Gap(0)).Class("menu-scroll").Frame(Width(240))
			trigger := Text("Trigger").Frame(Width(80), Height(40)).Class("trigger").
				Menu(Present(false, 7), menu)
			_, page := Render(ScrollView(Vertical, VStack(
				Text("Before").Frame(Height(1000)), trigger, Text("After").Frame(Height(1200)),
			).Gap(0)))
			fixture := "<!doctype html><style>" + string(staticCSS) + "</style><body>" + renderNode(t, page) +
				`<script type="module">` + string(rawClientJS) + `
				run({clone: e => e.cloneNode(true)}); globalThis.ready = true;</script>`
			uitest.Run(t, 800, 600, fixture, func(s *uitest.Session) {
				s.Run(chromedp.Poll(`globalThis.ready`, nil))
				s.Eval(fmt.Sprintf(`window.scrollTo(0, %d)`, tc.scroll), nil)
				s.Run(chromedp.Poll(fmt.Sprintf(`window.scrollY === %d`, tc.scroll), nil))
				var documentHeight float64
				s.Eval(`document.scrollingElement.scrollHeight`, &documentHeight)
				s.Eval(`document.querySelector('[popover]').dataset.hiOpen = 'true'`, nil)
				s.Run(chromedp.Poll(`document.querySelector('[popover]').matches(':popover-open')`, nil))
				opened := s.Rect("[popover]", 0)
				if math.Abs(opened.H-tc.wantHeight) > .1 {
					t.Errorf("opened height = %v, want most available height %v", opened.H, tc.wantHeight)
				}
				checkAttachment := func() {
					anchor, surface := s.Rect(".trigger", 0), s.Rect("[popover]", 0)
					delta := surface.Y - anchor.Bottom()
					if tc.above {
						delta = surface.Bottom() - anchor.Y
					}
					if math.Abs(delta) > .1 || math.Abs(surface.X-anchor.X) > .1 {
						t.Errorf("menu lost attachment: trigger=%+v surface=%+v", anchor, surface)
					}
				}
				checkAttachment()
				s.Eval(fmt.Sprintf(`document.querySelector('.menu-scroll').scrollTop = 96;
					window.scrollTo(0, %d); globalThis.scrolledFrame = false;
					requestAnimationFrame(() => requestAnimationFrame(() => scrolledFrame = true))`, tc.scroll+tc.move), nil)
				s.Run(chromedp.Poll(fmt.Sprintf(`scrolledFrame && window.scrollY === %d`, tc.scroll+tc.move), nil))
				checkAttachment()
				for _, selector := range []string{"[popover]", ".menu-scroll"} {
					r := s.Rect(selector, 0)
					wantHeight := opened.H
					if selector == ".menu-scroll" {
						wantHeight -= 8
					}
					if math.Abs(r.W-opened.W) > .1 || math.Abs(r.H-wantHeight) > .1 {
						t.Errorf("page scroll resized %s: before=%+v after=%+v", selector, opened, r)
					}
				}
				var clipped bool
				s.Eval(`(() => {
					const e = document.querySelector('.menu-scroll'), r = e.getBoundingClientRect(), x = r.x + r.width / 2;
					return r.top > 2 && r.bottom < innerHeight - 2 && e.scrollTop === 96 &&
						e.contains(document.elementFromPoint(x, (r.top + r.bottom) / 2)) &&
						[r.top - 2, r.bottom + 2].every(y => !document.elementFromPoint(x, y)?.closest('.entry'));
				})()`, &clipped)
				if !clipped {
					t.Error("menu contents escaped the scroll viewport hit-test region")
				}
				var finalHeight float64
				s.Eval(`document.scrollingElement.scrollHeight`, &finalHeight)
				if finalHeight != documentHeight {
					t.Errorf("presentation changed document height from %v to %v", documentHeight, finalHeight)
				}
				// Repairing native top-layer membership is not a fresh open.
				s.Eval(`(() => {const e = document.querySelector('[popover]'); e.parentElement.append(e)})()`, nil)
				s.Run(chromedp.Poll(`document.querySelector('[popover]').matches(':popover-open')`, nil))
				repaired := s.Rect("[popover]", 0)
				if math.Abs(repaired.W-opened.W) > .1 || math.Abs(repaired.H-opened.H) > .1 {
					t.Errorf("DOM move changed open dimensions: before=%+v after=%+v", opened, repaired)
				}
				checkAttachment()
				s.Eval(`document.querySelector('[popover]').dataset.hiOpen = 'false'`, nil)
				s.Run(chromedp.Poll(`!document.querySelector('[popover]').matches(':popover-open')`, nil))
				s.Eval(`document.querySelector('[popover]').dataset.hiOpen = 'true'`, nil)
				s.Run(chromedp.Poll(`document.querySelector('[popover]').matches(':popover-open')`, nil))
				reopened := s.Rect("[popover]", 0)
				if want := tc.wantHeight + math.Abs(float64(tc.move)); math.Abs(reopened.H-want) > .1 {
					t.Errorf("reopen retained stale dimensions: got=%+v, want height=%v", reopened, want)
				}
				checkAttachment()
			})
		})
	}
}

func TestDialogLocksDocumentScroll(t *testing.T) {
	for _, axis := range []AxisSet{Vertical, Horizontal} {
		t.Run(fmt.Sprint(axis), func(t *testing.T) {
			width, height, position, offset := complex(800, 0), complex(2400, 0), "scrollY", "scrollTop"
			dx, dy := 0.0, 120.0
			if axis == Horizontal {
				width, height, position, offset, dx, dy = 2400, 600, "scrollX", "scrollLeft", 120, 0
			}
			content := ScrollView(axis, Text("Dialog content").Frame(Width(2400), Height(2400))).
				Class("dialog-scroll").Frame(Width(200), Height(160))
			_, page := Render(ScrollView(axis, ZStack(
				Text("Page").Frame(Width(width), Height(height)).Menu(Present(false, 7), Text("Menu")),
				Text("Trigger").Dialog(Present(false, 7), content),
				Text("Trigger").Dialog(Present(false, 7), Text("Second dialog").Frame(Width(100), Height(60))),
			)))
			fixture := "<!doctype html><style>" + string(staticCSS) + "</style><body>" + renderNode(t, page) +
				`<script type="module">` + string(rawClientJS) + `
				run({clone: e => e.cloneNode(true)}); globalThis.ready = true;</script>`
			uitest.Run(t, 800, 600, fixture, func(s *uitest.Session) {
				s.Run(chromedp.Poll(`globalThis.ready`, nil))
				wheel := func(x, y float64) {
					s.Run(input.DispatchMouseEvent(input.MouseMoved, x, y),
						input.DispatchMouseEvent(input.MouseWheel, x, y).WithDeltaX(dx).WithDeltaY(dy))
				}
				setDialog := func(index int, open bool) {
					s.Eval(fmt.Sprintf(`document.querySelectorAll('dialog')[%d].dataset.hiOpen = '%t'`, index, open), nil)
					s.Run(chromedp.Poll(fmt.Sprintf(`document.querySelectorAll('dialog')[%d].matches(':modal') === %t`, index, open), nil))
				}
				locked := func() {
					var overflow string
					s.Eval(`getComputedStyle(document.documentElement).overflow`, &overflow)
					if overflow != "hidden" {
						t.Errorf("modal document overflow = %q, want hidden", overflow)
					}
					var got float64
					s.Eval("window."+position, &got)
					if got != 400 {
						t.Errorf("modal dialog changed page scroll offset: got %v, want 400", got)
					}
				}
				s.Eval(fmt.Sprintf(`window.scrollTo(%v, %v)`, dx/120*400, dy/120*400), nil)
				s.Run(chromedp.Poll("window."+position+" === 400", nil))
				setDialog(0, true)
				locked()
				r := s.Rect(".dialog-scroll", 0)
				wheel(r.X+r.W/2, r.Y+r.H/2)
				s.Run(chromedp.Poll(`document.querySelector('.dialog-scroll').`+offset+` > 0`, nil))
				locked()
				setDialog(1, true)
				setDialog(0, false)
				locked()
				setDialog(1, false)
				wheel(5, 5)
				s.Run(chromedp.Poll("window."+position+" > 400", nil))
				var before float64
				s.Eval("window."+position, &before)
				s.Eval(`document.querySelector('[popover]').dataset.hiOpen = 'true'`, nil)
				s.Run(chromedp.Poll(`document.querySelector('[popover]').matches(':popover-open')`, nil))
				wheel(5, 5)
				s.Run(chromedp.Poll(fmt.Sprintf("window.%s > %v", position, before), nil))
			})
		})
	}
}

func TestPresentationNestedAttachmentLayout(t *testing.T) {
	inner := Text("Nested").Frame(Width(100), Height(60)).Class("inner-content")
	outer := Text("Inner trigger").Frame(Width(100), Height(40)).
		modify(wrapAttachment{popover: nodePopover{
			isOpen: true, dismiss: Present(true, 2).dismiss, content: inner, at: Trailing, anchor: Leading,
		}}.modify).Class("inner-trigger").
		Frame(Width(200), Height(120)).Class("outer-content")
	v := Text("Outer trigger").Frame(Width(80), Height(40)).
		Menu(Present(true, 1), outer).Class("outer-trigger")
	presentationLayoutStage(t, v,
		".outer-trigger {position:fixed;left:300px;top:100px}", func(s *uitest.Session) {
			outer := s.Rect("[data-hi-presentation]", 0)
			inner := s.Rect("[data-hi-presentation]", 1)
			trigger := s.Rect(".inner-trigger", 0)
			if outer.X != 300 || outer.Y != 140 || outer.W != 200 || outer.H != 128 {
				t.Errorf("outer geometry = %+v, want x=300 y=140 w=200 h=128", outer)
			}
			if math.Abs(inner.X-trigger.Right()) > .1 ||
				math.Abs(inner.Y+inner.H/2-trigger.Y-trigger.H/2) > .1 {
				t.Errorf("nested attachment used wrong anchor: inner=%+v trigger=%+v", inner, trigger)
			}
			outerContent, innerContent := s.Rect(".outer-content", 0), s.Rect(".inner-content", 0)
			if outerContent.Y != outer.Y+4 || outerContent.H != outer.H-8 ||
				innerContent.X != inner.X+4 || innerContent.W != inner.W-8 {
				t.Errorf("missing exterior padding: outer=%+v inner=%+v", outerContent, innerContent)
			}
			var open int
			s.Eval(`document.querySelectorAll(':popover-open').length`, &open)
			if open != 2 {
				t.Errorf("open presentations = %d, want 2", open)
			}
		})
}

func runPresentationClient(t *testing.T, fn func(*uitest.Session)) {
	t.Helper()
	page := `<style>hi-dismiss{display:none} [popover],dialog{width:200px;height:100px}</style>
	<button id="trigger">Trigger</button><button id="outside">Outside</button>
	<div id="surfaces">
	<div id="popover" popover="manual" data-hi-presentation="popover" data-hi-open="false">
		<hi-dismiss hidden></hi-dismiss><button id="popover-control">Menu item</button>
	</div>
	<dialog id="dialog" data-hi-presentation="dialog" data-hi-open="false">
		<hi-dismiss hidden></hi-dismiss><button id="dialog-control">Dialog control</button>
		<div id="nested" popover="manual" data-hi-presentation="popover" data-hi-open="false">
			<hi-dismiss hidden></hi-dismiss><button id="nested-control">Nested control</button>
		</div>
	</dialog></div><script type="module">` + string(rawClientJS) + `
	globalThis.requests = [];
	document.addEventListener('click', e => {
		if (e.target.matches('hi-dismiss')) requests.push(e.target.parentElement.id);
	});
	run({clone: e => e.cloneNode(true)});
	globalThis.ready = true;
	</script>`
	uitest.Run(t, 800, 600, page, func(s *uitest.Session) {
		s.Run(chromedp.Poll(`globalThis.ready`, nil))
		fn(s)
	})
}

func TestPresentationClientControlledDismissal(t *testing.T) {
	t.Parallel()
	runPresentationClient(t, func(s *uitest.Session) {
		s.Eval(`document.querySelector('#trigger').focus();
			document.querySelector('#popover').dataset.hiOpen = 'true';`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#popover').matches(':popover-open')`, nil),
			chromedp.KeyEvent(kb.Escape))
		s.Run(chromedp.Poll(`requests.join() === 'popover' && document.querySelector('#popover').matches(':popover-open')`, nil))
		s.Run(chromedp.Click("#outside", chromedp.ByQuery))
		s.Run(chromedp.Poll(`requests.join() === 'popover,popover' && document.querySelector('#popover').matches(':popover-open')`, nil))
		s.Eval(`document.querySelector('#popover-control').focus();
			document.querySelector('#popover').dataset.hiOpen = 'false';`, nil)
		s.Run(chromedp.Poll(`!document.querySelector('#popover').matches(':popover-open') && document.activeElement.id === 'trigger'`, nil))
		s.Eval(`document.querySelector('#dialog').dataset.hiOpen = 'true';`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#dialog').matches(':modal')`, nil), chromedp.KeyEvent(kb.Escape))
		s.Run(chromedp.Poll(`requests.join() === 'popover,popover,dialog' && document.querySelector('#dialog').matches(':modal')`, nil))
		s.Eval(`document.querySelector('#dialog').dataset.hiOpen = 'false';`, nil)
		s.Run(chromedp.Poll(`!document.querySelector('#dialog').open && document.activeElement.id === 'trigger'`, nil))
	})
}

func TestPresentationClientNestingAndSnapshots(t *testing.T) {
	t.Parallel()
	runPresentationClient(t, func(s *uitest.Session) {
		s.Run(chromedp.Poll(`!document.querySelector('#nested').matches(':popover-open')`, nil))
		s.Eval(`document.querySelector('#dialog').dataset.hiOpen = 'true';
			document.querySelector('#nested').dataset.hiOpen = 'true';`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#dialog').matches(':modal') && document.querySelector('#nested').matches(':popover-open')`, nil),
			chromedp.KeyEvent(kb.Escape))
		s.Run(chromedp.Poll(`requests.join() === 'nested' && document.querySelector('#nested').matches(':popover-open')`, nil))
		s.Eval(`globalThis.snapshot = document.querySelector('#surfaces').cloneNode(true);
			document.querySelector('#surfaces').replaceWith(snapshot);`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#dialog').matches(':modal') && document.querySelector('#nested').matches(':popover-open')`, nil),
			chromedp.KeyEvent(kb.Escape))
		s.Run(chromedp.Poll(`requests.join() === 'nested,nested'`, nil))
		s.Eval(`document.querySelector('#dialog').dataset.hiOpen = 'false';
			document.querySelector('#nested').dataset.hiOpen = 'false';`, nil)
		s.Run(chromedp.Poll(`!document.querySelector('#dialog').open && !document.querySelector('#nested').matches(':popover-open')`, nil))
		s.Eval(`document.querySelector('#dialog').dataset.hiOpen = 'true';
			document.querySelector('#nested').dataset.hiOpen = 'true';`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#dialog').matches(':modal') && document.querySelector('#nested').matches(':popover-open')`, nil))
	})
}

func TestPresentationClientNativeStateRepair(t *testing.T) {
	t.Parallel()
	runPresentationClient(t, func(s *uitest.Session) {
		s.Eval(`document.querySelector('#popover').dataset.hiOpen = 'true';`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#popover').matches(':popover-open')`, nil))
		s.Eval(`document.querySelector('#dialog').dataset.hiOpen = 'true';`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#dialog').matches(':modal') && document.querySelector('#popover').matches(':popover-open')`, nil))
		// Native state can also change without a patch to the application flag.
		s.Eval(`document.querySelector('#dialog').close();`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#dialog').matches(':modal') && requests.length === 0`, nil))
		s.Eval(`document.querySelector('#popover').hidePopover();`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#popover').matches(':popover-open') && requests.length === 0`, nil))
		s.Run(chromedp.KeyEvent(kb.Escape), chromedp.Poll(`requests.join() === 'dialog'`, nil))
		s.Eval(`document.querySelector('#dialog').dataset.hiOpen = 'false';
			document.querySelector('#popover').dataset.hiOpen = 'false';`, nil)
		s.Run(chromedp.Poll(`!document.querySelector('#dialog').open && !document.querySelector('#popover').matches(':popover-open')`, nil))
		s.Eval(`document.querySelector('#popover').showPopover();
			document.querySelector('#dialog').showModal();`, nil)
		s.Run(chromedp.Poll(`!document.querySelector('#dialog').open && !document.querySelector('#popover').matches(':popover-open')`, nil))
	})
}

func TestPresentationClientRemovalRestoresFocus(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"dialog", "popover"} {
		t.Run(id, func(t *testing.T) {
			runPresentationClient(t, func(s *uitest.Session) {
				s.Eval(`document.querySelector('#trigger').focus();
					document.querySelector('#`+id+`').dataset.hiOpen = 'true';`, nil)
				s.Run(chromedp.Poll(`document.querySelector('#`+id+`').matches(':modal,:popover-open')`, nil))
				s.Eval(`document.querySelector('#`+id+`-control').focus();`, nil)
				s.Eval(`document.querySelector('#`+id+`').remove();`, nil)
				s.Run(chromedp.Poll(`document.activeElement.id === 'trigger'`, nil))
			})
		})
	}
}

func TestPresentationClientMovePreservesOrder(t *testing.T) {
	t.Parallel()
	for _, move := range []string{"common ancestor", "lower surface"} {
		t.Run(move, func(t *testing.T) {
			runPresentationClient(t, func(s *uitest.Session) {
				// Open the two surfaces in the reverse of registration order.
				s.Eval(`document.querySelector('#surfaces').append(document.querySelector('#nested'));
					for (const e of document.querySelectorAll('[popover]')) {
						e.style.cssText = 'position:fixed;inset:auto;left:100px;top:100px;margin:0';
					}
					document.querySelector('#trigger').focus();
					document.querySelector('#nested').dataset.hiOpen = 'true';`, nil)
				s.Run(chromedp.Poll(`document.querySelector('#nested').matches(':popover-open')`, nil))
				s.Eval(`document.querySelector('#nested-control').focus();
					document.querySelector('#popover').dataset.hiOpen = 'true';`, nil)
				s.Run(chromedp.Poll(`document.querySelector('#popover').matches(':popover-open')`, nil))
				s.Eval(`document.querySelector('#popover-control').focus();`, nil)
				selector := "#surfaces"
				if move == "lower surface" {
					selector = "#nested"
				}
				s.Eval(`document.body.append(document.querySelector('`+selector+`'));`, nil)
				s.Run(chromedp.Poll(`document.querySelector('#nested').matches(':popover-open') &&
					document.querySelector('#popover').matches(':popover-open') &&
					document.elementFromPoint(120, 120).closest('[data-hi-presentation]').id === 'popover' &&
					document.activeElement.id === 'popover-control'`, nil),

					chromedp.KeyEvent(kb.Escape))
				s.Run(chromedp.Poll(`requests.join() === 'popover'`, nil))
				s.Eval(`document.querySelector('#popover').dataset.hiOpen = 'false';`, nil)
				s.Run(chromedp.Poll(`!document.querySelector('#popover').matches(':popover-open') &&
					document.activeElement.id === 'nested-control'`, nil),
					chromedp.KeyEvent(kb.Escape))
				s.Run(chromedp.Poll(`requests.join() === 'popover,nested'`, nil))
			})
		})
	}
}

func TestPresentationClientFocusTargetLifetime(t *testing.T) {
	t.Parallel()
	runPresentationClient(t, func(s *uitest.Session) {
		s.Eval(`document.querySelector('#trigger').focus();
			document.querySelector('#popover').dataset.hiOpen = 'true';`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#popover').matches(':popover-open')`, nil))
		s.Eval(`document.querySelector('#popover-control').focus();
			document.querySelector('#trigger').remove();
			document.querySelector('#popover').dataset.hiOpen = 'false';`, nil)
		s.Run(chromedp.Poll(`!document.querySelector('#popover').matches(':popover-open')`, nil))
		// Reopening takes a fresh focus target, even though the native node
		// and its registration survived the previous presentation.
		s.Eval(`document.querySelector('#outside').focus();
			document.querySelector('#popover').dataset.hiOpen = 'true';`, nil)
		s.Run(chromedp.Poll(`document.querySelector('#popover').matches(':popover-open')`, nil))
		s.Eval(`document.querySelector('#popover-control').focus();
			document.querySelector('#popover').dataset.hiOpen = 'false';`, nil)
		s.Run(chromedp.Poll(`!document.querySelector('#popover').matches(':popover-open') &&
			document.activeElement.id === 'outside'`, nil))
	})
}
