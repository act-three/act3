package ui

import (
	"strings"
	"testing"

	"ily.dev/act3/html"
)

func renderNode(n html.Node) string {
	var buf strings.Builder
	html.Render(&buf, n)
	return buf.String()
}

func TestLiveText(t *testing.T) {
	tests := []struct {
		name string
		text string
		addr []string
		want string
	}{
		{
			name: "no addr",
			text: "hello",
			want: `<span data-live>hello</span>`,
		},
		{
			name: "single addr",
			text: "hello",
			addr: []string{"users"},
			want: `<span data-live data-addr0=users>hello</span>`,
		},
		{
			name: "multiple addr",
			text: "me@example.org",
			addr: []string{"users", "345", "email"},
			want: `<span data-live data-addr0=users data-addr1=345 data-addr2=email>me@example.org</span>`,
		},
		{
			name: "addr with spaces",
			text: "hello",
			addr: []string{"my table"},
			want: `<span data-live data-addr0="my table">hello</span>`,
		},
		{
			name: "empty text",
			text: "",
			addr: []string{"t", "1", "f"},
			want: `<span data-live data-addr0=t data-addr1=1 data-addr2=f></span>`,
		},
		{
			name: "html in text is escaped",
			text: "<b>bold</b>",
			addr: []string{"t"},
			want: `<span data-live data-addr0=t>&lt;b&gt;bold&lt;/b&gt;</span>`,
		},
		{
			name: "addr with double quotes",
			text: "x",
			addr: []string{`say "hi"`},
			want: `<span data-live data-addr0='say "hi"'>x</span>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderNode(LiveText(tt.text, tt.addr))
			if got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestLiveTextUpdate(t *testing.T) {
	tests := []struct {
		name string
		text string
		addr []string
	}{
		{
			name: "simple",
			text: "new value",
			addr: []string{"users", "345", "email"},
		},
		{
			name: "addr with double quotes",
			text: "val",
			addr: []string{`say "hi"`},
		},
		{
			name: "addr with backslash",
			text: "val",
			addr: []string{`path\to`},
		},
		{
			name: "addr with newline",
			text: "val",
			addr: []string{"line\none"},
		},
		{
			name: "addr with single and double quotes",
			text: "val",
			addr: []string{`it's "complicated"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderNode(LiveTextUpdate(tt.text, tt.addr))
			if !strings.Contains(got, "<turbo-stream") {
				t.Errorf("expected turbo-stream element, got: %s", got)
			}
			inner := renderNode(LiveText(tt.text, tt.addr))
			if !strings.Contains(got, inner) {
				t.Errorf("expected inner LiveText\ngot:  %s\nwant to contain: %s", got, inner)
			}
		})
	}
}

func TestCSSEscape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "double quote",
			input: `say "hi"`,
			want:  `say \"hi\"`,
		},
		{
			name:  "backslash",
			input: `path\to`,
			want:  `path\\to`,
		},
		{
			name:  "newline",
			input: "line\none",
			want:  `line\a one`,
		},
		{
			name:  "tab",
			input: "col\tone",
			want:  `col\9 one`,
		},
		{
			name:  "null byte",
			input: "a\x00b",
			want:  `a\0 b`,
		},
		{
			name:  "carriage return",
			input: "a\rb",
			want:  `a\d b`,
		},
		{
			name:  "unicode is preserved",
			input: "café",
			want:  "café",
		},
		{
			name:  "backslash and quote together",
			input: `\"`,
			want:  `\\\"`,
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "DEL character",
			input: "a\x7fb",
			want:  `a\7f b`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cssEscape(tt.input)
			if got != tt.want {
				t.Errorf("cssEscape(%q)\ngot:  %q\nwant: %q", tt.input, got, tt.want)
			}
		})
	}
}
