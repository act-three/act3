package ui_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
)

func TestButtonStates(t *testing.T) {
	for _, style := range []ui.ButtonStyle{ui.Bordered, ui.Prominent, ui.Subtle, ui.Borderless, ui.Destructive, ui.DestructiveSubtle} {
		for _, action := range []any{Msg{}, "/movies"} {
			t.Run(fmt.Sprintf("%d/%T", style, action), func(t *testing.T) {
				var buttons []ui.View
				shared := ui.Button(action, ui.Text("Save"))
				for bits := 0; bits < 8; bits++ {
					buttons = append(buttons, shared.Selected(bits&1 != 0).MenuOpen(bits&2 != 0).Disabled(bits&4 != 0))
				}
				buttons = append(buttons, shared, shared.Selected(false).Selected(true), shared.MenuOpen(false).MenuOpen(true), shared.Selected(true).Disabled(true).Opacity(0.5))
				stage(t, ui.VStack(buttons...).ButtonStyle(style), func(s *uitest.Session) {
					var got []struct {
						Background, Foreground, Opacity, Cursor, Tree, Edge string
						Disabled                                            bool
					}
					s.Eval(`Array.from(document.querySelectorAll('button, a'), e => {
      const c = getComputedStyle(e), label = getComputedStyle(e.querySelector('ui-text'));
      return {Background:c.backgroundColor, Foreground:label.color, Opacity:c.opacity, Cursor:c.cursor,
       Edge:getComputedStyle(e, '::after').boxShadow.match(/oklch\([^)]*\)/)?.[0],
       Tree:Array.from(e.querySelectorAll('*'), n => n.localName).join(','),
       Disabled:e.disabled === true || e.getAttribute('aria-disabled') === 'true'};
     })`, &got)
					if len(got) != 12 {
						t.Fatalf("controls = %d", len(got))
					}
					for i := range 8 {
						wantOpacity := "1"
						if _, link := action.(string); i&4 != 0 && (!link || i&1 == 0) {
							wantOpacity = "0.6"
						}
						if got[i].Opacity != wantOpacity || got[i].Cursor != "default" || got[i].Tree != "ui-text" || got[i].Disabled != (i&4 != 0) {
							t.Errorf("state %d = %+v", i, got[i])
						}
						if style == ui.Bordered {
							want := got[0].Edge
							if i&3 != 0 {
								want = got[1].Edge
								if want == got[0].Edge {
									t.Error("selected edge equals rest")
								}
							}
							if got[i].Edge != want || strings.Contains(got[i].Edge, "/") {
								t.Errorf("state %d edge = %s, want opaque %s", i, got[i].Edge, want)
							}
						} else if got[i].Edge != "oklch(0 0 0 / 0)" {
							t.Errorf("style %d state %d has visible edge %s", style, i, got[i].Edge)
						}
					}
					for _, pair := range [][2]int{{1, 5}, {2, 6}, {3, 7}} {
						if got[pair[0]].Background != got[pair[1]].Background {
							t.Errorf("disabled state %d changed face from state %d", pair[1], pair[0])
						}
					}
					if got[0].Background != got[4].Background {
						t.Error("disabled rest face changed")
					}
					for _, i := range []int{3, 6, 7} {
						if got[i].Background != got[2].Background {
							t.Errorf("state %d lost menu-open face", i)
						}
					}
					switch style {
					case ui.Borderless, ui.DestructiveSubtle:
						if got[0].Background != got[1].Background {
							t.Error("selection gained a background")
						}
					default:
						if got[0].Background == got[1].Background {
							t.Error("selected face equals rest")
						}
					}
					if style == ui.Subtle && (got[2].Background == got[1].Background || got[2].Background == got[0].Background) {
						t.Error("subtle menu-open must have a distinct hover face")
					}
					if style == ui.Bordered || style == ui.Subtle || style == ui.Borderless {
						if got[0].Foreground == got[1].Foreground {
							t.Error("selection did not highlight foreground")
						}
					}
					for i := 4; i < 8; i++ {
						want := got[i-4].Foreground
						if style == ui.Bordered || style == ui.DestructiveSubtle {
							want = got[4].Foreground
						}
						if got[i].Foreground != want {
							t.Errorf("disabled state %d foreground = %s, want %s", i, got[i].Foreground, want)
						}
					}
					if style == ui.Bordered && got[3].Foreground != got[1].Foreground {
						t.Error("menu-open replaced selected Bordered label")
					}
					switch style {
					case ui.Prominent, ui.Subtle, ui.Borderless, ui.Destructive:
						if got[4].Foreground != got[0].Foreground {
							t.Error("disabled label differs from resting label")
						}
					case ui.Bordered, ui.DestructiveSubtle:
						if got[4].Foreground == got[0].Foreground {
							t.Error("disabled label did not change from resting label")
						}
					}
					if style == ui.Destructive {
						for i, control := range got {
							if control.Foreground != "oklch(1 0 0)" {
								t.Errorf("destructive state %d foreground = %s, want white", i, control.Foreground)
							}
						}
					}
					for _, i := range []int{8, 9, 10} {
						if !reflect.DeepEqual(got[0], got[i]) {
							t.Errorf("shared view or explicit false changed state %d: %+v", i, got[i])
						}
					}
					wantOpacity := "0.3"
					if _, link := action.(string); link {
						wantOpacity = "0.5"
					}
					if got[11].Opacity != wantOpacity {
						t.Errorf("external opacity = %s", got[11].Opacity)
					}
				})
			})
		}
	}
}

