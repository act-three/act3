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

// A StyleSet holds the CSS declarations for one element.
//
// It may be copied.
//
// The zero value is an empty set, ready to use.
type StyleSet struct {
	m map[string]string
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
	if !validProperty(property) {
		panic(fmt.Sprintf("sheet: invalid CSS property %q", property))
	}
	if strings.ContainsAny(value, "{};") || strings.ContainsFunc(value, isCTL) {
		panic(fmt.Sprintf("sheet: invalid CSS value %q", value))
	}
	m := maps.Clone(s.m)
	if m == nil {
		m = make(map[string]string, 1)
	}
	m[property] = value
	s.m = m
}

// IsEmpty reports whether s has no declarations.
func (s StyleSet) IsEmpty() bool { return len(s.m) == 0 }

// declarations returns the declarations sorted by property and joined with semicolons.
func (s StyleSet) declarations() string {
	var b strings.Builder
	for _, p := range slices.Sorted(maps.Keys(s.m)) {
		if b.Len() > 0 {
			b.WriteByte(';')
		}
		b.WriteString(p)
		b.WriteByte(':')
		b.WriteString(s.m[p])
	}
	return b.String()
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
	classes map[string]string // canonical declarations → class name
	byClass map[string]string // class name → canonical declarations
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
	decls := s.declarations()
	if class, ok := sh.classes[decls]; ok {
		return class
	}
	class := className(decls)
	if prior, ok := sh.byClass[class]; ok && prior != decls {
		panic(fmt.Sprintf("sheet: class %s collides: %q vs %q", class, prior, decls))
	}
	if sh.classes == nil {
		sh.classes = make(map[string]string)
		sh.byClass = make(map[string]string)
	}
	sh.classes[decls] = class
	sh.byClass[class] = decls
	sh.rules = append(sh.rules, "."+class+"{"+decls+"}")
	return class
}

// CSS returns one rule for each declaration set.
// Rules appear in the order they were added.
func (sh *Sheet) CSS() string { return strings.Join(sh.rules, "\n") }

// className returns a deterministic class name for decls.
func className(decls string) string {
	h := fnv.New64a()
	h.Write([]byte(decls))
	name := strconv.FormatUint(h.Sum64(), 32)
	if len(name) > 8 {
		name = name[:8]
	}
	return "ui-" + name
}
