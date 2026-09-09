package ui_test

import (
	"fmt"
	"testing"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/chromedp"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
)

func TestButtonEdgeFeedback(t *testing.T) {
	for _, action := range []any{Msg{}, "/movies"} {
		t.Run(fmt.Sprintf("%T", action), func(t *testing.T) {
			stage(t, ui.HStack(
				ui.Button(action, ui.Text("Save")),
				ui.Button(action, ui.Text("Save")).Selected(true),
			), func(s *uitest.Session) {
				var nodes []*cdp.Node
				s.Run(css.Enable(), chromedp.Nodes("button, a", &nodes, chromedp.ByQueryAll))
				type paint struct {
					Edge, Border, Clip, Tree string
					Width, Height            float64
				}
				read := func() []paint {
					var got []paint
					s.Eval(`Array.from(document.querySelectorAll('button, a'), e => {
						const c = getComputedStyle(e), r = e.getBoundingClientRect();
						return {Edge:getComputedStyle(e, '::after').boxShadow.match(/oklch\([^)]*\)/)?.[0],
						 Border:c.borderWidth, Clip:[...new Set(c.backgroundClip.split(', '))].join(','), Width:r.width, Height:r.height,
						 Tree:Array.from(e.querySelectorAll('*'), n => n.localName).join(',')};
					})`, &got)
					return got
				}
				for _, scale := range []float64{1, 2} {
					s.Run(chromedp.EmulateViewport(600, 400, chromedp.EmulateScale(scale)))
					var canHover bool
					s.Eval(`matchMedia('(hover: hover)').matches`, &canHover)
					s.Run(css.ForcePseudoState(nodes[0].NodeID, nil))
					rest := read()
					if rest[0].Border != "0px" || rest[0].Clip != "border-box" || rest[0].Tree != "ui-text" {
						t.Errorf("unexpected button construction: %+v", rest[0])
					}
					for _, states := range [][]string{{"hover"}, {"active"}, {"hover", "active"}, nil} {
						s.Run(css.ForcePseudoState(nodes[0].NodeID, states))
						got := read()[0]
						want := rest[0]
						if len(states) > 0 && (states[0] != "hover" || canHover || len(states) > 1) {
							want.Edge = rest[1].Edge
						}
						if got != want {
							t.Errorf("scale %g, %v: paint/geometry = %+v, want %+v", scale, states, got, want)
						}
					}
				}
			})
		})
	}
}

func TestButtonEdgeLocalTheme(t *testing.T) {
	buttons := func() ui.View {
		return ui.HStack(
			ui.Button(Msg{}, ui.Text("Rest")),
			ui.Button(Msg{}, ui.Text("Selected")).Selected(true),
		)
	}
	for _, bg := range []ui.Color{ui.White, ui.OKLCH(0.95, 0.01, 90), ui.OKLCH(0.2, 0.02, 270)} {
		stage(t, ui.VStack(
			buttons(),
			buttons().Background(ui.Red),
			buttons().ThemeBackground(ui.OKLCH(0.4, 0.01, 270)),
			buttons(),
		).ThemeBackground(bg), func(s *uitest.Session) {
			var edges []string
			s.Eval(`Array.from(document.querySelectorAll('button'), e => getComputedStyle(e, '::after').boxShadow)`, &edges)
			for i := range 2 {
				if edges[i] != edges[i+2] || edges[i] != edges[i+6] {
					t.Errorf("edge followed authored backdrop or leaked nested theme: %v", edges)
				}
				if edges[i] == edges[i+4] {
					t.Errorf("edge did not follow nested theme: %v", edges)
				}
			}
		})
	}
}
