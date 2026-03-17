#!/bin/bash
set -e

# Post-write/post-edit hook: goimports, dprint, query sort check, sqlc, CSS/JS bundles.
# Receives tool JSON on stdin; dispatches by file path.
cd "$CLAUDE_PROJECT_DIR"
abs=$(jq -r '.tool_input.file_path')
f=${abs#"$CLAUDE_PROJECT_DIR"/}
exec </dev/null >/dev/null

case "$f" in
	*.go)
		goimports -w "$abs"
		;;
	*.js | *.css | *.json | *.sql | *.sh)
		dprint fmt "$abs" 2>/dev/null || true
		;;
esac

case "$f" in
	database/query.sql)
		./.claude/hooks/check-query-sort.sh
		;;
esac

case "$f" in
	database/ddl/* | database/query.sql | database/sqlc.json)
		go generate ./database 2>/dev/null
		;;
esac

case "$f" in
	*.js | *.css | ui/*.go | view/*.go)
		go generate 2>/dev/null
		;;
esac
