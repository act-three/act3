package html

import (
	"fmt"
	"html"
	"io"

	"ily.dev/act3/html/internal/env"
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

func (r raw) renderTo(w io.Writer, env env.Env) error {
	_, err := io.WriteString(w, string(r))
	return err
}

// With modifies e with m.
func (r raw) With(m Modifier) Node { return r }

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

func (t text) renderTo(w io.Writer, env env.Env) error {
	s := html.EscapeString(string(t))
	_, err := io.WriteString(w, s)
	return err
}

// With modifies e with m.
func (t text) With(m Modifier) Node { return t }
