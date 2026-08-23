package ui

import (
	"cmp"
	"strconv"

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

	text() textRun
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
	v.base = base{textNode{textConcat{v.text(), t.text()}}}
	return v
}

func (v textView) text() textRun { return v.base[0].(textBox).text() }

// styledWith returns a copy of v whose run is modified by f.
func (v textView) styledWith(f func(*textStyle)) TextView {
	v.base = base{textNode{textMod{f: f, run: v.text()}}}
	return v
}

// A textBox is the box of a text view.
// It lowers a run of text as a block,
// and exposes the run for use in a larger text.
type textBox interface {
	node
	text() textRun
}

// textStyle is the pending styling of a text run.
type textStyle struct {
	bold, italic, mono bool
	font               FontSize
	color              color
}

func (s textStyle) attr(sh *sheet.Sheet) domi.Attr {
	var ss sheet.StyleSet
	s.setStyles(&ss)
	return attr.Class(sh.ClassFor(ss))
}

func (s textStyle) setStyles(ss *sheet.StyleSet) {
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
	s.font.setStyles(ss)
	if s.color != nil {
		ss.Set("color", s.color.colorCSS())
	}
}

// textNode is the box of a plain text view.
type textNode struct{ run textRun }

func (n textNode) text() textRun { return n.run }

func (n textNode) render(env environment) box {
	env.tag = cmp.Or(env.tag, "ui-text")
	env.style.Set("display", "block")
	env.style.Set("overflow-wrap", "break-word")
	if env.lineLimit > 0 {
		env.style.Set("display", "-webkit-box")
		env.style.Set("-webkit-box-orient", "vertical")
		env.style.Set("-webkit-line-clamp", strconv.Itoa(env.lineLimit))
		env.style.Set("overflow", "clip")
	}
	tenv := textenv{sheet: env.sheet, disabled: env.disabled}
	return build(env, plan{content: n.run.html(tenv)})
}

// A textRun is a unary text node.
// Its whole job is to lower itself to HTML.
type textRun interface {
	html(textenv) domi.Node
}

// textenv carries the top-down state of a text lowering pass.
type textenv struct {
	sheet    *sheet.Sheet
	disabled bool
	style    textStyle // must be consumed before lowering a subrun
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