func TestButtonFeedback(t *testing.T) {
	for _, action := range []any{Msg{}, "/movies"} {
		for _, disabled := range []bool{false, true} {
			html := render(t, ui.Button(action, ui.Text("Save")).Disabled(disabled))
			for _, selector := range []string{":hover", ":active"} {
				if strings.Contains(html, selector) == disabled {
					t.Errorf("disabled=%v: unexpected %s rules", disabled, selector)
				}
			}
			if !disabled && !strings.Contains(html, "(hover: hover)") {
				t.Error("missing hover capability guard")
			}
			if disabled && strings.Contains(html, `href="/movies"`) {
				t.Error("disabled URL has navigation target")
			}
			if strings.Contains(html, "aria-pressed") || strings.Contains(html, "aria-expanded") {
				t.Error("button assigns application-owned state semantics")
			}
		}
	}
}

// Exercise the actual pointer states: emitted background feedback alone
// does not establish that foreground modifier precedence is correct.
func TestButtonPointerForeground(t *testing.T) {
	for _, style := range []ui.ButtonStyle{ui.Subtle, ui.Borderless} {
		for _, action := range []any{Msg{}, "/movies"} {
			t.Run(fmt.Sprintf("%d/%T", style, action), func(t *testing.T) {
				stage(t, ui.VStack(
					ui.Button(action, ui.Text("Save")),
					ui.Button(action, ui.Text("Selected")).Selected(true),
					ui.Button(action, ui.Text("Custom").Foreground(ui.Green)),
				).ButtonStyle(style), func(s *uitest.Session) {
					var canHover bool
					s.Eval(`matchMedia('(hover: hover)').matches`, &canHover)
					colors := func() []string {
						var got []string
						s.Eval("Array.from(document.querySelectorAll('button ui-text, a ui-text'), e => getComputedStyle(e).color)", &got)
						return got
					}
					rest := colors()
					r := s.Rect("button, a", 0)
					x, y := r.X+r.W/2, r.Y+r.H/2
					s.Run(input.DispatchMouseEvent(input.MouseMoved, x, y))
					hover := colors()
					wantHover := rest[0]
					if canHover {
						wantHover = rest[1]
					}
					if hover[0] != wantHover {
						t.Errorf("canHover=%v: hover foreground = %s, want %s", canHover, hover[0], wantHover)
					}
					s.Run(input.DispatchMouseEvent(input.MousePressed, x, y).WithButton(input.Left).WithClickCount(1))
					pressed := colors()
					if pressed[0] != rest[1] {
						t.Errorf("pressed foreground = %s, want %s", pressed[0], rest[1])
					}
					s.Run(input.DispatchMouseEvent(input.MouseMoved, 599, 399))
					s.Run(input.DispatchMouseEvent(input.MouseReleased, 599, 399).WithButton(input.Left))
					r = s.Rect("button, a", 2)
					s.Run(input.DispatchMouseEvent(input.MouseMoved, r.X+r.W/2, r.Y+r.H/2))
					if got := colors(); got[2] != rest[2] {
						t.Errorf("hover overrode explicit label foreground: %s != %s", got[2], rest[2])
					}
				})
			})
		}
	}
}

