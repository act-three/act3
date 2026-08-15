package sheet

import (
	"regexp"
	"strings"
	"testing"
)

var classShape = regexp.MustCompile(`^ui-[0-9a-v]{1,8}$`)

func TestClassInterning(t *testing.T) {
	var sh Sheet

	a := sh.ClassFor(Style("gap", "12px"))
	if !classShape.MatchString(a) {
		t.Errorf("class %q does not match %v", a, classShape)
	}
	if got := sh.ClassFor(Style("gap", "12px")); got != a {
		t.Errorf("same set interned twice: %q then %q", a, got)
	}
	if got := sh.ClassFor(Style("gap", "16px")); got == a {
		t.Errorf("distinct sets share class %q", got)
	}
	if n := strings.Count(sh.CSS(), "\n") + 1; n != 2 {
		t.Errorf("rule count = %d, want 2:\n%s", n, sh.CSS())
	}
}

func TestClassCanonicalOrder(t *testing.T) {
	var sh Sheet
	var a, b StyleSet
	a.Set("width", "56px")
	a.Set("height", "84px")
	b.Set("height", "84px")
	b.Set("width", "56px")
	if ca, cb := sh.ClassFor(a), sh.ClassFor(b); ca != cb {
		t.Errorf("insertion order changed the class: %q vs %q", ca, cb)
	}
	if want := "{height:84px;width:56px}"; !strings.Contains(sh.CSS(), want) {
		t.Errorf("rule not in canonical order:\n%s", sh.CSS())
	}
}

func TestClassStableAcrossSheets(t *testing.T) {
	var sh1, sh2 Sheet
	if a, b := sh1.ClassFor(Style("color", "#fff")), sh2.ClassFor(Style("color", "#fff")); a != b {
		t.Errorf("same set named %q and %q in different sheets", a, b)
	}
}

func TestSetReplaces(t *testing.T) {
	var sh Sheet
	var s StyleSet
	s.Set("opacity", "0.5")
	s.Set("opacity", "0.9")
	sh.ClassFor(s)
	if css := sh.CSS(); !strings.Contains(css, "opacity:0.9") || strings.Contains(css, "0.5") {
		t.Errorf("later Set should win:\n%s", css)
	}
}

func TestSetPseudoNests(t *testing.T) {
	var sh Sheet
	var s StyleSet
	s.Set("position", "relative")
	s.SetPseudo("::after", "inset", "0")
	s.SetPseudo("::after", "content", `""`)
	s.SetPseudo(":hover", "color", "red")
	s.SetPseudo(":nth-child(2n)", "color", "blue")
	sh.ClassFor(s)
	want := `{position:relative;&::after{content:"";inset:0}&:hover{color:red}&:nth-child(2n){color:blue}}`
	if css := sh.CSS(); !strings.Contains(css, want) {
		t.Errorf("nested rule body not in canonical order:\n%s", css)
	}
}

func TestSetPseudoDistinguishes(t *testing.T) {
	var sh Sheet
	var a, b StyleSet
	a.Set("color", "red")
	b.SetPseudo(":hover", "color", "red")
	if ca, cb := sh.ClassFor(a), sh.ClassFor(b); ca == cb {
		t.Errorf("element and pseudo declarations share class %q", ca)
	}
}

func TestPseudoOnlySetHasClass(t *testing.T) {
	var sh Sheet
	var s StyleSet
	s.SetPseudo("::after", "content", `""`)
	if class := sh.ClassFor(s); class == "" {
		t.Error("pseudo-only StyleSet has no class")
	}
	if want := `{&::after{content:""}}`; !strings.Contains(sh.CSS(), want) {
		t.Errorf("pseudo-only rule body missing:\n%s", sh.CSS())
	}
}

func TestInvalidPseudosPanic(t *testing.T) {
	for _, pseudo := range []string{"", "after", "::after{", ":not(x)}", ":hover;x", ":hover\n"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("SetPseudo(%q, ...) did not panic", pseudo)
				}
			}()
			var s StyleSet
			s.SetPseudo(pseudo, "color", "red")
		}()
	}
}

func TestStyleSetCopies(t *testing.T) {
	var a StyleSet
	a.Set("gap", "8px")
	b := a
	b.Set("gap", "12px")
	var sh Sheet
	if ca, cb := sh.ClassFor(a), sh.ClassFor(b); ca == cb {
		t.Error("Set after copy affected the original")
	}

	var c StyleSet
	c.SetPseudo("::after", "inset", "0")
	d := c
	d.SetPseudo("::after", "inset", "2px")
	if cc, cd := sh.ClassFor(c), sh.ClassFor(d); cc == cd {
		t.Error("SetPseudo after copy affected the original")
	}
}

func TestCSSAppendOnly(t *testing.T) {
	var sh Sheet
	sh.ClassFor(Style("gap", "8px"))
	before := sh.CSS()
	sh.ClassFor(Style("gap", "8px")) // revisit: no growth
	if sh.CSS() != before {
		t.Errorf("revisit changed the CSS:\n%s", sh.CSS())
	}
	sh.ClassFor(Style("gap", "16px"))
	if !strings.HasPrefix(sh.CSS(), before) {
		t.Errorf("CSS did not grow by suffix:\nbefore:\n%s\nafter:\n%s", before, sh.CSS())
	}
}

func TestInvalidDeclarationsPanic(t *testing.T) {
	for _, tt := range []struct{ property, value string }{
		{"", "x"},
		{"gap;color", "red"},
		{"gap", "8px}"},
		{"gap", "8px;color:red"},
		{"gap", "8px\n"},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Set(%q, %q) did not panic", tt.property, tt.value)
				}
			}()
			var s StyleSet
			s.Set(tt.property, tt.value)
		}()
	}
}

func TestEmptySetHasNoClass(t *testing.T) {
	var sh Sheet
	if got := sh.ClassFor(StyleSet{}); got != "" {
		t.Errorf("ClassFor of empty StyleSet = %q, want empty", got)
	}
	if css := sh.CSS(); css != "" {
		t.Errorf("empty StyleSet added a rule:\n%s", css)
	}
}
