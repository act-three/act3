//go:build ignore

// Bundles the CSS from main.css into web/static/static/bundle.css
// using esbuild. Inlines icon SVG url() references as data URIs,
// falling back to the missing icon if the referenced file doesn't
// exist.
//
// Usage (called by go generate in main.go):
//
//	go run web/static/gen.go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

const (
	cssEntry   = "main.css"
	cssOut     = "web/static/static/bundle.css"
	iconPrefix = "icon/"
	iconDir    = "ui/icon/svg"
)

// missingSVG is the fallback icon (Lucide square-dashed, MIT),
// with inline presentational attrs for standalone use.
const missingSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"` +
	` fill="none" stroke="black" stroke-width="1.5"` +
	` stroke-linecap="round" stroke-linejoin="round">` +
	`<path d="M5 3a2 2 0 0 0-2 2"/>` +
	`<path d="M19 3a2 2 0 0 1 2 2"/>` +
	`<path d="M21 19a2 2 0 0 1-2 2"/>` +
	`<path d="M5 21a2 2 0 0 1-2-2"/>` +
	`<path d="M9 3h1"/>` +
	`<path d="M9 21h1"/>` +
	`<path d="M14 3h1"/>` +
	`<path d="M14 21h1"/>` +
	`<path d="M3 9v1"/>` +
	`<path d="M21 9v1"/>` +
	`<path d="M3 14v1"/>` +
	`<path d="M21 14v1"/>` +
	`</svg>`

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{cssEntry},
		Outfile:     cssOut,
		Bundle:      true,
		Write:       true,
		LogLevel:    api.LogLevelWarning,
		Plugins:     []api.Plugin{iconPlugin()},
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("esbuild: %s", result.Errors[0].Text)
	}
	return nil
}

// iconPlugin inlines icon SVG url() references as data URIs.
// CSS references like url("icon/line/check.svg") are resolved
// against ui/icon/svg/. If the file is missing, the missing
// icon SVG is inlined instead.
func iconPlugin() api.Plugin {
	return api.Plugin{
		Name: "icon-dataurl",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(
				api.OnResolveOptions{Filter: `\.svg$`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					if args.Kind != api.ResolveCSSURLToken {
						return api.OnResolveResult{}, nil
					}
					path := args.Path
					if !strings.HasPrefix(path, iconPrefix) {
						return api.OnResolveResult{}, nil
					}
					// Map icon/line/foo.svg → ui/icon/svg/line/foo.svg
					name := strings.TrimPrefix(path, iconPrefix)
					abs, err := filepath.Abs(
						filepath.Join(iconDir, name))
					if err != nil {
						return api.OnResolveResult{}, err
					}
					return api.OnResolveResult{
						Path:      abs,
						Namespace: "icon",
					}, nil
				},
			)

			build.OnLoad(
				api.OnLoadOptions{
					Filter:    `\.svg$`,
					Namespace: "icon",
				},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					data, err := os.ReadFile(args.Path)
					if err != nil {
						s := missingSVG
						return api.OnLoadResult{
							Contents: &s,
							Loader:   api.LoaderDataURL,
						}, nil
					}
					s := standaloneSVG(string(data))
					return api.OnLoadResult{
						Contents: &s,
						Loader:   api.LoaderDataURL,
					}, nil
				},
			)
		},
	}
}

// standaloneSVG makes a generated icon SVG self-contained for
// use in a data URI (e.g. mask-image) where CSS classes don't
// apply. Adds xmlns and inline presentational attributes based
// on the icon's class.
func standaloneSVG(s string) string {
	xmlns := ` xmlns="http://www.w3.org/2000/svg"`
	switch {
	case strings.Contains(s, `class="u-icon"`):
		s = strings.Replace(s, `class="u-icon"`,
			xmlns+` fill="none" stroke="black"`+
				` stroke-width="1.5"`+
				` stroke-linecap="round"`+
				` stroke-linejoin="round"`, 1)
	case strings.Contains(s, `class="u-icon-solid"`):
		s = strings.Replace(s, `class="u-icon-solid"`,
			xmlns+` fill="black"`, 1)
	}
	return s
}
