package web

import "testing"

func TestIsEpisodeSlug(t *testing.T) {
	tests := []struct {
		slug string
		want bool
	}{
		// sNNeNN- pattern
		{"s01e01-pilot", true},
		{"s02e03-the-one-where", true},
		{"s99e99-x", true},

		// sNN- pattern
		{"s01-special", true},
		{"s02-bonus", true},

		// empty / too short
		{"", false},
		{"s", false},
		{"s0", false},
		{"s01", false},
		{"s01e", false},
		{"s01e0", false},
		{"s01e01", false},

		// missing trailing dash
		{"s01x", false},
		{"s01e01x", false},

		// not starting with s + digits
		{"e01e01-pilot", false},
		{"sxxe01-pilot", false},
		{"sx1e01-pilot", false},
		{"s1xe01-pilot", false},

		// edition-like slugs that must not match
		{"directors-cut", false},
		{"special-edition", false},
		{"season-1", false},
	}
	for _, tt := range tests {
		if got := isEpisodeSlug(tt.slug); got != tt.want {
			t.Errorf("isEpisodeSlug(%q) = %v, want %v", tt.slug, got, tt.want)
		}
	}
}
