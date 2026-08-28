// Package canon restricts CSS declarations to canonical properties.
//
// A [StyleSet] admits a fixed set of properties,
// chosen so that no two of them share a cascade slot:
// a shorthand is never admitted alongside its longhands,
// nor a physical edge alongside its logical one.
// Assigning an admitted property therefore
// replaces every earlier assignment to its slot.
// Assigning any other property panics;
// the panic names the admitted properties for its slot, if any.
package canon

import (
	"fmt"

	"ily.dev/act3/xui/internal/sheet"
)

// A StyleSet is a set of CSS declarations over canonical properties.
//
// The zero value is an empty set, ready to use.
type StyleSet struct {
	decls sheet.StyleSet
}

// Set is [sheet.StyleSet.Set] for a canonical property.
// It panics on any other property.
func (s *StyleSet) Set(property, value string) {
	check(property)
	s.decls.Set(property, value)
}

// SetPseudo is [sheet.StyleSet.SetPseudo] for a canonical property.
// It panics on any other property.
func (s *StyleSet) SetPseudo(pseudo, property, value string) {
	check(property)
	s.decls.SetPseudo(pseudo, property, value)
}

// Decls returns the declarations in s as a [sheet.StyleSet].
// The result is a copy: writing to it does not affect s.
func (s StyleSet) Decls() sheet.StyleSet {
	return s.decls
}

func check(property string) {
	if canonical[property] {
		return
	}
	if instead, ok := rejected[property]; ok {
		panic(fmt.Sprintf("canon: cannot set %q; use %s", property, instead))
	}
	panic(fmt.Sprintf("canon: %q is not a canonical property", property))
}

// canonical is the set of admitted properties.
var canonical = map[string]bool{
	"-webkit-box-orient":    true,
	"-webkit-line-clamp":    true,
	"align-items":           true,
	"align-self":            true,
	"aspect-ratio":          true,
	"column-gap":            true,
	"contain":               true,
	"cursor":                true,
	"display":               true,
	"flex-basis":            true,
	"flex-direction":        true,
	"flex-grow":             true,
	"flex-shrink":           true,
	"grid-column-end":       true,
	"grid-column-start":     true,
	"grid-row-end":          true,
	"grid-row-start":        true,
	"grid-template-columns": true,
	"grid-template-rows":    true,
	"height":                true,
	"inset-block-end":       true,
	"inset-block-start":     true,
	"inset-inline-end":      true,
	"inset-inline-start":    true,
	"isolation":             true,
	"justify-items":         true,
	"justify-self":          true,
	"min-height":            true,
	"min-width":             true,
	"object-fit":            true,
	"overflow-wrap":         true,
	"overflow-x":            true,
	"overflow-y":            true,
	"overscroll-behavior-x": true,
	"overscroll-behavior-y": true,
	"padding-block-end":     true,
	"padding-block-start":   true,
	"padding-inline-end":    true,
	"padding-inline-start":  true,
	"pointer-events":        true,
	"position":              true,
	"row-gap":               true,
	"translate":             true,
	"width":                 true,
	"writing-mode":          true,
	"z-index":               true,
}

// rejected names, for properties one might reach for,
// what to write instead.
var rejected = map[string]string{
	"overflow":            "overflow-x and overflow-y",
	"overscroll-behavior": "overscroll-behavior-x and overscroll-behavior-y",
	"place-items":         "align-items and justify-items",
	"place-self":          "align-self and justify-self",
	"gap":                 "row-gap and column-gap",
	"grid-area":           "grid-row-start, grid-column-start, grid-row-end, and grid-column-end",
	"grid-row":            "grid-row-start and grid-row-end",
	"grid-column":         "grid-column-start and grid-column-end",
	"flex":                "flex-grow, flex-shrink, and flex-basis",
	"inset":               edgeLonghands("inset"),
	"inset-block":         edgeLonghands("inset"),
	"inset-inline":        edgeLonghands("inset"),
	"top":                 edgeLonghands("inset"),
	"right":               edgeLonghands("inset"),
	"bottom":              edgeLonghands("inset"),
	"left":                edgeLonghands("inset"),
	"padding":             edgeLonghands("padding"),
	"padding-block":       edgeLonghands("padding"),
	"padding-inline":      edgeLonghands("padding"),
	"padding-top":         edgeLonghands("padding"),
	"padding-right":       edgeLonghands("padding"),
	"padding-bottom":      edgeLonghands("padding"),
	"padding-left":        edgeLonghands("padding"),
	"inline-size":         "width",
	"block-size":          "height",
	"min-inline-size":     "min-width",
	"min-block-size":      "min-height",
	"background":          "the Background modifier",
	"background-color":    "the Background modifier",
	"background-image":    "the Background modifier",
	"border-radius":       "the BorderShape modifier",
	"box-shadow":          "the BorderStroke modifier",
	"color":               "the Foreground modifier",
	"font":                "the font modifiers",
	"font-family":         "the font modifiers",
	"font-size":           "the font modifiers",
	"font-style":          "the font modifiers",
	"font-weight":         "the font modifiers",
	"line-height":         "the font modifiers",
	"opacity":             "the Opacity modifier",
}

func edgeLonghands(property string) string {
	return fmt.Sprintf("%[1]s-block-start, %[1]s-block-end, %[1]s-inline-start, and %[1]s-inline-end", property)
}
