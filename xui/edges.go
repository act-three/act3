package ui

import (
	"fmt"

	"ily.dev/act3/xui/internal/canon"
)

// EdgeSpace specifies spacing distances for the edges of a rectangle.
type EdgeSpace struct {
	Top, Bottom, Leading, Trailing float64
}

// Edges sets all four edges to px pixels.
func Edges(px float64) EdgeSpace {
	return EdgeSpace{px, px, px, px}
}

// EdgesPillarbox sets the leading and trailing edges to px pixels.
func EdgesPillarbox(px float64) EdgeSpace {
	return EdgeSpace{Leading: px, Trailing: px}
}

// EdgesLetterbox sets the top and bottom edges to px pixels.
func EdgesLetterbox(px float64) EdgeSpace {
	return EdgeSpace{Top: px, Bottom: px}
}

// EdgeTop sets the top edge to px pixels.
func EdgeTop(px float64) EdgeSpace { return EdgeSpace{Top: px} }

// EdgeBottom sets the bottom edge to px pixels.
func EdgeBottom(px float64) EdgeSpace { return EdgeSpace{Bottom: px} }

// EdgeLeading sets the leading edge to px pixels.
func EdgeLeading(px float64) EdgeSpace { return EdgeSpace{Leading: px} }

// EdgeTrailing sets the trailing edge to px pixels.
func EdgeTrailing(px float64) EdgeSpace { return EdgeSpace{Trailing: px} }

// add returns s with o's spacing added to each edge.
func (s EdgeSpace) add(o EdgeSpace) EdgeSpace {
	s.Top += o.Top
	s.Bottom += o.Bottom
	s.Leading += o.Leading
	s.Trailing += o.Trailing
	return s
}

// edgeSum returns the given spacing values added together.
func edgeSum(s ...EdgeSpace) (total EdgeSpace) {
	for _, s := range s {
		total = total.add(s)
	}
	return total
}

// setOn declares s on each logical edge longhand of property,
// such as padding or inset.
func (s EdgeSpace) setOn(ss *canon.StyleSet, property string) {
	ss.Set(property+"-block-start", cssPx(s.Top))
	ss.Set(property+"-block-end", cssPx(s.Bottom))
	ss.Set(property+"-inline-start", cssPx(s.Leading))
	ss.Set(property+"-inline-end", cssPx(s.Trailing))
}

func cssPx(v float64) string { return fmt.Sprintf("%gpx", v) }
