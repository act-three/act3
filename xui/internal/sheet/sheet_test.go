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

func TestStyleSetCopies(t *testing.T) {
	var a StyleSet
	a.Set("gap", "8px")
	b := a
	b.Set("gap", "12px")
	var sh Sheet
	if ca, cb := sh.ClassFor(a), sh.ClassFor(b); ca == cb {
		t.Error("Set after copy affected the original")
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
