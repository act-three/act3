package model

import (
	"fmt"
	"iter"

	"ily.dev/act3/xstrings"
)

// editionSlugCandidates yields candidates to try when choosing the slug
// for an edition with the given title.
// First "" (which indicates the default edition for a work),
// then ToSlug(title), then appending "-2", "-3", etc, indefinitely.
// If ToSlug(title) is empty, it uses "edition".
func editionSlugCandidates(title string) iter.Seq[string] {
	return func(yield func(string) bool) {
		if !yield("") {
			return
		}
		base := xstrings.ToSlug(title)
		if base == "" {
			base = "edition"
		}
		if !yield(base) {
			return
		}
		for i := 2; ; i++ {
			if !yield(fmt.Sprintf("%s-%d", base, i)) {
				return
			}
		}
	}
}