func TestButtonCombinedFeedback(t *testing.T) {
	for _, style := range []ui.ButtonStyle{ui.Bordered, ui.Prominent, ui.Subtle, ui.Borderless, ui.Destructive, ui.DestructiveSubtle} {
		for _, action := range []any{Msg{}, "/movies"} {
			t.Run(fmt.Sprintf("%d/%T", style, action), func(t *testing.T) {
				button := func() ui.ButtonView { return ui.Button(action, ui.Text("Save")) }
				stage(t, ui.VStack(
					button().Selected(true),
					button().Selected(true).MenuOpen(true),
					button().Selected(true).Disabled(true),
					button().MenuOpen(true).Disabled(true),
				).ButtonStyle(style), func(s *uitest.Session) {
					type paint struct{ Background, Foreground, Opacity, Edge string }
					read := func() []paint {
						var got []paint
						s.Eval(`Array.from(document.querySelectorAll('button, a'), e => {
							const style = getComputedStyle(e);
							const canvas = document.createElement('canvas');
							canvas.width = canvas.height = 1;
							const ctx = canvas.getContext('2d');
							const layers = Array.from(style.backgroundImage.matchAll(/linear-gradient\((oklch\([^)]*\))/g), m => m[1]);
							for (const color of ['white', style.backgroundColor, ...layers.reverse()]) {
								ctx.fillStyle = color;
								ctx.fillRect(0, 0, 1, 1);
							}
							return {
								Background: Array.from(ctx.getImageData(0, 0, 1, 1).data).join(','),
								Foreground: getComputedStyle(e.querySelector('ui-text')).color,
								Opacity: style.opacity,
								Edge: getComputedStyle(e, '::after').boxShadow.match(/oklch\([^)]*\)/)?.[0],
							};
						})`, &got)
						return got
					}
					var nodes []*cdp.Node
					s.Run(css.Enable(), chromedp.Nodes("button, a", &nodes, chromedp.ByQueryAll))
					for _, touch := range []bool{false, true} {
						var opts []chromedp.EmulateViewportOption
						if touch {
							opts = append(opts, chromedp.EmulateTouch)
						}
						s.Run(chromedp.EmulateViewport(600, 400, opts...))
						for _, node := range nodes {
							s.Run(css.ForcePseudoState(node.NodeID, nil))
						}
						var canHover bool
						s.Eval(`matchMedia('(hover: hover)').matches`, &canHover)
						if touch && canHover {
							t.Fatal("touch emulation did not disable hover capability")
						}
						rest := read()
						for _, state := range [][]string{{"hover"}, {"active"}, {"hover", "active"}} {
							for _, node := range nodes {
								s.Run(css.ForcePseudoState(node.NodeID, state))
							}
							got := read()
							want := rest[1]
							if !canHover && len(state) == 1 && state[0] == "hover" {
								want = rest[0]
							}
							if got[0] != want || got[1] != rest[1] {
								t.Errorf("canHover=%v: selected + %s = %+v, want %+v and %+v", canHover, state, got[:2], want, rest[1])
							}
							if got[2] != rest[2] || got[3] != rest[3] {
								t.Errorf("canHover=%v: disabled + %s changed paint: %+v, want %+v", canHover, state, got[2:], rest[2:])
							}
						}
					}
				})
			})
		}
	}
}
