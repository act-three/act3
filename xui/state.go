package ui

import (
	"slices"
	"strconv"
	"strings"

	"ily.dev/act3/xui/internal/sheet"
)

// A State identifies a dynamic state of a rendered view,
// such as the pointer being over it.
//
// [View.Modify] can associate a modifier with one or more states.
type State int

const (
	// Hovered is active while the pointer is over the view.
	Hovered State = 1 << iota

	// Focused is active while the view has visible keyboard focus.
	Focused

	// Pressed is active while the view is activated,
	// such as during a click.
	Pressed

	// Disabled is active when the view is a disabled control.
	Disabled

	// Checked is active when the view is a control while in its
	// checked or selected state.
	Checked

	// Invalid is active when the view is a control while its value
	// fails validation after the user has interacted with it.
	Invalid

	// Placeholder is active when the view is a text control
	// while showing its placeholder in lieu of a value.
	Placeholder
)

// pseudoClasses lists the selector for each state,
// in the order they are spelled within a compound selector.
var pseudoClasses = [...]struct {
	state State
	sel   string
}{
	{Hovered, ":hover"},
	{Focused, ":focus-visible"},
	{Pressed, ":active"},
	// Elements with no disabled attribute, such as anchors,
	// are disabled by aria-disabled instead.
	{Disabled, `:is(:disabled, [aria-disabled="true"])`},
	{Checked, ":checked"},
	{Invalid, ":user-invalid"},
	{Placeholder, ":placeholder-shown"},
}

// pseudo returns the pseudo-class selector suffix that matches an
// element exactly when every state in s is active.
func (s State) pseudo() string {
	var b strings.Builder
	for _, pc := range pseudoClasses {
		if s&pc.state != 0 {
			b.WriteString(pc.sel)
		}
	}
	return b.String()
}

// media returns the media feature query gating s, if any.
// Hover styling applies only on devices that can hover,
// so touch interfaces don't stick in a hover state after a tap.
func (s State) media() string {
	if s&Hovered != 0 {
		return "(hover: hover)"
	}
	return ""
}

// A term value takes effect only while every state in state is
// active. The zero state applies always.
type term[T any] struct {
	state State
	value T
}

// appliesUnder reports whether the term takes effect while exactly
// the states in s are active.
func (t term[T]) appliesUnder(s State) bool { return t.state&s == t.state }

// lastUnder returns the value of the last term applying under s,
// or the zero value if none applies.
func lastUnder[T any](terms []term[T], s State) (v T) {
	for _, t := range terms {
		if t.appliesUnder(s) {
			v = t.value
		}
	}
	return v
}

// allUnder returns the values of the terms applying under s, in order.
func allUnder[T any](terms []term[T], s State) (vs []T) {
	for _, t := range terms {
		if t.appliesUnder(s) {
			vs = append(vs, t.value)
		}
	}
	return vs
}

// paint is a box's effective paint while a given state set is active:
// each term list folded down to its final value.
type paint struct {
	fontFamily string
	fontSize   string
	fontStyle  string
	fontWeight string
	lineHeight string
	fg         color
	bg         []color
	stroke     []stroke
	shape      Shape
	opacity    float64 // 1 is opaque
}

// paintUnder folds b's paint terms into the effective paint
// under state set s.
func (b nextenv) paintUnder(s State) paint {
	opacity := 1.0
	for _, t := range b.opacity {
		if t.appliesUnder(s) {
			opacity *= t.value
		}
	}
	return paint{
		fontFamily: lastUnder(b.fontFamily, s),
		fontSize:   lastUnder(b.fontSize, s),
		fontStyle:  lastUnder(b.fontStyle, s),
		fontWeight: lastUnder(b.fontWeight, s),
		lineHeight: lastUnder(b.lineHeight, s),
		fg:         lastUnder(b.fg, s),
		bg:         allUnder(b.bg, s),
		stroke:     allUnder(b.stroke, s),
		shape:      lastUnder(b.shape, s),
		opacity:    opacity,
	}
}

// states returns every state set under which b's paint can differ,
// in ascending order. The sets are closed under union: when several
// active states style the same property, the union set carries their
// combined effect and outweighs each state's own variant.
func (b nextenv) states() []State {
	var sets []State
	add := func(s State) {
		if s != 0 && !slices.Contains(sets, s) {
			sets = append(sets, s)
		}
	}
	for _, s := range b.termStates() {
		add(s)
	}
	// The bounds re-evaluate on purpose: unions the loop appends
	// combine with the earlier sets in later iterations.
	for i := 0; i < len(sets); i++ {
		for j := 0; j < i; j++ {
			add(sets[i] | sets[j])
		}
	}
	slices.Sort(sets)
	return sets
}

// termStates returns the state of every paint term in b.
func (b nextenv) termStates() []State {
	var ss []State
	for _, t := range b.fontFamily {
		ss = append(ss, t.state)
	}
	for _, t := range b.fontSize {
		ss = append(ss, t.state)
	}
	for _, t := range b.fontStyle {
		ss = append(ss, t.state)
	}
	for _, t := range b.fontWeight {
		ss = append(ss, t.state)
	}
	for _, t := range b.lineHeight {
		ss = append(ss, t.state)
	}
	for _, t := range b.fg {
		ss = append(ss, t.state)
	}
	for _, t := range b.bg {
		ss = append(ss, t.state)
	}
	for _, t := range b.stroke {
		ss = append(ss, t.state)
	}
	for _, t := range b.shape {
		ss = append(ss, t.state)
	}
	for _, t := range b.opacity {
		ss = append(ss, t.state)
	}
	return ss
}

