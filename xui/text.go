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
	return textView{base{textNode{textLeaf(s)}}}
}

type textView struct{ base }

func (v textView) Bold() TextView { return v.styledWith(func(s *textStyle) { s.bold = true }) }

func (v textView) Italic() TextView { return v.styledWith(func(s *textStyle) { s.italic = true }) }

func (v textView) Monospace() TextView { return v.styledWith(func(s *textStyle) { s.mono = true }) }

func (v textView) TextFont(f FontSize) TextView {
	return v.styledWith(func(s *textStyle) { s.font = f })
}

func (v textView) TextForeground(c Color) TextView {
	cc := c.color()
	return v.styledWith(func(s *textStyle) { s.color = cc })
}

func (v textView) Concat(t TextView) TextView {
	v.base = base{textNode{textConcat{v.node().run, t.node().run}}}
	return v
}

func (v textView) node() textNode { return v.base[0].(textNode) }

// styledWith returns a copy of v whose run is modified by f.
func (v textView) styledWith(f func(*textStyle)) TextView {
	v.base = base{textNode{textMod{f: f, run: v.node().run}}}
	return v
}

// textStyle is the pending styling of a text run.
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

// textNode is the box of a text view.
// It lowers the view's run as a block of text.
type textNode struct{ run textRun }

func (n textNode) render(env environment) box {
	env.tag = cmp.Or(env.tag, "ui-text")
	env.style.Set("display", "block")
	env.style.Set("overflow-wrap", "break-word")
	return build(env, plan{content: n.run.html(textenv{sheet: env.sheet})})
}

// A textRun is a unary text node.
// Its whole job is to lower itself to HTML.
type textRun interface {
	html(textenv) domi.Node
}

// textenv carries the top-down state of a text lowering pass.
type textenv struct {
	sheet *sheet.Sheet
	style textStyle // must be consumed before lowering a subrun
}

// styled lowers content inside the pending style, if any,
// consuming it so that no subrun applies it again.
func (env textenv) styled(content func(textenv) domi.Node) domi.Node {
	s := env.style
	if s == (textStyle{}) {
		return content(env)
	}
	env.style = textStyle{}
	return html.Span(s.attr(env.sheet))(content(env))
}

// textLeaf is a run of plain text.
type textLeaf string

func (l textLeaf) html(env textenv) domi.Node {
	return env.styled(func(textenv) domi.Node {
		return domi.Text(string(l))
	})
}

// textConcat is the concatenation of its runs.
type textConcat []textRun

func (c textConcat) html(env textenv) domi.Node {
	return env.styled(func(env textenv) (n domi.Node) {
		for _, r := range c {
			n = domi.Fragment(n, r.html(env))
		}
		return n
	})
}

// textMod applies f to the pending style of its run.
type textMod struct {
	f   func(*textStyle)
	run textRun
}

func (m textMod) html(env textenv) domi.Node {
	m.f(&env.style)
	return m.run.html(env)
}
