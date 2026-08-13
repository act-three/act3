package ui

import (
	"fmt"

	"ily.dev/act3/xui/internal/sheet"
)

// presentation is a box's styling in the package's own model.
// Each field is nil when the box does not emit that value.
// Values stay in model terms until [presentation.lower]
// translates them to CSS declarations at build time.
type presentation struct {
	color      *Color // foreground
	background *Color
	opacity    *float64
}

// lower adds the CSS declarations for p's values to set.
func (p presentation) lower(set *sheet.StyleSet) {
	if p.color != nil {
		set.Set("color", string(*p.color))
	}
	if p.background != nil {
		set.Set("background-color", string(*p.background))
	}
	if p.opacity != nil {
		set.Set("opacity", fmt.Sprintf("%g", *p.opacity))
	}
}
