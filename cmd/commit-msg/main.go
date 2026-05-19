// Command commit-msg validates a commit subject's prefix conventions
// (e.g. "model:" or "video/ffmpeg:" before the colon).
//
// Usage: commit-msg <subject>
//
// The list of files touched by the change is read from stdin, one
// path per line. Used in CI to lint a PR title against the files the
// PR touches.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: commit-msg <subject>")
		os.Exit(2)
	}
	subject := os.Args[1]

	var changed []string
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		f := strings.TrimSpace(sc.Text())
		if f != "" {
			changed = append(changed, f)
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading stdin:", err)
		os.Exit(2)
	}

	if err := check(subject, changed); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// check validates a commit subject line.
// changed is the list of files modified in the commit (may be nil).
func check(subject string, changed []string) error {
	if len(subject) > 55 {
		return fmt.Errorf("commit-msg: subject must be at most 55 characters (got %d)\n  subject: %s", len(subject), subject)
	}
	if strings.HasSuffix(subject, ".") {
		return fmt.Errorf("commit-msg: subject must not end with a period\n  subject: %s", subject)
	}

	// Extract prefix (everything before the first colon).
	prefixPart, _, ok := strings.Cut(subject, ":")
	if !ok {
		return fmt.Errorf("commit-msg: subject must start with a prefix followed by a colon\n  subject: %s", subject)
	}

	// Split comma-separated prefixes and validate each one.
	prefixes := strings.Split(prefixPart, ",")
	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "all" {
			continue
		}
		if strings.HasPrefix(p, ".") {
			return fmt.Errorf("commit-msg: prefix %q must not start with a dot (omit the leading dot)", p)
		}
		if !exists(p) && !exists("."+p) {
			return fmt.Errorf("commit-msg: prefix %q is not a file or directory\n  subject: %s", p, subject)
		}
	}

	// Specificity check: single prefix must be as specific as possible.
	if len(prefixes) > 1 || strings.TrimSpace(prefixes[0]) == "all" {
		return nil
	}
	prefix := strings.TrimSpace(prefixes[0])

	if len(changed) == 0 {
		return nil
	}

	common := commonDir(changed)
	if common == "." {
		return nil
	}

	// Resolve the prefix to its real path for comparison.
	realPrefix := prefix
	if !exists(realPrefix) && exists("."+realPrefix) {
		realPrefix = "." + realPrefix
	}

	if realPrefix != common && strings.HasPrefix(common, realPrefix+"/") {
		return fmt.Errorf("commit-msg: prefix %q is too broad; all changes are in %q\n  subject: %s", prefix, common, subject)
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func commonDir(files []string) string {
	common := filepath.Dir(files[0])
	for _, f := range files[1:] {
		dir := filepath.Dir(f)
		for common != "." {
			if dir == common || strings.HasPrefix(dir, common+"/") {
				break
			}
			common = filepath.Dir(common)
		}
	}
	return common
}
