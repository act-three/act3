package ui

import (
	"fmt"

	"ily.dev/act3/xui/internal/sheet"
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

// setPadding adds the shortest equivalent padding declarations to ss.
func (s EdgeSpace) setPadding(ss *sheet.StyleSet) {
	t, b := cssPx(s.Top), cssPx(s.Bottom)
	le, tr := cssPx(s.Leading), cssPx(s.Trailing)
	if t == b && le == tr && t == le {
		ss.Set("padding", t)
		return
	}
	block, inline := t, le
	if b != t {
		block += " " + b
	}
	if tr != le {
		inline += " " + tr
	}
	ss.Set("padding-block", block)
	ss.Set("padding-inline", inline)
}

func cssPx(v float64) string { return fmt.Sprintf("%gpx", v) }
