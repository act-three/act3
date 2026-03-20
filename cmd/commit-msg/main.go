// Command commit-msg is a git commit-msg hook that validates
// the commit subject prefix conventions.
//
// Usage: commit-msg <message-file>
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: commit-msg <message-file>")
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	subject, _, _ := strings.Cut(string(data), "\n")
	subject = strings.TrimSpace(subject)

	changed, err := stagedFiles()
	if err != nil {
		return err
	}

	return check(subject, changed)
}

// check validates a commit subject line.
// changed is the list of staged file paths (may be nil).
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

func stagedFiles() ([]string, error) {
	// Diff against HEAD^ to capture all files in the resulting commit.
	// This works for both normal commits (HEAD^ is the previous commit)
	// and --amend (HEAD^ is the parent of the commit being replaced).
	// Fall back to the empty tree for initial commits.
	base := "HEAD^"
	if exec.Command("git", "rev-parse", "--verify", "-q", "HEAD^").Run() != nil {
		base = "4b825dc642cb6eb9a060e54bf899d69f82cf7258" // empty tree
	}
	out, err := exec.Command("git", "diff", "--cached", "--name-only", base).Output()
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, "\n"), nil
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
