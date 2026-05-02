#!/bin/sh
# Source of truth: lib/pre-push.sh
set -e

# Save stdin (refs being pushed) for commit subject validation below.
refs=$(cat)

# Check that .go files are formatted.
before=$(git status --porcelain)
gofmt -w .
after=$(git status --porcelain)
if [ "$before" != "$after" ]; then
	echo "gofmt: these files are not formatted:"
	git status --short
	echo "please run 'jj fix' to fix"
	exit 1
fi

# Check that JS/CSS/JSON/SQL files are formatted.
before=$(git status --porcelain)
dprint fmt
after=$(git status --porcelain)
if [ "$before" != "$after" ]; then
	echo "dprint: these files are not formatted:"
	git status --short
	echo "please run 'jj fix' to fix"
	exit 1
fi

# Check for direct env var reads outside of package main.
go vet -vettool=.git/hooks/act3vet ./...

# Check that go.mod and go.sum are tidy.
cp go.mod go.mod.tmp
cp go.sum go.sum.tmp
go mod tidy
if ! diff -q go.mod go.mod.tmp >/dev/null 2>&1 || ! diff -q go.sum go.sum.tmp >/dev/null 2>&1; then
	mv go.mod.tmp go.mod
	mv go.sum.tmp go.sum
	echo "go mod tidy: go.mod or go.sum is not tidy"
	exit 1
fi
rm go.mod.tmp go.sum.tmp

# Check that generated files are up to date.
before=$(git status --porcelain)
go generate ./...
after=$(git status --porcelain)
if [ "$before" != "$after" ]; then
	echo "go generate: generated files are out of date"
	git status --short || true
	echo "please run './mk regen' to fix"
	exit 1
fi

# Validate commit subject prefixes for each commit being pushed.
# Stdin format (one line per ref): <local-ref> <local-sha> <remote-ref> <remote-sha>.
zero=0000000000000000000000000000000000000000
while IFS=' ' read -r local_ref local_sha remote_ref remote_sha; do
	[ -z "$local_sha" ] && continue
	[ "$local_sha" = "$zero" ] && continue # branch deletion
	if [ "$remote_sha" = "$zero" ]; then
		commits=$(git rev-list "$local_sha" --not --remotes)
	else
		commits=$(git rev-list "$remote_sha..$local_sha")
	fi
	if [ -n "$commits" ]; then
		.git/hooks/commit-msg-check $commits
	fi
done <<EOF
$refs
EOF
