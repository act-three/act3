package hi

import "ily.dev/act3/hi/internal/canon"

// TextEdgeSet is a set of edges
// at the top and bottom of a text box.
// It is used to trim one or both edges
// for typographic alignment.
// See [View.TextTrim].
//
// The zero value is the empty set.
type TextEdgeSet int

const (
	// TextTop specifies the top of the text as a top edge.
	// This includes ascenders and excludes half-leading.
	TextTop TextEdgeSet = 1 << iota

	// TextCap specifies the height of capital letters as a top edge.
	// This excludes ascenders and half-leading.
	TextCap

	// TextEx specifies the height of lowercase letters as a top edge.
	// This excludes the cap height, ascenders, and half-leading.
	TextEx

	// TextLastBaseline specifies the last baseline as a bottom edge.
	// This excludes descenders and half-leading.
	TextLastBaseline

	// TextBottom specifies the bottom of the text as a bottom edge.
	// This includes descenders and excludes half-leading.
	TextBottom
)

func (s TextEdgeSet) hasAny(x TextEdgeSet) bool { return s&x != 0 }

// addTrimStylesTo adds to ss the CSS declarations
// that trim a text box to the edges in s.
// It adds nothing when s trims neither edge.
func (s TextEdgeSet) addTrimStylesTo(ss *canon.StyleSet) {
	var over, under string
	switch {
	case s.hasAny(TextEx):
		over = "ex"
	case s.hasAny(TextCap):
		over = "cap"
	case s.hasAny(TextTop):
		over = "text"
	}
	switch {
	case s.hasAny(TextLastBaseline):
		under = "alphabetic"
	case s.hasAny(TextBottom):
		under = "text"
	}
	var trim string
	switch {
	case over != "" && under != "":
		trim = "trim-both"
	case over != "":
		trim, under = "trim-start", "text"
	case under != "":
		trim, over = "trim-end", "text"
	default:
		return
	}
	ss.Set("text-box-trim", trim)
	ss.Set("text-box-edge", over+" "+under)
}
