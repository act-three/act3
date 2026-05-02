#!/bin/sh
# Source of truth: lib/pre-push.sh
set -e

# Save stdin (refs being pushed) for the loop below.
refs=$(cat)

# Capture absolute source paths up front — the per-commit checks below
# cd into a worktree, so repo-relative paths wouldn't resolve there.
src=$(git rev-parse --show-toplevel)
hooks_dir=$(cd "$(git rev-parse --git-path hooks)" && pwd)
act3vet=$hooks_dir/act3vet
commit_msg_check=$hooks_dir/commit-msg-check

# check_commit runs all checks against a single commit in a temporary
# worktree, so the user's working tree is never disturbed. Gitignored
# files (e.g. ui/icon/svg/* used by 'go generate') are copied in from
# the source tree, since 'git worktree add' only checks out tracked
# files and several checks need those untracked inputs to behave
# identically to running in the source tree.
check_commit() {
	commit=$1
	worktree=$(mktemp -d -t pre-push-XXXXXX)
	rmdir "$worktree" # git worktree add wants a fresh path
	(
		trap "git worktree remove --force '$worktree' >/dev/null 2>&1; rm -rf '$worktree'" EXIT
		git worktree add --detach -q "$worktree" "$commit"
		git -C "$src" ls-files -z --others --ignored --exclude-standard \
			| rsync -a --from0 --files-from=- "$src/" "$worktree/"
		cd "$worktree"

		# Check that .go files are formatted.
		before=$(git status --porcelain)
		gofmt -w .
		after=$(git status --porcelain)
		if [ "$before" != "$after" ]; then
			echo "$commit:"
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
			echo "$commit:"
			echo "dprint: these files are not formatted:"
			git status --short
			echo "please run 'jj fix' to fix"
			exit 1
		fi

		# Check for direct env var reads outside of package main.
		go vet -vettool="$act3vet" ./...

		# Check that go.mod and go.sum are tidy.
		cp go.mod go.mod.tmp
		cp go.sum go.sum.tmp
		go mod tidy
		if ! diff -q go.mod go.mod.tmp >/dev/null 2>&1 || ! diff -q go.sum go.sum.tmp >/dev/null 2>&1; then
			echo "$commit:"
			echo "go mod tidy: go.mod or go.sum is not tidy"
			echo "please run './mk regen' to fix"
			exit 1
		fi
		rm go.mod.tmp go.sum.tmp

		# Check that generated files are up to date.
		before=$(git status --porcelain)
		go generate ./...
		after=$(git status --porcelain)
		if [ "$before" != "$after" ]; then
			echo "$commit:"
			echo "go generate: generated files are out of date"
			git status --short
			echo "please run './mk regen' to fix"
			exit 1
		fi
	)
}

# Iterate over refs being pushed: validate commit subjects, then run
# the per-commit checks against every commit in the push range.
# Stdin format (one line per ref): <local-ref> <local-sha> <remote-ref> <remote-sha>.
zero=0000000000000000000000000000000000000000
while IFS=' ' read -r local_ref local_sha remote_ref remote_sha; do
	[ -z "$local_sha" ] && continue
	[ "$local_sha" = $zero ] && continue # branch deletion
	if [ "$remote_sha" = $zero ]; then
		commits=$(git rev-list "$local_sha" --not --remotes)
	else
		commits=$(git rev-list "$remote_sha..$local_sha")
	fi
	[ -z "$commits" ] && continue

	"$commit_msg_check" $commits

	for commit in $commits; do
		check_commit "$commit"
	done
done <<EOF
$refs
EOF
