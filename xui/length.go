package ui

import (
	"fmt"
)

type rect struct{ width, height size }

// size is either a fixed length in CSS pixels or the special value Auto.
// The meaning of Auto is determined by context.
type size struct {
	definite bool
	px       float64
}

// newSize converts a generic size to its internal representation.
func newSize[Size int | float64 | Auto](v Size) size {
	switch v := any(v).(type) {
	case Auto:
		return size{}
	case int:
		return size{definite: true, px: float64(v)}
	case float64:
		return size{definite: true, px: v}
	}
	panic("unreachable")
}

func (l size) css() string {
	if !l.definite {
		// Auto emits no declaration at all; lowering it is a bug.
		panic("ui: no CSS for Auto")
	}
	return fmt.Sprintf("%gpx", l.px)
}
