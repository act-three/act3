package attr

import (
	"io"
	"reflect"
	"strings"
)

//go:generate go run gen.go -output all.go

func Render(w io.Writer, n Node) error {
	return n.renderTo(w)
}

type Node interface {
	Has(name string) bool
	renderTo(io.Writer) error
	attr()
}

// AttrName represents a named attribute.
// It is a function, it can be called to give the attribute a value.
//
// AttrName itself also satisfies the Node interface.
// When rendered, it produces an "empty attribute":
// it writes just the attribute name, with no equals sign and no value.
type AttrName func(value string) Node

func Attr(name string) AttrName {
	return func(value string) Node {
		return attrValue{name, value}
	}
}

func (a AttrName) Has(name string) bool {
	return a(empty).Has(name)
}

func (a AttrName) renderTo(w io.Writer) error {
	return a(empty).renderTo(w)
}

func (AttrName) attr() {}

type attrValue struct {
	name  string
	value string
}

func (a attrValue) Has(name string) bool {
	return a.name == name
}

func (a attrValue) Name() string {
	return a.name
}

// renderTo writes a to w.
func (a attrValue) renderTo(w io.Writer) error {
	return render(w, a.name, a.value)
}

func (attrValue) attr() {}

func render(w io.Writer, name string, v string) error {
	_, err := io.WriteString(w, name)
	if isSameString(v, empty) {
		return err
	}
	w.Write(equals)

	// An attribute value cannot contain an "ambiguous ampersand".
	// See https://html.spec.whatwg.org/multipage/syntax.html.

	if _, ok := combining[name]; ok {
		err = renderQuoted(w, v)
	} else if !strings.ContainsAny(v, ` "'=<>`+"`") && v != "" {
		v = strings.ReplaceAll(v, `&`, "&amp;")
		_, err = io.WriteString(w, v)
	} else if strings.Contains(v, `"`) && !strings.Contains(v, `'`) {
		v = strings.ReplaceAll(v, `&`, "&amp;")
		w.Write(squote)
		io.WriteString(w, v)
		_, err = w.Write(squote)
	} else {
		err = renderQuoted(w, v)
	}
	return err
}

func renderQuoted(w io.Writer, v string) error {
	v = doubleQuoteEscaper.Replace(v)
	w.Write(dquote)
	io.WriteString(w, v)
	_, err := w.Write(dquote)
	return err
}

var (
	equals      = []byte("=")
	equalsQuote = []byte(`="`)
	dquote      = []byte(`"`)
	squote      = []byte(`'`)
	space       = []byte(` `)
)

var doubleQuoteEscaper = strings.NewReplacer(
	`&`, "&amp;",
	`"`, "&#34;",
)

const empty = `empty`

func isSameString(a, b string) bool {
	pa := reflect.ValueOf(a).UnsafePointer()
	pb := reflect.ValueOf(b).UnsafePointer()
	return pa == pb && len(a) == len(b)
}
