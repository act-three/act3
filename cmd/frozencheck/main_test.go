package main

import "testing"

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		old     string
		cur     string
		wantErr bool
	}{
		{"identical", "a\nb\n", "a\nb\n", false},
		{"append one", "a\nb\n", "a\nb\nc\n", false},
		{"append many", "a\n", "a\nb\nc\n", false},
		{"empty old", "", "a\n", false},
		{"both empty", "", "", false},
		{"missing trailing newline", "a\nb", "a\nb\nc", false},
		{"changed last line", "a\nb\n", "a\nB\n", true},
		{"changed first line", "a\nb\n", "x\nb\nc\n", true},
		{"removed line", "a\nb\n", "a\n", true},
		{"removed all", "a\nb\n", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := check([]byte(tt.old), []byte(tt.cur))
			if (err != nil) != tt.wantErr {
				t.Errorf("check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
