package ui_test

import (
	"fmt"
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
)

func TestButtonStateStructure(t *testing.T) {
	for _, style := range []ui.ButtonStyle{ui.Bordered, ui.Prominent, ui.Subtle, ui.Borderless, ui.Destructive, ui.DestructiveSubtle} {
		for _, action := range []any{Msg{}, "/movies"} {
			t.Run(fmt.Sprintf("%d/%T", style, action), func(t *testing.T) {
				var buttons []ui.View
				shared := ui.Button(action, ui.Text("Save"))
				for bits := 0; bits < 8; bits++ {
					buttons = append(buttons, shared.Selected(bits&1 != 0).MenuOpen(bits&2 != 0).Disabled(bits&4 != 0))
				}
				stage(t, ui.VStack(buttons...).ButtonStyle(style), func(s *uitest.Session) {
					var got []struct {
						Tree     string
						Disabled bool
					}
					s.Eval(`Array.from(document.querySelectorAll('button, a'), e => ({
						Tree: Array.from(e.querySelectorAll('*'), n => n.localName).join(','),
						Disabled: e.disabled === true || e.getAttribute('aria-disabled') === 'true'
					}))`, &got)
					if len(got) != 8 {
						t.Fatalf("controls = %d", len(got))
					}
					for i := range 8 {
						if got[i].Tree != "ui-text,span" || got[i].Disabled != (i&4 != 0) {
							t.Errorf("state %d = %+v", i, got[i])
						}
					}
				})
			})
		}
	}
}

func TestButtonStateSemantics(t *testing.T) {
	for _, action := range []any{Msg{}, "/movies"} {
		for _, disabled := range []bool{false, true} {
			html := render(t, ui.Button(action, ui.Text("Save")).Disabled(disabled))
			if disabled && strings.Contains(html, `href="/movies"`) {
				t.Error("disabled URL has navigation target")
			}
			if strings.Contains(html, "aria-pressed") || strings.Contains(html, "aria-expanded") {
				t.Error("button assigns application-owned state semantics")
			}
		}
	}
}
