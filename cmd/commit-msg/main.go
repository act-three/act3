// Command commit-msg validates commit subject prefix conventions
// for one or more commits.
//
// It is intended to be invoked from a pre-push hook, with the SHAs of
// the commits being pushed passed as arguments.
//
// Usage: commit-msg <sha>...
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: commit-msg <sha>...")
		os.Exit(2)
	}
	fail := false
	for _, sha := range os.Args[1:] {
		if err := checkCommit(sha); err != nil {
			fmt.Fprintln(os.Stderr, err)
			fail = true
		}
	}
	if fail {
		os.Exit(1)
	}
}

func checkCommit(sha string) error {
	subject, err := commitSubject(sha)
	if err != nil {
		return err
	}
	files, err := commitFiles(sha)
	if err != nil {
		return err
	}
	if err := check(subject, files); err != nil {
		return fmt.Errorf("%s: %w", short(sha), err)
	}
	return nil
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

func commitSubject(sha string) (string, error) {
	out, err := exec.Command("git", "log", "-1", "--format=%s", sha).Output()
	if err != nil {
		return "", fmt.Errorf("git log %s: %w", short(sha), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func commitFiles(sha string) ([]string, error) {
	out, err := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", sha).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff-tree %s: %w", short(sha), err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, "\n"), nil
}

// check validates a commit subject line.
// changed is the list of files modified in the commit (may be nil).
func check(subject string, changed []string) error {
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
