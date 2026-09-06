---
name: jujutsu
description: Use for any version-control operation in this repo (commit, describe, squash, rebase, push, conflict resolution). This repo is colocated jj+git — read-only git commands are fine, but mutations go through jj. Covers non-interactive invariants, this repo's bookmark/push policy, and op-log recovery.
allowed-tools: Bash(jj *), Bash(git log*), Bash(git diff*), Bash(git show*), Bash(git status*)
---

# Jujutsu (jj) in act3

Act3 is a colocated jj + git repo: `.jj/` and `.git/` both exist.
This skill documents what's specific to this repo and this
environment — assume general jj knowledge. For anything not covered
below, `jj <cmd> --help`.

This skill was last reviewed against jj **0.40.0**. If the installed
version is newer, some flags, defaults, or command names may have
shifted — when something unexpected happens, check `--help` before
assuming the skill is right.

## Colocated repo rules

- Read-only git commands are fine: `git log`, `git diff`, `git show`,
  `git status`, `git blame`. CLAUDE.md assumes they work.
- Don't mutate via git: no `git commit`, `git rebase`, `git reset`,
  `git cherry-pick`, `git checkout <branch>`. Go through jj so the op
  log stays coherent.
- If git-side state changes anyway, jj re-imports on the next `jj`
  command (expect an "Abandoned N commits" / "Rebased" notice).

## Non-interactive invariants

No editor in this environment. Commands that open one will hang.

- **Always pass `-m`**: `jj desc -m`, `jj squash -m`, `jj new -m`,
  `jj commit -m`.
- **Don't run** `jj split`, `jj resolve`, `jj squash -i`, `jj diffedit`,
  or `jj desc` without `-m`. All interactive.
  - Splitting: use `jj restore <paths>` or `jj squash --from <id>
    --into <id>` instead.
  - Resolving conflicts: edit files directly, remove conflict markers,
    `jj st` to verify.
- After any mutation (`squash`, `abandon`, `rebase`, `restore`, `new`),
  run `jj st` to confirm.

## Workflow in this repo

**`@` is usually empty,** but run `jj st` before starting work.
(If `@` is not empty, run `jj new -m "..."` before making edits.)

```sh
jj st                               # confirm whether empty
jj desc -m "short imperative msg"   # describe the empty @
# ... edit files ...
jj st                               # confirm
jj diff                             # review
jj new                              # finalize; leave a fresh empty @
```

Change the message mid-flight with `jj desc -m "new msg"`.

**Commit message style** — first line in imperative mood, sentence
case, no trailing period, ~50 chars. Follow with a blank line and
additional paragraphs as useful (motivation, caveats, what was tried,
links to issues). Keep the subject terse; let the body carry detail.

Prefix the subject with the package or directory that is semantically
central to the change (e.g. `model:`, `view:`, `database:`). Use
`all:` only when there truly isn't a single motivating location.

Subject-only examples from the existing log:

- `model: create pass1 statsDir with 0o755 mode`
- `view: use paperclip to indicate attachment`
- `web: drop redundant db.Close before degraded-mode db reset`

When a change warrants it:

```
web: drop redundant db.Close before degraded-mode db reset

The connection is read-only, os.Remove unlinks open files fine on
Linux, and serveDegraded already defers db.Close after the server
shuts down — which is the right moment, since closing inside the
handler would race with a concurrent GET / hitting TableStats.
```

The first line should be kept very short: about 50 columns at most.
Subsequent paragraphs should be hard-wrapped to about 70 columns.

## Recovery

`jj undo` reverses the last operation. For anything deeper, use the op
log — it's the real escape hatch, almost always better than trying to
reconstruct state manually:

```sh
jj op log                # operations, newest first
jj op diff --op <op-id>  # what did this op do?
jj op restore <op-id>    # rewind the whole repo to that state
```

Reach for `jj op restore` after an accidental `abandon`, a botched
`rebase`, or any mutation that went sideways.

## Bookmarks and pushing

**Policy here:** no bookmarks for local or experimental work. Create or
advance a bookmark only when something is about to be pushed. Local
chains don't need names — `jj log` shows them fine via change IDs.

`jj git push` only pushes bookmarks; a commit with no bookmark at or
above it can't be published. That's the one hard rule.

**Publishing to `main`** (the common case — solo repo, linear history):

```sh
jj git fetch                        # sync with origin first
jj bookmark move main --to @-       # @- because @ is in-progress
jj git push                         # pushes main
```

Use `@-` unless `@` is finished and you specifically want to publish it.

**Publishing a feature branch** (occasionally, for things worth
sharing):

```sh
jj git push --change <rev>          # auto-creates push-<id> bookmark
```

Don't invent a bookmark name — let jj name it.

**Pre-push checklist:**

1. `jj log` — do the commits to push look right?
2. `jj diff -r <bookmark>..@-` — sanity-check the full diff.
3. User has explicitly asked to push. Never push unprompted.

Force-push (`--force-with-lease`) only when the user asks.
