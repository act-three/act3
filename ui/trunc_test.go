package ui

import (
	"strings"
	"testing"

	"ily.dev/domi"
)

func renderNode(n domi.Node) string {
	var buf strings.Builder
	domi.RenderTo(&buf, n)
	return buf.String()
}

func TestTruncTail(t *testing.T) {
	// Short enough to have nothing to collapse: rendered as plain
	// text, with no wrapper span and no tooltip.
	for _, s := range []string{"", "Off", "Auto", "1080p", "exactly10!"} {
		got := renderNode(TruncTail(s))
		if strings.Contains(got, "u-trunc") {
			t.Errorf("TruncTail(%q) = %q, want plain text (no u-trunc)", s, got)
		}
		if got != s {
			t.Errorf("TruncTail(%q) = %q, want %q", s, got, s)
		}
	}
}

func TestTruncTailSplit(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantHead string
		wantTail string
	}{
		{
			name:     "ascii",
			in:       "AC-3 5.1 (Stereo)",
			wantHead: "AC-3 5.",
			wantTail: "1 (Stereo)",
		},
		{
			name:     "one rune over the limit",
			in:       "elevenchars",
			wantHead: "e",
			wantTail: "levenchars",
		},
		{
			name:     "multibyte split on rune boundary, not byte",
			in:       strings.Repeat("日", 12),
			wantHead: strings.Repeat("日", 2),
			wantTail: strings.Repeat("日", 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantHead+tt.wantTail != tt.in {
				t.Fatalf("test setup: head+tail %q != in %q", tt.wantHead+tt.wantTail, tt.in)
			}
			got := renderNode(TruncTail(tt.in))
			if !strings.Contains(got, ">"+tt.wantHead+"<") {
				t.Errorf("head %q not found in %q", tt.wantHead, got)
			}
			if !strings.Contains(got, ">"+tt.wantTail+"<") {
				t.Errorf("tail %q not found in %q", tt.wantTail, got)
			}
			// The full, untruncated string rides along as the tooltip.
			if !strings.Contains(got, tt.in) {
				t.Errorf("full title %q not found in %q", tt.in, got)
			}
		})
	}
}
