//go:build ignore

// Vendors jassub (libass via WebAssembly) for the optional
// jassub-tagged build. Builds without -tags jassub don't need
// this — the player falls back to native WebVTT.
//
// Fetches jassub plus its npm dependencies from
// registry.npmjs.org with pinned SHA-512 integrity checks, then
// bundles the host module and worker via esbuild into self-
// contained ESM files in web/jassub/dist/. The wasm and
// LICENSE pass through as-is.
//
// Run from the project root (network required):
//
//	go run web/jassub/gen.go
//
// Idempotent: skips when dist/version.txt matches the pinned
// jassub version. Delete dist/ to force a re-fetch.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha3"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"

	"ily.dev/act3/encoding/base32c"
)

// digestStr returns the 6-char content digest used for hashed
// asset filenames. Matches the format used by http/digest.
func digestStr(data []byte) string {
	sum := sha3.Sum256(data)
	return strings.ToLower(base32c.EncodeToString(sum[:])[:6])
}

// pkg pins one npm package by version + SHA-512 integrity. The
// integrity string is the registry's dist.integrity field, encoded
// as "sha512-<base64>".
type pkg struct {
	name      string
	version   string
	integrity string
}

// packages lists every npm package needed to bundle jassub. The
// first entry must be jassub itself; the rest are jassub's flat
// dep tree (none of them have runtime deps of their own as of
// these versions).
var packages = []pkg{
	{"jassub", "2.5.1", "sha512-cQrQa42zfhNCBcNsYfep8E9acLW66PP87M3ZiNyWPEUyj4JBGhi7buf6aOhUbkNRkHe9iA+TNTZms0JJcS7zIA=="},
	{"abslink", "1.2.2", "sha512-HCOItvzORPjMzddCmEpbH8+W1VM3yjH+CR7DLtxlxFKX/jz5twJbm7KLfblO7eKLnXINLah8Hk2ZAJqxXeSkOg=="},
	{"lfa-ponyfill", "1.1.0", "sha512-YS3/DmyDdywWwoEu1ZacAudqkJ4q7WtKE9+bWlaSuEoVrXva7ChIJHMJYs19zyVc1H198pzqAreQU0r/+YNeew=="},
	{"rvfc-polyfill", "1.0.8", "sha512-uA+0wwTkZ4OT8v45pfDfH+7Yq8mY6MvNngiF5Sq6VBgjJsvsfgt7Q18cyZqZjfAhW9rhkgXPX0cW0R9uw7yElA=="},
	{"throughput", "1.0.2", "sha512-jvK1ZXuhsggjb3qYQjMiU/AVYYiTeqT5thWvYR2yuy2LGM84P5MSSyAinwHahGsdBYKR9m9HncVR/3f3nFKkxg=="},
}

