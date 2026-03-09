//go:build ignore

// Extracts SVGs from the Untitled UI Icons PRO zip file and
// writes them to svg/line/ and svg/solid/.
//
// Usage:
//
//	go run gen.go
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	zipFile = "untitled-icons.zip"
	outDir  = "svg"
)

// SHA-256 of the expected zip file.
const zipHash = "68e45b812e178ef70429b09035623529873ff9d51e1b74607e92c0285f677fbe"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	data, err := os.ReadFile(zipFile)
	if err != nil {
		return fmt.Errorf("read zip: %w", err)
	}

	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != zipHash {
		return fmt.Errorf("hash mismatch:\n  want %s\n  got  %s", zipHash, got)
	}

	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	// Clear output dirs.
	for _, sub := range []string{"line", "solid"} {
		dir := filepath.Join(outDir, sub)
		os.RemoveAll(dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	var nLine, nSolid int
	for _, f := range r.File {
		name := f.Name

		// Determine style from path.
		var style string
		switch {
		case strings.Contains(name, "Line icons/") ||
			strings.Contains(name, "Line Icons/"):
			style = "line"
		case strings.Contains(name, "Solid icons/") ||
			strings.Contains(name, "Solid Icons/"):
			style = "solid"
		default:
			continue
		}

		// Flatten: use only the base filename, skipping
		// macOS resource fork files and other junk.
		base := filepath.Base(name)
		if !validName.MatchString(base) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open %s: %w", name, err)
		}
		buf := &bytes.Buffer{}
		_, err = buf.ReadFrom(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		svg := cleanSVG(buf.Bytes(), style)
		dst := filepath.Join(outDir, style, base)
		if err := os.WriteFile(dst, svg, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}

		switch style {
		case "line":
			nLine++
		case "solid":
			nSolid++
		}
	}

	log.Printf("wrote %d line + %d solid = %d icons",
		nLine, nSolid, nLine+nSolid)
	return nil
}

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]*\.svg$`)

// stripAttr removes presentational attributes that CSS
// provides. Both line and solid icons are cleaned the same
// way — only the class name differs.
var stripAttr = regexp.MustCompile(
	`\s+(?:width|height|xmlns|fill|stroke|stroke-width|stroke-linecap|stroke-linejoin)="[^"]*"`)

var classForStyle = map[string]string{
	"line":  "u-icon",
	"solid": "u-icon-solid",
}

func cleanSVG(data []byte, style string) []byte {
	data = stripAttr.ReplaceAll(data, nil)
	data = bytes.Replace(data,
		[]byte("<svg"),
		[]byte(`<svg class="`+classForStyle[style]+`"`), 1)
	return data
}
