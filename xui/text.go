package ui

import (
	"cmp"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/xui/internal/sheet"
)

// A TextView displays one or more lines of read-only text.
type TextView interface {
	View

	// Concat concatenates the receiver with t.
	Concat(t TextView) TextView

	// Bold uses a bold font to draw the text in the receiver.
	Bold() TextView

	// Italic uses an italic font to draw the text in the receiver.
	Italic() TextView

	// Monospace uses a monospace font to draw the text in the receiver.
	Monospace() TextView

	// TextFont sets the font size for the text in the receiver.
	//
	// It is equivalent to Font, but it returns a TextView.
	TextFont(FontSize) TextView

	// TextForeground uses c to draw the text in the receiver.
	//
	// It is equivalent to Foreground, but it returns a TextView.
	TextForeground(c Color) TextView

	node() textNode
}

// Text displays s.
func Text(s string) TextView {
	return textView{base{textNode{text: s}}}
}

type textView struct{ base }

func (v textView) Bold() TextView { return v.styledWith(func(s *textStyle) { s.bold = true }) }

func (v textView) Italic() TextView { return v.styledWith(func(s *textStyle) { s.italic = true }) }

func (v textView) Monospace() TextView { return v.styledWith(func(s *textStyle) { s.mono = true }) }

func (v textView) TextFont(f FontSize) TextView {
	return v.styledWith(func(s *textStyle) {
		if s.font == "" {
			s.font = f
		}
	})
}

func (v textView) TextForeground(c Color) TextView {
	cc := c.color()
	return v.styledWith(func(s *textStyle) {
		if s.color == nil {
			s.color = cc
		}
	})
}

func (v textView) Concat(t TextView) TextView {
	v.base = base{textNode{parts: []textNode{v.node(), t.node()}}}
	return v
}

func (v textView) node() textNode { return v.base[0].(textNode) }

// styledWith returns a copy of v with f applied to its style.
func (v textView) styledWith(f func(*textStyle)) TextView {
	n := v.node()
	f(&n.style)
	v.base = base{n}
	return v
}

// textStyle is the styling a text node applies to its whole subtree.
type textStyle struct {
	bold, italic, mono bool
	font               FontSize
	color              color
}

func (s textStyle) attr(sh *sheet.Sheet) domi.Attr {
	var ss sheet.StyleSet
	if s.mono {
		ss.Set("font-family", "var(--ui-font-mono)")
	}
	if s.bold {
		ss.Set("font-weight", "600")
	}
	if s.italic {
		ss.Set("font-style", "italic")
	}
	// The type scale's weight beats Bold's, matching the write order.
	s.font.setStyles(&ss)
	if s.color != nil {
		ss.Set("color", s.color.colorCSS())
	}
	return attr.Class(sh.ClassFor(ss))
}

// textNode is a tree of styled text nodes. It has 2 cases:
//   - leaf text
//   - concatenation of subtrees
type textNode struct {
	text  string     // leaf text
	parts []textNode // concatenation members, or nil for leaf text
	style textStyle
}

func (n textNode) render(env environment) box {
	env.tag = cmp.Or(env.tag, "ui-text")
	env.style.Set("display", "block")
	env.style.Set("overflow-wrap", "break-word")
	return build(env, plan{content: n.html(env.sheet)})
}

// html lowers n (with its own style, if any).
func (n textNode) html(sh *sheet.Sheet) domi.Node {
	if n.style == (textStyle{}) {
		return n.content(sh)
	}
	return html.Span(n.style.attr(sh))(n.content(sh))
}

// content lowers n's content without its own style.
// Any subtrees still lower their style.
func (n textNode) content(sh *sheet.Sheet) domi.Node {
	if n.parts == nil {
		return domi.Text(n.text)
	}
	var out []domi.Node
	for _, p := range n.parts {
		out = append(out, p.html(sh))
	}
	return domi.Fragment(out...)
}
