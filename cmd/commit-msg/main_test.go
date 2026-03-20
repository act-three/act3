package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// All tests run against the testdata directory tree.
	if err := os.Chdir("testdata"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		changed []string // staged files
		wantErr string   // substring of error, empty means no error
	}{
		{
			name:    "valid prefix",
			subject: "model: add new field",
		},
		{
			name:    "missing colon",
			subject: "no colon here",
			wantErr: "must start with a prefix followed by a colon",
		},
		{
			name:    "nonexistent prefix",
			subject: "fakedir: something",
			wantErr: `"fakedir" is not a file or directory`,
		},
		{
			name:    "all prefix",
			subject: "all: big refactor",
		},
		{
			name:    "comma-separated valid",
			subject: "model, view: update types",
		},
		{
			name:    "comma-separated one invalid",
			subject: "model, fakedir: update types",
			wantErr: `"fakedir" is not a file or directory`,
		},
		{
			name:    "dot prefix rejected",
			subject: ".claude: update config",
			wantErr: "must not start with a dot",
		},
		{
			name:    "dotless prefix maps to dot dir",
			subject: "claude: update config",
		},
		{
			name:    "merge commit rejected",
			subject: "Merge branch 'feature' into main",
			wantErr: "must start with a prefix followed by a colon",
		},
		{
			name:    "nested prefix",
			subject: "video/ffmpeg: fix encoding",
		},
		{
			name:    "specificity ok",
			subject: "video/ffmpeg: fix encoding",
			changed: []string{"video/ffmpeg/encode.go", "video/ffmpeg/decode.go"},
		},
		{
			name:    "too broad",
			subject: "video: fix encoding",
			changed: []string{"video/ffmpeg/encode.go", "video/ffmpeg/decode.go"},
			wantErr: `"video" is too broad`,
		},
		{
			name:    "specificity skipped for multi-prefix",
			subject: "video, model: refactor",
			changed: []string{"video/ffmpeg/encode.go"},
		},
		{
			name:    "specificity skipped for all",
			subject: "all: refactor",
			changed: []string{"video/ffmpeg/encode.go"},
		},
		{
			name:    "specificity skipped when files span root",
			subject: "model: cross-cutting change",
			changed: []string{"model/movie.go", "view/home.go"},
		},
		{
			name:    "no staged files",
			subject: "model: allow-empty commit",
			changed: nil,
		},
		{
			name:    "prefix matches common dir exactly",
			subject: "video: touches multiple subdirs",
			changed: []string{"video/ffmpeg/encode.go", "video/hls/playlist.go"},
		},
		{
			name:    "root file as prefix",
			subject: "main.txt: fix startup",
			changed: []string{"main.txt"},
		},
		{
			name:    "dir prefix ok for single file in subdir",
			subject: "model: fix one file",
			changed: []string{"model/movie.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := check(tt.subject, tt.changed)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Fatalf("error %q does not contain %q", got, tt.wantErr)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestCommonDir(t *testing.T) {
	tests := []struct {
		files []string
		want  string
	}{
		{[]string{"a/b/c.go"}, "a/b"},
		{[]string{"a/b/c.go", "a/b/d.go"}, "a/b"},
		{[]string{"a/b/c.go", "a/d/e.go"}, "a"},
		{[]string{"a/b/c.go", "x/y/z.go"}, "."},
		{[]string{"a.go"}, "."},
		{[]string{"a/b/c.go", "a/b/d/e.go"}, "a/b"},
	}

	for _, tt := range tests {
		got := commonDir(tt.files)
		if got != tt.want {
			t.Errorf("commonDir(%v) = %q, want %q", tt.files, got, tt.want)
		}
	}
}
