package main

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

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

func TestCheckReports(t *testing.T) {
	e000 := entry{"000", "d0", "000_meta.up.sql"}
	e001 := entry{"001", "d1", "001_init.up.sql"}
	e002 := entry{"002", "d2", "002_foo.up.sql"}
	e003 := entry{"003", "d3", "003_bar.up.sql"}

	frozen := func(es ...entry) string {
		var b strings.Builder
		for _, e := range es {
			fmt.Fprintf(&b, "%s %s %s\n", e.Version, e.Digest, e.Name)
		}
		return b.String()
	}
	report := func(up, base entry, total int, result string) string {
		return fmt.Sprintf(`{"update":{"version":%q,"digest":%q,"name":%q},`+
			`"base":{"version":%q,"digest":%q,"name":%q},`+
			`"rows":{"total":%d},"result":%q}`,
			up.Version, up.Digest, up.Name,
			base.Version, base.Digest, base.Name, total, result)
	}

	tests := []struct {
		name    string
		old     string
		cur     string
		files   map[string]string
		wantErr bool
	}{
		{"no new lines", frozen(e000, e001), frozen(e000, e001), nil, false},
		{"bootstrap exempt", "", frozen(e000, e001), nil, false},
		{
			"valid, base has no report (first real report)",
			frozen(e000, e001), frozen(e000, e001, e002),
			map[string]string{"002_foo.up.sql.json": report(e002, e001, 100, "ok")},
			false,
		},
		{"missing report", frozen(e000, e001), frozen(e000, e001, e002), nil, true},
		{
			"update mismatch",
			frozen(e000, e001), frozen(e000, e001, e002),
			map[string]string{"002_foo.up.sql.json": report(entry{"002", "WRONG", "002_foo.up.sql"}, e001, 100, "ok")},
			true,
		},
		{
			"base mismatch",
			frozen(e000, e001), frozen(e000, e001, e002),
			map[string]string{"002_foo.up.sql.json": report(e002, entry{"001", "WRONG", "001_init.up.sql"}, 100, "ok")},
			true,
		},
		{
			"result not ok",
			frozen(e000, e001), frozen(e000, e001, e002),
			map[string]string{"002_foo.up.sql.json": report(e002, e001, 100, "fail")},
			true,
		},
		{
			"floor satisfied",
			frozen(e000, e001, e002), frozen(e000, e001, e002, e003),
			map[string]string{
				"002_foo.up.sql.json": report(e002, e001, 100, "ok"),
				"003_bar.up.sql.json": report(e003, e002, 150, "ok"),
			},
			false,
		},
		{
			"floor violated",
			frozen(e000, e001, e002), frozen(e000, e001, e002, e003),
			map[string]string{
				"002_foo.up.sql.json": report(e002, e001, 100, "ok"),
				"003_bar.up.sql.json": report(e003, e002, 50, "ok"),
			},
			true,
		},
		{
			"two new lines: second's base frozen this change, so exempt",
			frozen(e000, e001), frozen(e000, e001, e002, e003),
			map[string]string{"002_foo.up.sql.json": report(e002, e001, 100, "ok")},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{}
			for name, body := range tt.files {
				fsys[name] = &fstest.MapFile{Data: []byte(body)}
			}
			err := checkReports([]byte(tt.old), []byte(tt.cur), fsys)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkReports() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
