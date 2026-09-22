package hi

import (
	"regexp"
	"strings"
	"testing"
)

func TestNoAction(t *testing.T) {
	for _, tc := range []struct {
		name string
		view View
	}{
		{"button", Button(noAction{}, Text("Dismiss"))},
		{"link", Link(noAction{}, Text("Dismiss"))},
		{"inline link", Text("Note ").Concat(Link(noAction{}, Text("Dismiss")))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, disabled := range []bool{false, true} {
				_, page := Render(tc.view.Disabled(disabled))
				html := renderNode(t, page)
				button := regexp.MustCompile(`<button\b[^>]*>`).FindString(html)
				if !strings.Contains(button, `type="button"`) {
					t.Fatalf("missing button type in %s", html)
				}
				if strings.Contains(button, " disabled") != disabled {
					t.Errorf("disabled=%t: %s", disabled, button)
				}
				for _, unwanted := range []string{"domi-msg-click", "onclick", "href="} {
					if strings.Contains(html, unwanted) {
						t.Errorf("unexpected %q in %s", unwanted, html)
					}
				}
			}
		})
	}
}
