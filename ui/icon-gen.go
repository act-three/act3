//go:build ignore

// Fetches Lucide icon SVGs from the lucide-static npm package
// and writes them to the icon/ directory.
//
// Usage:
//
//	go run icon-gen.go
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	registryURL = "https://registry.npmjs.org/lucide-static/latest"
	iconDir     = "icon"
	prefix      = "package/icons/"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	tarballURL, err := getTarballURL()
	if err != nil {
		return fmt.Errorf("get tarball url: %w", err)
	}
	log.Printf("fetching %s", tarballURL)

	resp, err := http.Get(tarballURL)
	if err != nil {
		return fmt.Errorf("fetch tarball: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("fetch tarball: %s", resp.Status)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return err
	}

	var n int
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		name, ok := strings.CutPrefix(hdr.Name, prefix)
		if !ok || !strings.HasSuffix(name, ".svg") {
			continue
		}
		// Skip subdirectories.
		if strings.Contains(name, "/") {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		data = cleanSVG(data)
		dst := filepath.Join(iconDir, name)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
		n++
	}
	log.Printf("wrote %d icons to %s/", n, iconDir)
	return nil
}

func getTarballURL() (string, error) {
	resp, err := http.Get(registryURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("%s", resp.Status)
	}
	var meta struct {
		Dist struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", err
	}
	if meta.Dist.Tarball == "" {
		return "", fmt.Errorf("no tarball in response")
	}
	return meta.Dist.Tarball, nil
}

var stripAttr = regexp.MustCompile(`\n\s*(?:` +
	`class="lucide[^"]*"|` +
	`xmlns="[^"]*"|` +
	`width="[^"]*"|` +
	`height="[^"]*"|` +
	`fill="[^"]*"|` +
	`stroke="[^"]*"|` +
	`stroke-width="[^"]*"|` +
	`stroke-linecap="[^"]*"|` +
	`stroke-linejoin="[^"]*"` +
	`)`)

// cleanSVG strips the license comment and presentational
// attributes (handled by icon.css), and adds class="u-icon".
func cleanSVG(data []byte) []byte {
	// Remove <!-- @license ... --> comment line.
	if i := bytes.Index(data, []byte("<!--")); i >= 0 {
		if j := bytes.Index(data[i:], []byte("-->\n")); j >= 0 {
			data = append(data[:i], data[i+j+4:]...)
		}
	}
	data = stripAttr.ReplaceAll(data, nil)
	data = bytes.Replace(data, []byte("<svg"), []byte(`<svg class="u-icon"`), 1)
	return data
}
