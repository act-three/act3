package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/xui/internal/sheet"
)

// A TextView displays one or more lines of read-only text.
type TextView struct{ base }

// Text displays s.
func Text(s string) TextView {
	return TextView{base{textNode{text: s}}}
}

// Bold uses a bold font to draw the text in v.
func (v TextView) Bold() TextView { return v.styledWith(func(s *textStyle) { s.bold = true }) }

// Italic uses an italic font to draw the text in v.
func (v TextView) Italic() TextView { return v.styledWith(func(s *textStyle) { s.italic = true }) }

// Monospace uses a monospace font to draw the text in v.
func (v TextView) Monospace() TextView { return v.styledWith(func(s *textStyle) { s.mono = true }) }

// TextFont sets the font size for the text in v.
//
// It is equivalent to [View.Font], but it returns a TextView.
func (v TextView) TextFont(f FontSize) TextView {
	return v.styledWith(func(s *textStyle) {
		if s.font == "" {
			s.font = f
		}
	})
}

// TextForeground uses c to draw the text in v.
//
// It is equivalent to [View.Foreground], but it returns a TextView.
func (v TextView) TextForeground(c Color) TextView {
	return v.styledWith(func(s *textStyle) {
		if s.color == "" {
			s.color = c
		}
	})
}

// Concat concatenates v and t.
func (v TextView) Concat(t TextView) TextView {
	v.base = base{textNode{parts: []textNode{v.node(), t.node()}}}
	return v
}

func (v TextView) node() textNode { return v.base[0].(textNode) }

// styledWith returns a copy of v with f applied to its style.
func (v TextView) styledWith(f func(*textStyle)) TextView {
	n := v.node()
	f(&n.style)
	v.base = base{n}
	return v
}

// textStyle is the styling a text node applies to its whole subtree.
type textStyle struct {
	bold, italic, mono bool
	font               FontSize
	color              Color
}

func (s textStyle) attr(sh *sheet.Sheet) domi.Attr {
	var a []domi.Attr
	if s.mono {
		a = append(a, attr.Class("ui-mono"))
	}
	if c := s.font.class(); c != "" {
		a = append(a, attr.Class(c))
	}
	if s.bold {
		a = append(a, attr.Class("ui-bold"))
	}
	if s.italic {
		a = append(a, attr.Class("ui-italic"))
	}
	if s.color != "" {
		a = append(a, attr.Class(sh.ClassFor(sheet.Style("color", string(s.color)))))
	}
	return domi.Group(a...)
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
	env.add(attr.Class("ui-text"))
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
