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

func TestButtonInteractionLayout(t *testing.T) {
	for _, action := range []any{Msg{}, "/movies"} {
		t.Run(fmt.Sprintf("%T", action), func(t *testing.T) {
			stage(t, ui.HStack(
				ui.Button(action, ui.Text("Save")),
				ui.Button(action, ui.Text("Save")).Selected(true),
			), func(s *uitest.Session) {
				var nodes []*cdp.Node
				s.Run(css.Enable(), chromedp.Nodes("button, a", &nodes, chromedp.ByQueryAll))
				type layout struct {
					Tree          string
					Width, Height float64
				}
				read := func() []layout {
					var got []layout
					s.Eval(`Array.from(document.querySelectorAll('button, a'), e => {
						const r = e.getBoundingClientRect();
						return {Width:r.width, Height:r.height,
						 Tree:Array.from(e.querySelectorAll('*'), n => n.localName).join(',')};
					})`, &got)
					return got
				}
				for _, scale := range []float64{1, 2} {
					s.Run(chromedp.EmulateViewport(600, 400, chromedp.EmulateScale(scale)))
					s.Run(css.ForcePseudoState(nodes[0].NodeID, nil))
					rest := read()
					if rest[0] != rest[1] || rest[0].Tree != "ui-text" {
						t.Errorf("selection changed layout: %+v", rest)
					}
					for _, states := range [][]string{{"hover"}, {"active"}, {"hover", "active"}, nil} {
						s.Run(css.ForcePseudoState(nodes[0].NodeID, states))
						got := read()[0]
						want := rest[0]
						if got != want {
							t.Errorf("scale %g, %v: layout = %+v, want %+v", scale, states, got, want)
						}
					}
				}
			})
		})
	}
}