const (
	outDir   = "web/jassub/dist"
	registry = "https://registry.npmjs.org"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	jassub := packages[0]

	if data, err := os.ReadFile(filepath.Join(outDir, "version.txt")); err == nil &&
		strings.TrimSpace(string(data)) == jassub.version {
		log.Printf("dist/ already at jassub %s — skipping (delete %s to refetch)", jassub.version, outDir)
		return nil
	}

	work, err := os.MkdirTemp("", "jassub-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	for _, p := range packages {
		dst := filepath.Join(work, "node_modules", p.name)
		if err := fetchAndExtract(p, dst); err != nil {
			return fmt.Errorf("%s@%s: %w", p.name, p.version, err)
		}
	}

	// Wipe everything in dist/ except the tracked placeholders
	// (Readme is what makes //go:embed dist succeed before gen.go
	// has ever run; .gitignore keeps every artifact gitignored
	// while still tracking Readme and itself).
	entries, err := os.ReadDir(outDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, e := range entries {
		if e.Name() == "Readme" || e.Name() == ".gitignore" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(outDir, e.Name())); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}
	jassubDir := filepath.Join(work, "node_modules", "jassub")

	// Bundle the emscripten WASM factory standalone first. This is
	// the file every em-pthread sub-worker loads — and it must not
	// carry the higher-level worker.js's abslink expose code, or
	// emscripten's pthread bootstrap competes with abslink for
	// `self.onmessage` and the parent's "loaded" handshake never
	// lands. (jassub.pages.dev's official demo splits these into
	// two files and it works there; bundling them combined hangs.)
	emBytes, err := bundleBytes(work,
		filepath.Join(jassubDir, "dist", "wasm", "jassub-worker.js"),
	)
	if err != nil {
		return fmt.Errorf("bundle emscripten: %w", err)
	}
	// Write under the bare basename so digest.Handler computes the
	// hash itself and serves at /-/jassub/jassub-emscripten.<digest>.js.
	// We pre-compute the same digest here so the worker bundle can
	// embed the URL string.
	if err := os.WriteFile(filepath.Join(absOut, "jassub-emscripten.js"), emBytes, 0o644); err != nil {
		return err
	}
	emName := "jassub-emscripten." + digestStr(emBytes) + ".js"

	// Bundle the higher-level worker (with abslink expose) and
	// patch the emscripten pthread spawn URL to point at the
	// standalone bundle written above. The literal
	// `"jassub-worker.js"` is baked into emscripten's output at
	// libass-wasm build time; substituting our digest URL routes
	// every em-pthread spawn at the cache-busted standalone file.
	workerBytes, err := bundleBytes(work,
		filepath.Join(jassubDir, "dist", "worker", "worker.js"),
	)
	if err != nil {
		return fmt.Errorf("bundle worker: %w", err)
	}
	patched := bytes.ReplaceAll(workerBytes,
		[]byte(`"jassub-worker.js"`),
		[]byte(`"./`+emName+`"`),
	)
	if bytes.Equal(patched, workerBytes) {
		return fmt.Errorf("worker bundle missing expected `\"jassub-worker.js\"` literal — emscripten output may have changed")
	}
	// Strip the trailing `export { ... as ASSRenderer }` statement.
	// esbuild emits it because worker.js's source declares `export
	// class ASSRenderer`, but the worker is loaded as a module
	// Worker — there's no consumer for the export, and (empirically)
	// keeping it deadlocks libass-wasm's runtime init in our setup
	// while jassub.pages.dev's Vite-bundled worker (which omits it)
	// works fine. Vite's Worker entry treatment drops dead exports
	// for the same reason; we replicate that here.
	// Replace with nothing (and ensure preceding char terminates).
	before, _, found := bytes.CutLast(patched, []byte("export{"))
	if !found {
		return fmt.Errorf("worker bundle missing trailing export statement")
	}
	patched = before
	if err := os.WriteFile(filepath.Join(absOut, "jassub-worker.js"), patched, 0o644); err != nil {
		return err
	}

	hostBytes, err := bundleBytes(work,
		filepath.Join(jassubDir, "dist", "jassub.js"),
	)
	if err != nil {
		return fmt.Errorf("bundle host: %w", err)
	}
	// Short-circuit JASSUB._test() so the host never calls
	// WebAssembly.Module() / .validate() / .Instance() on the
	// main thread. Those need 'wasm-unsafe-eval' in the document
	// CSP; the worker has its own CSP that allows it. Pre-setting
	// _supportsSIMD lets the static _test method return at its
	// first guard. We always ship the SIMD ("modern") wasm, so
	// the value is correct for our distribution.
	hostMarker := []byte(`t._test()`)
	hostReplace := []byte(`t._supportsSIMD=!0`)
	if !bytes.Contains(hostBytes, hostMarker) {
		return fmt.Errorf("host bundle missing expected `t._test()` call")
	}
	hostBytes = bytes.Replace(hostBytes, hostMarker, hostReplace, 1)
	if err := os.WriteFile(filepath.Join(absOut, "jassub.js"), hostBytes, 0o644); err != nil {
		return err
	}

	if err := copyFile(
		filepath.Join(jassubDir, "dist", "wasm", "jassub-worker-modern.wasm"),
		filepath.Join(outDir, "jassub-worker-modern.wasm"),
	); err != nil {
		return err
	}
	// jassub falls back on a bundled Liberation Sans woff2 when
	// the document doesn't supply other fonts; without it libass
	// can't open any font and renders nothing. Ship it.
	if err := copyFile(
		filepath.Join(jassubDir, "dist", "default.woff2"),
		filepath.Join(outDir, "default.woff2"),
	); err != nil {
		return err
	}
	if err := copyFile(
		filepath.Join(jassubDir, "LICENSE"),
		filepath.Join(outDir, "LICENSE"),
	); err != nil {
		return err
	}

	if err := os.WriteFile(
		filepath.Join(outDir, "version.txt"),
		[]byte(jassub.version+"\n"),
		0o644,
	); err != nil {
		return err
	}

	log.Printf("jassub %s vendored into %s", jassub.version, outDir)
	return nil
}

// fetchAndExtract downloads <pkg>-<ver>.tgz from npm, verifies the
// SHA-512 integrity, and extracts the contents into dst with the
// "package/" tarball prefix stripped.
func fetchAndExtract(p pkg, dst string) error {
	url := fmt.Sprintf("%s/%s/-/%s-%s.tgz", registry, p.name, p.name, p.version)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	sum := sha512.Sum512(body)
	got := "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
	if got != p.integrity {
		return fmt.Errorf("integrity mismatch:\n  want %s\n  got  %s", p.integrity, got)
	}

	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		name, ok := strings.CutPrefix(h.Name, "package/")
		if !ok || name == "" {
			continue
		}
		path := filepath.Join(dst, name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

// bundleBytes runs esbuild on entry and returns the bundled ESM
// module bytes. workDir is the AbsWorkingDir passed to esbuild so
// its node-style resolver finds the dep tree under
// workDir/node_modules. Returning bytes (rather than writing to
// disk) lets the caller patch the bundle before writing.
//
// TreeShakingFalse is required for the worker entry: the
// emscripten-generated jassub-worker.js auto-bootstraps em-pthread
// sub-workers via a top-level `isPthread && Module()` side effect,
// and esbuild's default tree-shaker DCEs that line because the
// importing worker.js only consumes the default export. Without
// the auto-call, em-pthread workers never set up their message
// handler and the parent's WASM init hangs forever waiting for a
// "loaded" reply. Disabling tree-shaking is also fine for the
// host bundle — neither file has dead code worth dropping.
func bundleBytes(workDir, entry string) ([]byte, error) {
	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{entry},
		Bundle:            true,
		Write:             false,
		Format:            api.FormatESModule,
		Platform:          api.PlatformBrowser,
		Target:            api.ESNext,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		TreeShaking:       api.TreeShakingFalse,
		AbsWorkingDir:     workDir,
		LogLevel:          api.LogLevelWarning,
	})
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("%s", result.Errors[0].Text)
	}
	if len(result.OutputFiles) != 1 {
		return nil, fmt.Errorf("expected 1 output file, got %d", len(result.OutputFiles))
	}
	return result.OutputFiles[0].Contents, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