// A decl is one CSS declaration.
type decl struct{ property, value string }

// decls returns p's declarations for the element's own selector.
// Stroke declarations ride the ::after carrier instead;
// see addPaintStylesTo.
//
// A complete paint also declares properties at their default values,
// which a state variant needs to override its base.
func (p paint) decls(t theme, complete bool) []decl {
	var ds []decl
	for _, d := range []decl{
		{"font-family", p.fontFamily},
		{"font-size", p.fontSize},
		{"font-style", p.fontStyle},
		{"font-weight", p.fontWeight},
		{"line-height", p.lineHeight},
	} {
		if d.value != "" {
			ds = append(ds, d)
		}
	}
	if len(p.bg) > 0 {
		// The outermost color paints as the background color, and the
		// inner colors as image layers listed innermost first.
		ds = append(ds, decl{"background-color", p.bg[0].colorCoords(t).css()})
		if len(p.bg) > 1 {
			var img []string
			for _, c := range slices.Backward(p.bg[1:]) {
				css := c.colorCoords(t).css()
				img = append(img, "linear-gradient("+css+","+css+")")
			}
			ds = append(ds, decl{"background-image", strings.Join(img, ",")})
		}
	}
	if complete || p.shape != Rectangle {
		ds = append(ds, decl{"border-radius", p.shape.radius()})
	}
	if p.fg != nil {
		ds = append(ds, decl{"color", p.fg.colorCoords(t).css()})
	}
	if complete || p.opacity < 1 {
		ds = append(ds, decl{"opacity", strconv.FormatFloat(p.opacity, 'g', 4, 64)})
	}
	return ds
}

// carrierDecls returns p's declarations for the ::after stroke carrier.
func (p paint) carrierDecls(t theme) []decl {
	if len(p.stroke) == 0 {
		return nil
	}
	return []decl{{"box-shadow", shadowList(t, p.stroke)}}
}

// shadowList returns the strokes as a box-shadow list,
// listed outermost first, so an outer stroke paints over an inner one.
func shadowList(t theme, strokes []stroke) string {
	var shadows []string
	for _, s := range strokes {
		shadows = append(shadows, "inset 0 0 0 "+cssPx(s.px)+" "+s.c.colorCoords(t).css())
	}
	return strings.Join(shadows, ",")
}

// addPaintStylesTo adds a box's paint declarations to ss:
// the zero-state paint on the element itself,
// and a variant per state set,
// declaring the properties that differ from the zero state
// under the matching pseudo-classes.
//
// A union variant also redeclares every property declared by a
// variant of any subset of its states, even at the zero-state value:
// those variants match too while the union is active,
// and only a redeclaration outweighs them.
func addPaintStylesTo(ss *sheet.StyleSet, env environment) {
	b, t := env.nextenv, env.theme
	base := b.paintUnder(0)
	for _, d := range base.decls(t, false) {
		ss.Set(d.property, d.value)
	}
	if len(b.stroke) > 0 {
		// The strokes of every state share one ::after carrier
		// covering the box; each state draws its own shadow list.
		ss.Set("position", "relative")
		ss.SetPseudo("::after", "content", `""`)
		ss.SetPseudo("::after", "position", "absolute")
		ss.SetPseudo("::after", "inset", "0")
		ss.SetPseudo("::after", "border-radius", "inherit")
		ss.SetPseudo("::after", "pointer-events", "none")
		for _, d := range base.carrierDecls(t) {
			ss.SetPseudo("::after", d.property, d.value)
		}
	}
	// declared records the properties each variant declares, so the
	// variants of supersets can redeclare them. Subsets are smaller
	// as integers, so ascending order visits them first.
	declared := make(map[State]map[string]bool)
	for _, s := range b.states() {
		declared[s] = make(map[string]bool)
		v := b.paintUnder(s)
		for _, sel := range []struct {
			suffix     string
			base, want []decl
		}{
			{"", base.decls(t, true), v.decls(t, true)},
			{"::after", base.carrierDecls(t), v.carrierDecls(t)},
		} {
			baseValue := make(map[string]string)
			for _, d := range sel.base {
				baseValue[d.property] = d.value
			}
			for _, d := range sel.want {
				key := sel.suffix + d.property
				if baseValue[d.property] == d.value && !declaredBelow(declared, s, key) {
					continue
				}
				declared[s][key] = true
				if media := s.media(); media != "" {
					ss.SetMediaPseudo(media, s.pseudo()+sel.suffix, d.property, d.value)
				} else {
					ss.SetPseudo(s.pseudo()+sel.suffix, d.property, d.value)
				}
			}
		}
	}
}

// declaredBelow reports whether the variant of any proper subset of s
// declares key.
func declaredBelow(declared map[State]map[string]bool, s State, key string) bool {
	for t, keys := range declared {
		if t != s && t&s == t && keys[key] {
			return true
		}
	}
	return false
}
