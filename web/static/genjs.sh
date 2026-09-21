#!/bin/bash
set -eo pipefail

# Run from project root by go generate.
# See main.go.

domi=$(go list -f '{{.Dir}}' ily.dev/domi)/domi.js
out=web/static/static/bundle.js

go tool esbuild --alias:domi="$domi" \
	--bundle --outfile="$out" main.js

# esbuild annotates each bundled source with its path relative to the
# working directory. domi resolves through the module cache, whose
# location varies by machine, so rewrite that one path to a stable
# label to keep the bundle reproducible.
sed -E 's#// [^ ]*ily\.dev/domi@[^/]*/domi\.js#// domi/domi.js#' \
	"$out" >"$out.tmp"
mv "$out.tmp" "$out"
