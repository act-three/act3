package html

import (
	"io"

	"ily.dev/act3/html/attr"
)

//go:generate go run gen.go -output tag.go

func Render(w io.Writer, n Node) error {
	return n.renderTo(w)
}

type Node interface {
	renderTo(io.Writer) error
}

// Tag returns a new HTML tag with the given name.
// The tag is a function;
// it should be called with a list of attributes.
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

// Element represents an element.
// It already contains attributes.
// Children can be added by calling it.
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

func (element) node() {}

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
