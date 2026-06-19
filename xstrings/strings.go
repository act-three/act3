package xstrings

import (
	"cmp"
	"iter"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// LastCut slices s around the last instance of sep,
// returning the text before and after sep.
// The found result reports whether sep appears in s.
// If sep does not appear in s, LastCut returns "", s, false.
func LastCut(s, sep string) (before, after string, found bool) {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return "", s, false
}

var hyphens = regexp.MustCompile(`-+`)

func mapSlug(r rune) rune {
	switch {
	case unicode.IsSpace(r), unicode.IsSymbol(r), r == '-':
		return '-'
	case unicode.IsLetter(r), unicode.IsNumber(r):
		return r
	}
	return -1
}

// ToSlug modifies s to be used in a URL path component or filename.
func ToSlug(s string) string {
	s = norm.NFKC.String(s)
	s = strings.ToLower(s)
	s = strings.Map(mapSlug, s)
	s = strings.Trim(s, "-")
	s = hyphens.ReplaceAllString(s, "-")
	return s
}

// SanitizeFilename removes characters that are unsafe in filenames
// across operating systems and normalizes Unicode to NFC.
func SanitizeFilename(s string) string {
	s = norm.NFC.String(s)
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7F || unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return -1
		}
		return r
	}, s)
	s = strings.TrimRight(s, ". ")
	return s
}

func LongestCommonPrefix(it iter.Seq[string]) string {
	s := slices.Collect(it)
	if len(s) == 0 {
		return ""
	}
	min := slices.Min(s)
	max := slices.Max(s)
	for i := range min {
		if i >= len(max) || min[i] != max[i] {
			return min[:i]
		}
	}
	return min
}

// CompareNatural compares a and b with a "natural" ordering in which
// maximal runs of ASCII digits are compared by numeric value rather
// than lexically, so that "file9" sorts before "file10".
// Non-digit runs compare as in [strings.Compare].
// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
func CompareNatural(a, b string) int {
	for a != "" && b != "" {
		var ac, bc string
		ac, a = chunk(a)
		bc, b = chunk(b)
		var c int
		if isDigit(ac[0]) && isDigit(bc[0]) {
			c = compareDigits(ac, bc)
		} else {
			c = strings.Compare(ac, bc)
		}
		if c != 0 {
			return c
		}
	}
	return cmp.Compare(len(a), len(b))
}

func isDigit(b byte) bool { return '0' <= b && b <= '9' }

// chunk splits off the leading run of s consisting entirely of ASCII
// digits or entirely of non-digits, returning it and the remainder.
func chunk(s string) (head, rest string) {
	num := isDigit(s[0])
	i := 1
	for i < len(s) && isDigit(s[i]) == num {
		i++
	}
	return s[:i], s[i:]
}

// compareDigits orders two runs of ASCII digits by numeric value,
// ignoring leading zeros so that "08" and "8" compare equal.
func compareDigits(a, b string) int {
	a = strings.TrimLeft(a, "0")
	b = strings.TrimLeft(b, "0")
	if len(a) != len(b) {
		return cmp.Compare(len(a), len(b))
	}
	return strings.Compare(a, b)
}
