#!/bin/bash
set -eo pipefail

# Run from project root by go generate.
# See main.go.

domi=$(go list -m -f '{{.Dir}}' ily.dev/domi)/client.js

go tool esbuild --alias:domi="$domi" \
	--bundle --outfile=web/static/static/bundle.js main.js
