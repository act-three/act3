package xstrings

import "testing"

func TestToSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// basic cases
		{"empty", "", ""},
		{"simple lowercase", "hello", "hello"},
		{"simple words", "hello world", "hello-world"},

		// case folding
		{"uppercase", "HELLO", "hello"},
		{"mixed case", "Hello World", "hello-world"},

		// whitespace
		{"multiple spaces", "hello   world", "hello-world"},
		{"tabs", "hello\tworld", "hello-world"},
		{"leading/trailing spaces", "  hello  ", "hello"},
		{"newlines", "hello\nworld", "hello-world"},
		{"mixed whitespace", " \t hello \n world \t ", "hello-world"},

		// hyphens
		{"existing hyphens", "hello-world", "hello-world"},
		{"multiple hyphens", "hello---world", "hello-world"},
		{"leading hyphens", "---hello", "hello"},
		{"trailing hyphens", "hello---", "hello"},
		{"only hyphens", "---", ""},

		// punctuation removal
		{"apostrophe", "it's", "its"},
		{"comma", "hello, world", "hello-world"},
		{"period", "hello.world", "helloworld"},
		{"colon", "part: one", "part-one"},
		{"parentheses", "hello (world)", "hello-world"},
		{"exclamation", "hello!", "hello"},
		{"question mark", "hello?", "hello"},

		// symbols become hyphens
		{"ampersand", "rock & roll", "rock-roll"},
		{"plus", "a+b", "a-b"},
		{"equals", "a=b", "a-b"},
		{"dollar", "price $10", "price-10"},

		// numbers
		{"numbers", "season 1", "season-1"},
		{"leading number", "24 hours", "24-hours"},
		{"only numbers", "12345", "12345"},
		{"mixed alphanumeric", "s01e02", "s01e02"},

		// unicode
		{"accented chars NFKC", "caf\u00e9", "caf\u00e9"},             // café
		{"umlaut", "\u00fcber", "\u00fcber"},                          // über
		{"fullwidth NFKC", "\uff28\uff45\uff4c\uff4c\uff4f", "hello"}, // Ｈｅｌｌｏ → hello
		{"composed e-acute", "caf\u0065\u0301", "caf\u00e9"},          // e + combining acute → é (NFKC)
		{"cjk characters", "hello\u4e16\u754c", "hello\u4e16\u754c"},

		// realistic media titles
		{"movie title", "The Dark Knight", "the-dark-knight"},
		{"series with year", "Doctor Who (2005)", "doctor-who-2005"},
		{"title with colon", "Star Trek: Discovery", "star-trek-discovery"},
		{"title with ampersand", "Law & Order", "law-order"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSlug(tt.input)
			if got != tt.want {
				t.Logf("hex input %x", tt.input)
				t.Logf("hex got   %x", got)
				t.Logf("hex want  %x", tt.want)
				t.Errorf("ToSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
