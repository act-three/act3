package xstrings

import (
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
