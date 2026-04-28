#!/usr/bin/env bash
# Inject the jujutsu skill at session start so jj rules are in
# context from turn 1 without needing an explicit /jujutsu call.
set -euo pipefail

skill="$CLAUDE_PROJECT_DIR/.claude/skills/jujutsu/SKILL.md"

{
	echo "This repo uses jujutsu (jj) on top of git. Always use jj for mutations (never git commit/rebase/reset/cherry-pick). Skill content follows:"
	echo
	cat "$skill"
} | jq -Rs --arg event SessionStart '{hookSpecificOutput: {hookEventName: $event, additionalContext: .}}'
