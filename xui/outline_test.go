package ui_test

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/chromedp"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
)

func TestBorderOutlineStates(t *testing.T) {
	v := ui.Transparent.Frame(ui.Width(80), ui.Height(40)).
		WhileFocused(ui.BorderOutline(4, 3, ui.Red)).
		WhilePressed(ui.BorderOutline(6, 5, ui.Blue)).
		WhileHovered(ui.BorderOutline(8, 7, ui.White)).
		BorderOutline(2, 1, ui.Black).
		BorderStroke(2, ui.Red).
		BorderShadow(0, 0, 10, 0, ui.Blue).
		BorderShape(ui.Capsule).Tag("button").Class("subject")
	stage(t, v, func(s *uitest.Session) {
		var nodes []*cdp.Node
		s.Run(css.Enable(), chromedp.Nodes(".subject", &nodes, chromedp.ByQuery))
		before := s.Rect(".subject", 0)
		for _, touch := range []bool{false, true} {
			var opts []chromedp.EmulateViewportOption
			if touch {
				opts = append(opts, chromedp.EmulateTouch)
			}
			s.Run(chromedp.EmulateViewport(600, 400, opts...))
			var canHover bool
			s.Eval(`matchMedia('(hover: hover)').matches`, &canHover)
			if touch && canHover {
				t.Fatal("touch emulation did not disable hover capability")
			}
			for _, states := range [][]string{
				nil, {"hover"}, {"active"}, {"focus-visible"},
				{"hover", "active"}, {"hover", "focus-visible"},
				{"active", "focus-visible"}, {"hover", "active", "focus-visible"}, nil,
			} {
				s.Run(css.ForcePseudoState(nodes[0].NodeID, states))
				color, width, gap := "oklch(0 0 0)", "1px", "2px"
				switch {
				case slices.Contains(states, "focus-visible"):
					color, width, gap = redCSS, "3px", "4px"
				case slices.Contains(states, "active"):
					color, width, gap = blueCSS, "5px", "6px"
				case canHover && slices.Contains(states, "hover"):
					color, width, gap = whiteCSS, "7px", "8px"
				}
				var got []string
				s.Eval(`(() => {
					const e = document.querySelector('.subject');
					const p = getComputedStyle(e, '::after'), b = getComputedStyle(e);
					return [p.outlineColor, p.outlineWidth, p.outlineOffset,
						p.outlineStyle, p.boxShadow, b.boxShadow, b.outlineStyle];
				})()`, &got)
				want := []string{color, width, gap, "solid",
					redCSS + " 0px 0px 0px 2px inset", blueCSS + " 0px 0px 0px 10px", "none"}
				if !slices.Equal(got, want) {
					t.Errorf("canHover=%v, %v paint = %v, want %v", canHover, states, got, want)
				}
				if after := s.Rect(".subject", 0); after != before {
					t.Errorf("outline changed geometry: %+v -> %+v", before, after)
				}
			}
		}
	})
}

func TestBorderOutlineInvisiblePrecedence(t *testing.T) {
	for _, tt := range []struct {
		name         string
		inner        ui.Modifier
		width, color string
	}{
		{"transparent", ui.BorderOutline(4, 3, ui.Transparent), "3px", "oklch(0 0 0 / 0)"},
		{"zero width", ui.BorderOutline(4, 0, ui.Red), "0px", redCSS},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, ui.Text("x").WhileFocused(tt.inner).
				BorderOutline(2, 1, ui.Blue).Tag("button").Class("subject"), func(s *uitest.Session) {
				var nodes []*cdp.Node
				s.Run(css.Enable(), chromedp.Nodes(".subject", &nodes, chromedp.ByQuery))
				for _, focused := range []bool{false, true, false} {
					var states []string
					want := []string{"1px", "2px", blueCSS}
					if focused {
						states = []string{"focus-visible"}
						want = []string{tt.width, "4px", tt.color}
					}
					s.Run(css.ForcePseudoState(nodes[0].NodeID, states))
					var got []string
					s.Eval(`(() => {
						const p = getComputedStyle(document.querySelector('.subject'), '::after');
						return [p.outlineWidth, p.outlineOffset, p.outlineColor];
					})()`, &got)
					if !slices.Equal(got, want) {
						t.Errorf("focused=%v: %v, want %v", focused, got, want)
					}
				}
			})
		})
	}
}

func TestBorderOutlineBrowserFocus(t *testing.T) {
	// Without a matching state, no authored outline exists to replace the
	// browser's focus indicator. Pointer activation adds paint without
	// moving focus.
	v := ui.Text("x").Padding(ui.Edges(4)).
		WhilePressed(ui.BorderOutline(3, 2, ui.Red)).Tag("button").Class("subject")
	stage(t, v, func(s *uitest.Session) {
		s.Run(chromedp.KeyEvent("\t"))
		var focused bool
		s.Eval(`document.activeElement.matches('.subject:focus-visible')`, &focused)
		if !focused {
			t.Fatal("Tab did not focus the control")
		}
		var original string
		s.Eval(`getComputedStyle(document.activeElement).outline`, &original)
		var nodes []*cdp.Node
		s.Run(css.Enable(), chromedp.Nodes(".subject", &nodes, chromedp.ByQuery))
		for _, pressed := range []bool{true, false} {
			var states []string
			if pressed {
				states = []string{"active"}
			}
			s.Run(css.ForcePseudoState(nodes[0].NodeID, states))
			var got []string
			s.Eval(`(() => {
				const e = document.activeElement;
				return [getComputedStyle(e).outline, getComputedStyle(e, '::after').outlineStyle];
			})()`, &got)
			if pressed {
				if !strings.Contains(got[0], "none") || got[1] != "solid" {
					t.Errorf("authored indicator did not replace browser outline: %v", got)
				}
			} else if got[0] != original || got[1] != "none" {
				t.Errorf("browser outline did not return: %v, originally %s", got, original)
			}
		}
	})
}

