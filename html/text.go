package html

import (
	"fmt"
	"html"
	"io"
)

type raw string

// Raw renders its contents directly to its output,
// with no further processing.
// This should be used only with trustworthy data,
// since any HTML special characters
// will be interpreted by the browser.
func Raw(s string) Node {
	return raw(s)
}

func (r raw) renderTo(w io.Writer) error {
	_, err := io.WriteString(w, string(r))
	return err
}

type text string

// Text escapes HTML special characters in its contents
// before rendering the result.
func Text(s string) Node {
	return text(s)
}

// Textf renders formatted text to its output.
// It escapes the formatted text as in [Text].
func Textf(format string, arg ...any) Node {
	return text(fmt.Sprintf(format, arg...))
}

func (t text) renderTo(w io.Writer) error {
	s := html.EscapeString(string(t))
	_, err := io.WriteString(w, s)
	return err
}
