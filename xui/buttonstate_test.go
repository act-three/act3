package ui_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
)

func TestButtonStates(t *testing.T) {
	for _, role := range []ui.ButtonRole{ui.RoleDefault, ui.RolePrimary, ui.RoleDestructive} {
		for _, action := range []any{Msg{}, "/movies"} {
			t.Run(fmt.Sprintf("%d/%T", role, action), func(t *testing.T) {
				var buttons []ui.View
				shared := ui.Button(action, ui.Text("Save")).Role(role)
				for bits := 0; bits < 8; bits++ {
					buttons = append(buttons, shared.Selected(bits&1 != 0).MenuOpen(bits&2 != 0).Disabled(bits&4 != 0))
				}
				buttons = append(buttons, shared, shared.Selected(false).Selected(true), shared.MenuOpen(false).MenuOpen(true), shared.Selected(true).Disabled(true).Opacity(0.5))
				stage(t, ui.VStack(buttons...), func(s *uitest.Session) {
					var got []struct {
						Background, Foreground, Opacity, Cursor, Tree string
						Disabled                                      bool
					}
					s.Eval(`Array.from(document.querySelectorAll('button, a'), e => {
      const c = getComputedStyle(e), label = getComputedStyle(e.querySelector('ui-text'));
      return {Background:c.backgroundColor, Foreground:label.color, Opacity:c.opacity, Cursor:c.cursor,
       Tree:Array.from(e.querySelectorAll('*'), n => n.localName).join(','),
       Disabled:e.disabled === true || e.getAttribute('aria-disabled') === 'true'};
     })`, &got)
					if len(got) != 12 {
						t.Fatalf("controls = %d", len(got))
					}
					for i := range 8 {
						wantOpacity := "1"
						if i&4 != 0 && i&1 == 0 {
							wantOpacity = "0.6"
						}
						if got[i].Opacity != wantOpacity || got[i].Cursor != "default" || got[i].Tree != "ui-text" || got[i].Disabled != (i&4 != 0) {
							t.Errorf("state %d = %+v", i, got[i])
						}
					}
					for _, i := range []int{2, 3, 5, 7} {
						if got[i].Background != got[1].Background {
							t.Errorf("state %d lost interaction face", i)
						}
					}
					if got[0].Background == got[1].Background {
						t.Error("selected face equals rest")
					}
					if got[6].Background != got[4].Background || got[4].Background != got[0].Background {
						t.Error("disabled menu-open changed rest face")
					}
					for _, i := range []int{5, 6, 7} {
						if got[i].Foreground != got[4].Foreground {
							t.Errorf("disabled state %d did not mute foreground", i)
						}
					}
					for _, i := range []int{8, 9, 10} {
						if !reflect.DeepEqual(got[0], got[i]) {
							t.Errorf("shared view or explicit false changed state %d: %+v", i, got[i])
						}
					}
					if got[11].Opacity != "0.5" {
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