func TestBorderOutlineScopeAndStructure(t *testing.T) {
	base := ui.Text("x").Opacity(.5)
	tags := regexp.MustCompile(`</?[a-z][a-z0-9-]*`)
	var structure []string
	for _, width := range []complex128{0, 1, 3i} {
		for _, gap := range []complex128{0, 2, 5i} {
			for _, c := range []ui.Color{ui.Transparent, ui.Red} {
				v := base.BorderOutline(gap, width, c).
					WhileFocused(ui.BorderOutline(4, 3, ui.Blue)).BorderOutline(6, 5, ui.Red)
				html := render(t, v)
				got := tags.FindAllString(html, -1)
				if structure == nil {
					structure = got
				}
				if !slices.Equal(got, structure) || strings.Count(html, "<ui-box ") != 1 {
					t.Errorf("outline values changed the opacity boundary: %s", html)
				}
			}
		}
	}
	inner := ui.Text("x").BorderOutline(2, 1, ui.Red).Class("inner")
	stage(t, ui.HStack(inner).BorderOutline(4, 3, ui.Blue).Class("outer"), func(s *uitest.Session) {
		var got []string
		s.Eval(`['.inner', '.outer'].map(sel => getComputedStyle(document.querySelector(sel), '::after').outlineColor)`, &got)
		if !slices.Equal(got, []string{redCSS, blueCSS}) {
			t.Errorf("descendant outline affected enclosing box: %v", got)
		}
	})
}

func TestBorderOutlineLengths(t *testing.T) {
	v := ui.Text("x").BorderOutline(-4+8i, 12-8i, ui.Red).Class("subject")
	stage(t, v, func(s *uitest.Session) {
		for _, root := range []float64{8, 16, 32} {
			s.Eval(fmt.Sprintf(`document.documentElement.style.fontSize = '%gpx'`, root), nil)
			var got []float64
			s.Eval(`(() => {
				const p = getComputedStyle(document.querySelector('.subject'), '::after');
				return [parseFloat(p.outlineWidth), parseFloat(p.outlineOffset)];
			})()`, &got)
			want := []float64{max(0, 12-8*root/16), max(0, -4+8*root/16)}
			if !slices.Equal(got, want) {
				t.Errorf("root %g: outline lengths = %v, want %v", root, got, want)
			}
		}
	})
}

func TestBorderOutlinePaint(t *testing.T) {
	base := ui.Transparent.Frame(ui.Width(80), ui.Height(40))
	outline := ui.BorderOutline(2, 3, ui.Black)
	for _, tt := range []struct {
		name    string
		view    ui.View
		outside uint32
	}{
		{"transparent", base.Modify(outline), 0},
		{"clip outside", base.Modify(outline).BorderClipped(), 65535},
		{"clip inside", base.BorderClipped().Modify(outline), 0},
		{"opacity outside", base.Modify(outline).Opacity(.5), 127 * 257},
		{"opacity inside", base.Opacity(.5).Modify(outline), 0},
		{"ancestor clip", base.Modify(outline).Padding(ui.Edges(1)).BorderClipped(), 65535},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, ui.HStack(tt.view.Class("subject")).Padding(ui.Edges(20)).Background(ui.White), func(s *uitest.Session) {
				for _, scale := range []int{1, 2} {
					s.Run(chromedp.EmulateViewport(600, 400, chromedp.EmulateScale(float64(scale))))
					r := s.Rect(".subject", 0)
					img := shadowScreenshot(t, s)
					for _, sample := range []struct {
						x    float64
						want uint32
					}{
						{r.X + r.W/2, 65535}, {r.X - 1, 65535}, {r.X - 4, tt.outside},
					} {
						got, _, _, _ := img.At(int(sample.x*float64(scale)), int((r.Y+r.H/2)*float64(scale))).RGBA()
						if absDiff(got, sample.want) > 257 {
							t.Errorf("%dx: pixel at x=%g = %d, want %d", scale, sample.x, got, sample.want)
						}
					}
				}
			})
		})
	}
}

func TestBorderOutlineOverflowAndLayers(t *testing.T) {
	large := ui.Black.Frame(ui.Width(100), ui.Height(60))
	for _, tt := range []struct {
		name string
		view ui.View
	}{
		{"overflow", large.Frame(ui.Width(80), ui.Height(40))},
		{"overlay", ui.Transparent.Frame(ui.Width(80), ui.Height(40)).Overlay(ui.Center, large)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, tt.view.BorderOutline(2, 3, ui.White).Class("subject").
				Padding(ui.Edges(20)).Background(ui.White), func(s *uitest.Session) {
				for _, scale := range []int{1, 2} {
					s.Run(chromedp.EmulateViewport(600, 400, chromedp.EmulateScale(float64(scale))))
					r := s.Rect(".subject", 0)
					img := shadowScreenshot(t, s)
					for _, sample := range []struct {
						x    float64
						want uint32
					}{
						{r.X - 1, 0}, {r.X - 4, 65535}, {r.X - 8, 0},
					} {
						got, _, _, _ := img.At(int(sample.x*float64(scale)), int((r.Y+r.H/2)*float64(scale))).RGBA()
						if got != sample.want {
							t.Errorf("%dx: pixel at x=%g = %d, want %d", scale, sample.x, got, sample.want)
						}
					}
				}
			})
		})
	}
}
