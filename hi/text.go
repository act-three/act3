package hi

import (
	"cmp"
	"strconv"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/hi/internal/canon"
	"ily.dev/act3/hi/internal/sheet"
)

// A TextView displays one or more lines of read-only text.
type TextView interface {
	View

	// Concat concatenates the receiver with t.
	Concat(t TextView) TextView

	// TextFont configures typography for text in the receiver.
	//
	// It is equivalent to Font, but it returns a TextView.
	TextFont(...FontOption) TextView

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

func (v textView) TextFont(opts ...FontOption) TextView {
	if len(opts) == 0 {
		return v
	}
	f := font(opts)
	return v.styledWith(func(env *environment) {
		*env = f(*env)
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
	inner := env
	inner.nextenv = nextenv{}
	content := r.renderText(inner)
	if env.lineLimit == 1 {
		// A nowrap block's intrinsic minimum is the entire line, even
		// with min-width:0. The track supplies a zero minimum without
		// discarding the full line's ideal width, so enclosing stacks
		// can size themselves from the text's actual range of sizes.
		// The inner block handles ellipsis and trimming; a multicol
		// block fails both in Safari.
		env.style.Set("display", "grid")
		env.style.Set("grid-template-columns", "minmax(0, max-content)")
		if env.unbounded.hasAll(Horizontal) {
			env.style.Set("width", "max-content")
		}
		var line canon.StyleSet
		line.Set("display", "block")
		line.Set("white-space", "nowrap")
		line.Set("text-overflow", "ellipsis")
		line.Set("overflow-x", "clip")
		line.Set("overflow-y", "visible")
		env.textTrim.addTrimStylesTo(&line)
		content = html.Span(attr.Class(env.sheet.ClassFor(line.Decls())))(content)
		return build(env, plan{content: content})
	}
	if env.lineLimit > 1 {
		env.style.Set("display", "-webkit-box")
		env.style.Set("-webkit-box-orient", "vertical")
		env.style.Set("-webkit-line-clamp", strconv.Itoa(env.lineLimit))
		env.style.Set("overflow-x", "clip")
		env.style.Set("overflow-y", "clip")
	}
	env.textTrim.addTrimStylesTo(&env.style)
	return build(env, plan{content: content})
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
	var ss sheet.StyleSet
	for _, d := range ds {
		ss.Set(d.property, d.value)
	}
	addFontStylesTo(&ss, env)
	env.nextenv = nextenv{}
	if ss.IsEmpty() {
		return content(env)
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
