package hi_test

import (
	"fmt"
	"strings"
	"testing"

	"ily.dev/act3/hi"
	"ily.dev/act3/hi/internal/uitest"
)

func TestButtonStateStructure(t *testing.T) {
	t.Parallel()
	var views []hi.View
	var names []string
	for _, style := range []hi.ButtonStyle{hi.Bordered, hi.Prominent, hi.Subtle, hi.Borderless, hi.Destructive, hi.DestructiveSubtle} {
		for _, action := range []any{Msg{}, "/movies"} {
			var buttons []hi.View
			shared := hi.Button(action, hi.Text("Save"))
			for bits := 0; bits < 8; bits++ {
				buttons = append(buttons, shared.Selected(bits&1 != 0).MenuOpen(bits&2 != 0).Disabled(bits&4 != 0))
			}
			v := hi.VStack(buttons...).ButtonStyle(style).Frame(hi.Height(400)).Class(fmt.Sprintf("case-%d", len(views)))
			views = append(views, v)
			names = append(names, fmt.Sprintf("%d/%T", style, action))
		}
	}
	stage(t, hi.VStack(views...), func(s *uitest.Session) {
		for i, name := range names {
			t.Run(name, func(t *testing.T) {
				var got []struct {
					Tree     string
					Disabled bool
				}
				s.Eval(fmt.Sprintf(`Array.from(document.querySelector('.case-%d').querySelectorAll('button, a'), e => ({
					Tree: Array.from(e.querySelectorAll('*'), n => n.localName).join(','),
					Disabled: e.disabled === true || e.getAttribute('aria-disabled') === 'true'
				}))`, i), &got)
				if len(got) != 8 {
					t.Fatalf("controls = %d", len(got))
				}
				for i := range 8 {
					if got[i].Tree != "hi-text,span" || got[i].Disabled != (i&4 != 0) {
						t.Errorf("state %d = %+v", i, got[i])
					}
				}
			})
		}
	})
}

func TestButtonStateSemantics(t *testing.T) {
	t.Parallel()
	for _, action := range []any{Msg{}, "/movies"} {
		for _, disabled := range []bool{false, true} {
			html := render(t, hi.Button(action, hi.Text("Save")).Disabled(disabled))
			if disabled && strings.Contains(html, `href="/movies"`) {
				t.Error("disabled URL has navigation target")
			}
			if strings.Contains(html, "aria-pressed") || strings.Contains(html, "aria-expanded") {
				t.Error("button assigns application-owned state semantics")
			}
		}
	}
}
