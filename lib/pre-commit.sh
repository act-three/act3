#!/bin/sh
# Source of truth: lib/pre-commit.sh
set -e

# Check that staged .go files are formatted.
unformatted=$(git diff --cached --name-only --diff-filter=ACM -- '*.go' | while read f; do
	if [ -n "$(gofmt -l "$f")" ]; then
		echo "$f"
	fi
done)
if [ -n "$unformatted" ]; then
	echo "gofmt: these files are not formatted:"
	echo "$unformatted"
	exit 1
fi

# Check that JS/CSS/JSON/SQL files are formatted.
dprint check

# Check for direct env var reads outside of package main.
go vet -vettool=.git/hooks/act3vet ./...

# Check that go.mod and go.sum are tidy.
cp go.mod go.mod.tmp
cp go.sum go.sum.tmp
go mod tidy
if ! diff -q go.mod go.mod.tmp >/dev/null 2>&1 || ! diff -q go.sum go.sum.tmp >/dev/null 2>&1; then
	rm go.mod.tmp go.sum.tmp
	echo "go mod tidy: go.mod or go.sum is not tidy, please stage the changes"
	git diff --stat -- go.mod go.sum
	exit 1
fi
rm go.mod.tmp go.sum.tmp

# Check that generated files are up to date.
before=$(git status --porcelain)
go generate ./...
after=$(git status --porcelain)
if [ "$before" != "$after" ]; then
	echo "go generate: generated files are out of date, please stage them"
	git status --short
	exit 1
fi
