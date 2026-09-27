package hi

import "ily.dev/act3/hi/internal/sheet"

// A position specifies a box's placement within its context,
// as well as how its children are placed within itself.
// CSS conflates these independent concepts, but we try to
// represent them orthogonally.
type position struct {
	exterior exteriorPosition
	interior bool // make a containing block for absolute descendants
}

type exteriorPosition int

const (
	positionFlow exteriorPosition = iota
	positionSticky
	positionAbsolute
	positionFixed
)

func (p position) setOn(ss *sheet.StyleSet) {
	value := [...]string{"", "sticky", "absolute", "fixed"}[p.exterior]
	// Sticky, absolute, and fixed already establish a containing block.
	// In normal flow, relative provides one without moving the box.
	if p.exterior == positionFlow && p.interior {
		value = "relative"
	}
	if value != "" {
		ss.Set("position", value)
	}
}
