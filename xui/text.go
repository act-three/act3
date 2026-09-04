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
	return newTextView(textLeaf(s))
}

type textView struct {
	base
	run textRun
}

func newTextView(r textRun) textView { return textView{base{r.render}, r} }

func (v textView) Bold() TextView {
	return v.styledWith(func(env *environment) {
		env.fontWeight = append(env.fontWeight, term[string]{value: "600"})
	})
}

func (v textView) Italic() TextView {
	return v.styledWith(func(env *environment) {
		env.fontStyle = append(env.fontStyle, term[string]{value: "italic"})
	})
}

func (v textView) Monospace() TextView {
	return v.styledWith(func(env *environment) {
		env.fontFamily = append(env.fontFamily, term[string]{value: "var(--ui-font-mono)"})
	})
}

func (v textView) TextFont(f FontSize) TextView {
	size, weight, height := f.values()
	if size == "" {
		return v
	}
	return v.styledWith(func(env *environment) {
		env.fontSize = append(env.fontSize, term[string]{value: size})
		env.fontWeight = append(env.fontWeight, term[string]{value: weight})
		env.lineHeight = append(env.lineHeight, term[string]{value: height})
	})
}

func (v textView) TextForeground(c Color) TextView {
	cc := c.color()
	return v.styledWith(func(env *environment) {
		env.fg = append(env.fg, term[color]{value: cc})
	})
}

func (v textView) Concat(t TextView) TextView {
	return newTextView(textConcat{v.run, t.text()})
}

func (v textView) text() textRun { return v.run }

// styledWith returns a copy of v whose run is modified by f.
func (v textView) styledWith(f func(*environment)) textView {
	return newTextView(textMod{f: f, run: v.run})
}

// buildText lowers r as a text block.
func buildText(env environment, r textRun) box {
	env.tag = cmp.Or(env.tag, "ui-text")
	env.style.Set("display", "block")
	env.style.Set("overflow-wrap", "break-word")
	if env.lineLimit > 0 {
		env.style.Set("display", "-webkit-box")
		env.style.Set("-webkit-box-orient", "vertical")
		env.style.Set("-webkit-line-clamp", strconv.Itoa(env.lineLimit))
		env.style.Set("overflow-x", "clip")
		env.style.Set("overflow-y", "clip")
	}
	inner := env
	inner.nextenv = nextenv{}
	return build(env, plan{content: r.renderText(inner)})
}

// A textRun is a unary text node.
// It lowers itself to inline HTML as part of an enclosing text,
// or renders itself as a box when it is the whole view.
type textRun interface {
	render(environment) box
	renderText(environment) domi.Node
}

// styled lowers content inside the pending text styling, if any,
// consuming it so that no subrun applies it again.
func (env environment) styled(content func(environment) domi.Node) domi.Node {
	ds := env.paintUnder(0).decls(env.theme, false)
	env.nextenv = nextenv{}
	if len(ds) == 0 {
		return content(env)
	}
	var ss sheet.StyleSet
	for _, d := range ds {
		ss.Set(d.property, d.value)
	}
	return html.Span(attr.Class(env.sheet.ClassFor(ss)))(content(env))
}

// textLeaf is a run of plain text.
type textLeaf string

var _ textRun = textLeaf("")

func (l textLeaf) render(env environment) box {
	return buildText(env, l)
}

func (l textLeaf) renderText(env environment) domi.Node {
	return env.styled(func(environment) domi.Node {
		return domi.Text(string(l))
	})
}

// textConcat is the concatenation of its runs.
type textConcat []textRun

var _ textRun = textConcat(nil)

func (c textConcat) render(env environment) box {
	return buildText(env, c)
}

func (c textConcat) renderText(env environment) domi.Node {
	return env.styled(func(env environment) (n domi.Node) {
		for _, r := range c {
			n = domi.Fragment(n, r.renderText(env))
		}
		return n
	})
}

// textMod applies f to the pending style of its run.
type textMod struct {
	f   func(*environment)
	run textRun
}

var _ textRun = textMod{}

func (m textMod) render(env environment) box {
	m.f(&env)
	return m.run.render(env)
}

func (m textMod) renderText(env environment) domi.Node {
	m.f(&env)
	return m.run.renderText(env)
}
