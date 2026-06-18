// Command frozencheck verifies that database/frozen.txt is append-only:
// the previous revision of the file must be a line-prefix of the
// current one. Already-frozen schema updates may never change or be
// removed; new entries may only be appended.
//
// Usage:
//
//	git show <base>:database/frozen.txt | frozencheck database/frozen.txt
//
// The previous revision is read from stdin and the current file is
// named by the sole argument. Intended for CI, where <base> is the
// merge base with the target branch.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: frozencheck <file>  (previous revision on stdin)")
		os.Exit(2)
	}
	old, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading stdin:", err)
		os.Exit(2)
	}
	cur, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading file:", err)
		os.Exit(2)
	}
	if err := check(old, cur); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// check reports whether cur preserves old as a line-prefix: every line
// of old must appear unchanged at the same position in cur, and cur may
// only add lines after them. The returned error names the first
// divergence.
func check(old, cur []byte) error {
	o, c := lines(old), lines(cur)
	if len(c) < len(o) {
		return fmt.Errorf("frozen.txt: %d frozen entries removed; entries are append-only", len(o)-len(c))
	}
	for i := range o {
		if o[i] != c[i] {
			return fmt.Errorf("frozen.txt:%d: frozen entry changed; entries are append-only\n  was: %s\n  now: %s",
				i+1, o[i], c[i])
		}
	}
	return nil
}

func lines(b []byte) []string {
	b = bytes.TrimRight(b, "\n")
	if len(b) == 0 {
		return nil
	}
	return strings.Split(string(b), "\n")
}
