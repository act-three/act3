package ui

import (
	"fmt"
)

// EdgeSpace specifies spacing distances for the edges of a rectangle.
type EdgeSpace struct {
	Top, Bottom, Leading, Trailing float64
}

// Edges sets all four edges to px pixels.
func Edges[L Length](px L) EdgeSpace {
	v := float64(px)
	return EdgeSpace{v, v, v, v}
}

// EdgesPillarbox sets the leading and trailing edges to px pixels.
func EdgesPillarbox[L Length](px L) EdgeSpace {
	v := float64(px)
	return EdgeSpace{Leading: v, Trailing: v}
}

// EdgesLetterbox sets the top and bottom edges to px pixels.
func EdgesLetterbox[L Length](px L) EdgeSpace {
	v := float64(px)
	return EdgeSpace{Top: v, Bottom: v}
}

// EdgeTop sets the top edge to px pixels.
func EdgeTop[L Length](px L) EdgeSpace { return EdgeSpace{Top: float64(px)} }

// EdgeBottom sets the bottom edge to px pixels.
func EdgeBottom[L Length](px L) EdgeSpace { return EdgeSpace{Bottom: float64(px)} }

// EdgeLeading sets the leading edge to px pixels.
func EdgeLeading[L Length](px L) EdgeSpace { return EdgeSpace{Leading: float64(px)} }

// EdgeTrailing sets the trailing edge to px pixels.
func EdgeTrailing[L Length](px L) EdgeSpace { return EdgeSpace{Trailing: float64(px)} }

// add returns s with o's spacing added to each edge.
func (s EdgeSpace) add(o EdgeSpace) EdgeSpace {
	s.Top += o.Top
	s.Bottom += o.Bottom
	s.Leading += o.Leading
	s.Trailing += o.Trailing
	return s
}

// setPadding adds the shortest equivalent padding declarations to x.
func (s EdgeSpace) setPadding(x *box) {
	t, b := cssPx(s.Top), cssPx(s.Bottom)
	le, tr := cssPx(s.Leading), cssPx(s.Trailing)
	if t == b && le == tr && t == le {
		x.setStyle("padding", t)
		return
	}
	block, inline := t, le
	if b != t {
		block += " " + b
	}
	if tr != le {
		inline += " " + tr
	}
	x.setStyle("padding-block", block)
	x.setStyle("padding-inline", inline)
}

func cssPx(v float64) string { return fmt.Sprintf("%gpx", v) }
