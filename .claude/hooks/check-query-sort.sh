#!/bin/bash
# Verify that query names in database/query.sql are sorted alphabetically.
cat >/dev/null

names=$(grep -o '\-\- name: [^ ]*' "$CLAUDE_PROJECT_DIR/database/query.sql" | sed 's/-- name: //')
sorted=$(echo "$names" | sort)

if [ "$names" != "$sorted" ]; then
	echo "database/query.sql: query names are not sorted alphabetically. Please keep them sorted by name." >&2
	exit 2
fi
