package canon

import (
	"strings"
	"testing"
)

func TestSet(t *testing.T) {
	for _, tt := range []struct {
		property string
		panics   string // substring of the panic message, or "" for none
	}{
		{"overflow-x", ""},
		{"padding-inline-start", ""},
		{"overflow", `cannot set "overflow"; use overflow-x and overflow-y`},
		{"padding", `use padding-block-start, padding-block-end, padding-inline-start, and padding-inline-end`},
		{"opacity", `use the Opacity modifier`},
		{"margin", `"margin" is not a canonical property`},
	} {
		t.Run(tt.property, func(t *testing.T) {
			defer func() {
				r := recover()
				msg, _ := r.(string)
				if tt.panics == "" && r != nil {
					t.Fatalf("Set(%q) panicked: %v", tt.property, r)
				}
				if tt.panics != "" && !strings.Contains(msg, tt.panics) {
					t.Fatalf("Set(%q) panic = %q, want one containing %q", tt.property, msg, tt.panics)
				}
			}()
			var s StyleSet
			s.Set(tt.property, "0")
		})
	}
}
