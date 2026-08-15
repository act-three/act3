// Package sheet collects dynamic CSS declarations into reusable class rules.
package sheet

import (
	"fmt"
	"hash/fnv"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// A StyleSet holds the CSS declarations for one element,
// including declarations under pseudo-classes and pseudo-elements.
//
// It may be copied.
//
// The zero value is an empty set, ready to use.
type StyleSet struct {
	m map[string]map[string]string // ":hover" → "color" → "red"
}

// Style returns a new StyleSet containing one declaration.
func Style(property, value string) StyleSet {
	var s StyleSet
	s.Set(property, value)
	return s
}

// Set assigns value to property.
// It replaces the previous value, if any.
//
// Set panics if the declaration cannot be safely inserted into a CSS rule.
func (s *StyleSet) Set(property, value string) {
	s.set("", property, value)
}

// SetPseudo assigns value to property
// under the given pseudo-class or pseudo-element,
// such as ":hover" or "::after".
// It replaces the previous value, if any.
//
// SetPseudo panics
// if the declaration cannot be safely inserted into a CSS rule.
func (s *StyleSet) SetPseudo(pseudo, property, value string) {
	if !validPseudo(pseudo) {
		panic(fmt.Sprintf("sheet: invalid pseudo selector %q", pseudo))
	}
	s.set(pseudo, property, value)
}

func (s *StyleSet) set(suffix, property, value string) {
	if !validProperty(property) {
		panic(fmt.Sprintf("sheet: invalid CSS property %q", property))
	}
	if strings.ContainsAny(value, "{};") || strings.ContainsFunc(value, isCTL) {
		panic(fmt.Sprintf("sheet: invalid CSS value %q", value))
	}
	m := maps.Clone(s.m)
	if m == nil {
		m = make(map[string]map[string]string, 1)
	}
	d := maps.Clone(m[suffix])
	if d == nil {
		d = make(map[string]string, 1)
	}
	d[property] = value
	m[suffix] = d
	s.m = m
}

// IsEmpty reports whether s has no declarations.
func (s StyleSet) IsEmpty() bool { return len(s.m) == 0 }

// body returns the set as a rule body:
// the element's declarations sorted by property,
// followed by a nested block per pseudo selector, sorted by selector.
func (s StyleSet) body() string {
	var b strings.Builder
	writeDecls(&b, s.m[""])
	sep := b.Len() > 0
	for _, suffix := range slices.Sorted(maps.Keys(s.m)) {
		if suffix == "" {
			continue
		}
		if sep {
			b.WriteByte(';')
			sep = false
		}
		b.WriteByte('&')
		b.WriteString(suffix)
		b.WriteByte('{')
		writeDecls(&b, s.m[suffix])
		b.WriteByte('}')
	}
	return b.String()
}

// writeDecls writes the declarations sorted by property and joined with semicolons.
func writeDecls(b *strings.Builder, body map[string]string) {
	for i, p := range slices.Sorted(maps.Keys(body)) {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(p)
		b.WriteByte(':')
		b.WriteString(body[p])
	}
}

// validPseudo reports whether p can be safely nested as a selector suffix:
// it must start with a colon and cannot escape its rule.
func validPseudo(p string) bool {
	return strings.HasPrefix(p, ":") &&
		!strings.ContainsAny(p, "{};") &&
		!strings.ContainsFunc(p, isCTL)
}

func validProperty(p string) bool {
	if p == "" {
		return false
	}
	for _, c := range p {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c != '-' {
			return false
		}
	}
	return true
}

func isCTL(c rune) bool { return c < ' ' || c == 0x7f }

// A Sheet assigns generated class names to sets of CSS declarations.
// It retains every class it creates.
//
// The zero value is an empty sheet, ready to use.
type Sheet struct {
	classes map[string]string // canonical rule body → class name
	byClass map[string]string // class name → canonical rule body
	rules   []string
}

// ClassFor returns the generated class name for s.
//
// The first call to ClassFor
// for a given set of declarations
// also adds a rule to sh.
//
// Equal declaration sets get the same class name in every Sheet.
//
// An empty s has no class: ClassFor returns the empty string
// and adds nothing to sh.
func (sh *Sheet) ClassFor(s StyleSet) string {
	if s.IsEmpty() {
		return ""
	}
	body := s.body()
	if class, ok := sh.classes[body]; ok {
		return class
	}
	class := className(body)
	if prior, ok := sh.byClass[class]; ok && prior != body {
		panic(fmt.Sprintf("sheet: class %s collides: %q vs %q", class, prior, body))
	}
	if sh.classes == nil {
		sh.classes = make(map[string]string)
		sh.byClass = make(map[string]string)
	}
	sh.classes[body] = class
	sh.byClass[class] = body
	sh.rules = append(sh.rules, "."+class+"{"+body+"}")
	return class
}

// CSS returns one rule for each declaration set.
// Rules appear in the order they were added.
func (sh *Sheet) CSS() string { return strings.Join(sh.rules, "\n") }

// className returns a deterministic class name for body.
func className(body string) string {
	h := fnv.New64a()
	h.Write([]byte(body))
	name := strconv.FormatUint(h.Sum64(), 32)
	if len(name) > 8 {
		name = name[:8]
	}
	return "ui-" + name
}
