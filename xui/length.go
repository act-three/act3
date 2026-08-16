package ui

import (
	"fmt"
)

// A Length is a fixed length in CSS pixels.
type Length interface {
	int | float64
}

type rect struct{ width, height size }

// size is either a fixed length in CSS pixels or the special value Auto.
// The meaning of Auto is determined by context.
// It has no universally-defined lowering form.
type size struct {
	definite bool
	px       float64
}

// newSize lowers a [Length] or [Auto] argument to its internal representation.
func newSize[L Length | Auto](v L) size {
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
