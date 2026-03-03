package attr

import (
	"strings"
	"testing"
)

func renderGroup(attrs ...Node) string {
	var buf strings.Builder
	Group(attrs...).renderTo(&buf)
	return buf.String()
}

func TestGroupCombining(t *testing.T) {
	tests := []struct {
		name     string
		attrs    []Node
		want     string   // exact match; empty to skip
		contains []string // substring checks; used when map order is nondeterministic
	}{
		{
			name: "empty",
			want: "",
		},
		{
			name:  "single class",
			attrs: []Node{Class("a")},
			want:  ` class="a"`,
		},
		{
			name:  "two classes combined with space",
			attrs: []Node{Class("a"), Class("b")},
			want:  ` class="a b"`,
		},
		{
			name:  "three classes combined",
			attrs: []Node{Class("a"), Class("b"), Class("c")},
			want:  ` class="a b c"`,
		},
		{
			name:  "two styles combined with semicolon",
			attrs: []Node{Style("color:red"), Style("font-size:12px")},
			want:  ` style="color:red;font-size:12px"`,
		},
		{
			name:     "class and style together",
			attrs:    []Node{Class("a"), Style("color:red"), Class("b")},
			contains: []string{`class="a b"`, `style="color:red"`},
		},
		{
			name:  "non-combining first wins",
			attrs: []Node{ID("first"), ID("second")},
			want:  ` id=first`,
		},
		{
			name:     "non-combining and combining",
			attrs:    []Node{ID("x"), Class("a"), Class("b")},
			contains: []string{`id=x`, `class="a b"`},
		},
		{
			name:  "nested groups flatten",
			attrs: []Node{Group(Class("a"), Class("b")), Class("c")},
			want:  ` class="a b c"`,
		},
		{
			name:  "nils ignored",
			attrs: []Node{nil, Class("a"), nil, Class("b")},
			want:  ` class="a b"`,
		},
		{
			name:  "combining empty attr skipped",
			attrs: []Node{Class, Class("a")},
			want:  ` class="a"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderGroup(tt.attrs...)
			if tt.want != "" {
				if got != tt.want {
					t.Errorf("got %q, want %q", got, tt.want)
				}
			} else if tt.contains != nil {
				for _, s := range tt.contains {
					if !strings.Contains(got, s) {
						t.Errorf("got %q, missing %q", got, s)
					}
				}
			} else if got != "" {
				t.Errorf("got %q, want empty", got)
			}
		})
	}
}

func TestRegisterCombining(t *testing.T) {
	RegisterCombining("data-test", ",")
	defer delete(combining, "data-test")

	a := Attr("data-test")
	got := renderGroup(a("x"), a("y"), a("z"))
	want := ` data-test="x,y,z"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGroupHas(t *testing.T) {
	g := Group(Class("a"), ID("x"))
	tests := []struct {
		name string
		want bool
	}{
		{"class", true},
		{"id", true},
		{"href", false},
	}
	for _, tt := range tests {
		if got := g.Has(tt.name); got != tt.want {
			t.Errorf("Has(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
