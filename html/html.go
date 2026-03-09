// Package html representes HTML as Go function calls.
// Construct trees of HTML nodes:
//
//	h := Div(attr.Class("text-red-50"))(
//	    P()(Text("Watch ")), I()(Text("Casablanca")), Text(" tonight!")),
//	)
//
// Render to HTML code:
//
//	b := &bytes.Buffer{}
//	Render(&b, h)
//	// b contains <div class=text-red-50><p>Watch <i>Casablanca</i> tonight!</p></div>
package html

import (
	"io"

	"ily.dev/act3/html/attr"
)

//go:generate go run gen.go -output tag.go

// Render writes n to w as HTML code.
// The only errors returned are from w.
func Render(w io.Writer, n Node) error {
	return n.renderTo(w)
}

// Node is an HTML node (element or text).
// Its complete tree (including children)
// can be rendered to HTML code using [Render].
type Node interface {
	renderTo(io.Writer) error
}

// Tag returns a new HTML tag with the given name.
// The tag is a function;
// call it to add attributes.
func Tag(name string) func(...attr.Node) Element {
	return func(attrs ...attr.Node) Element {
		return func(child ...Node) Node {
			return element{
				tag:   name,
				attrs: attr.Group(attrs...),
				child: Group(child...),
			}
		}
	}
}

// Element represents an open HTML element,
// with tag name and attributes as given by [Tag].
//
// An Element is a function; call it to add children.
// For convenience, Element also implements Node;
// Render(w, e) is equivalent to Render(w, e()).
type Element func(...Node) Node

// renderTo renders e with no children.
// To add children, call e(child1, child2, ...).
func (e Element) renderTo(w io.Writer) error {
	return e().renderTo(w)
}

type element struct {
	tag   string
	attrs attr.Node
	child Node
}

func (e element) renderTo(w io.Writer) error {
	w.Write(lt)
	io.WriteString(w, e.tag)
	w.Write(space)
	err := attr.Render(w, e.attrs)
	if err != nil {
		return err
	}
	if !isVoid[e.tag] {
		w.Write(gt)
		err = e.child.renderTo(w)
		if err != nil {
			return err
		}
		w.Write(ltSlash)
		io.WriteString(w, e.tag)
	}
	_, err = w.Write(gt)
	return err
}

var (
	lt      = []byte(`<`)
	gt      = []byte(`>`)
	ltSlash = []byte(`</`)
	space   = []byte(` `)
)

var isVoid = map[string]bool{
	"area":   true,
	"base":   true,
	"br":     true,
	"col":    true,
	"embed":  true,
	"hr":     true,
	"img":    true,
	"input":  true,
	"link":   true,
	"meta":   true,
	"param":  true,
	"source": true,
	"track":  true,
	"wbr":    true,
}
