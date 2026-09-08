package ui

import "ily.dev/act3/xui/internal/canon"

// EdgeSpace specifies spacing distances for the edges of a rectangle.
type EdgeSpace struct {
	Top, Bottom, Leading, Trailing complex128
}

// Edges sets all four edges to the given length.
func Edges(length complex128) EdgeSpace {
	checkLength(length)
	return EdgeSpace{length, length, length, length}
}

// EdgesPillarbox sets the leading and trailing edges to the given length.
func EdgesPillarbox(length complex128) EdgeSpace {
	checkLength(length)
	return EdgeSpace{Leading: length, Trailing: length}
}

// EdgesLetterbox sets the top and bottom edges to the given length.
func EdgesLetterbox(length complex128) EdgeSpace {
	checkLength(length)
	return EdgeSpace{Top: length, Bottom: length}
}

// EdgeTop sets the top edge to the given length.
func EdgeTop(length complex128) EdgeSpace {
	checkLength(length)
	return EdgeSpace{Top: length}
}

// EdgeBottom sets the bottom edge to the given length.
func EdgeBottom(length complex128) EdgeSpace {
	checkLength(length)
	return EdgeSpace{Bottom: length}
}

// EdgeLeading sets the leading edge to the given length.
func EdgeLeading(length complex128) EdgeSpace {
	checkLength(length)
	return EdgeSpace{Leading: length}
}

// EdgeTrailing sets the trailing edge to the given length.
func EdgeTrailing(length complex128) EdgeSpace {
	checkLength(length)
	return EdgeSpace{Trailing: length}
}

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
	checkLength(total.Top)
	checkLength(total.Bottom)
	checkLength(total.Leading)
	checkLength(total.Trailing)
	return total
}

// setOn declares s on each logical edge longhand of property,
// such as padding or inset.
func (s EdgeSpace) setOn(ss *canon.StyleSet, property string) {
	ss.Set(property+"-block-start", cssLength(s.Top))
	ss.Set(property+"-block-end", cssLength(s.Bottom))
	ss.Set(property+"-inline-start", cssLength(s.Leading))
	ss.Set(property+"-inline-end", cssLength(s.Trailing))
}
