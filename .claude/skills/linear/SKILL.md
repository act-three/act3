---
name: linear
description: Guidelines for working with Linear issues in this project. Use when the user references a Linear issue (e.g. A3-NN, linear.app/act-three/...), asks to pick up work from Linear, or asks to create/update a Linear issue.
---

# Linear

Issues are tracked in the Act Three Linear workspace.
Access Linear via the `mcp__claude_ai_Linear__*` tools —
do not shell out to a `lin` CLI or hit the Linear GraphQL API directly.

## Workspace shape

The workspace is deliberately minimal.
Don't invent structure that isn't there:

- One team: `Act Three` (key `A3`).
- One active project: `Beta Launch`. Issues may also have no project.
- No initiatives, no cycles, no milestones.
- Four labels: `Security`, `Bug`, `Feature`, `Task`. Don't create new ones without asking.
- Statuses: `Backlog`, `Todo`, `In Progress`, `Done`, `Canceled`, `Duplicate`.
- Default assignee is `me` (the user running Claude Code).

## Verify against the codebase before acting

Issue descriptions can be stale, wrong, or describe code that has since changed.
Before doing work based on an issue:

1. Read the issue with `get_issue` for the full description, not just the truncated list view.
2. Open the files and symbols it references and confirm they still exist and behave as described.
3. If the description disagrees with the code, surface that to the user before proceeding — don't silently "fix" something the issue claims is broken if it isn't, and don't assume the issue is right if the code looks fine.

## Keep Linear in sync while you work

If the current task came from a Linear issue, keep the issue reflecting reality
as you go. This holds even when the work is non-code
(investigation, design discussion, ops tasks):

- **Starting work**: move the issue to `In Progress` via `save_issue`.
- **Finishing work**: move it to `Done`. If the work wrapped up without
  actually implementing the issue (e.g. decided it was invalid, already fixed,
  or superseded), use `Canceled` or `Duplicate` and leave a comment explaining.
- **Significant findings or scope changes**: leave a comment via `save_comment`.
  Examples: discovered the bug was already fixed, split the work into sub-issues,
  chose a different approach than the issue proposed, found a related problem.
- **Splitting work**: create sub-issues with `save_issue` using `parentId`.
  See A3-77 → A3-81..A3-93 for the pattern.

Don't update Linear for work that didn't originate from a Linear issue.
Not every commit needs a ticket.

## Reference the issue in commits

When a commit implements (or partially implements) a Linear issue,
reference the issue ID in the commit message so the link is discoverable
from `git log`. Use a trailer on its own line at the end of the body:

```
model: create pass1 statsDir with 0o755 mode

Matches the rest of the codebase; the directory holds ffmpeg first-pass
log files and has no reason to be group- or world-writable.

Fixes: ACT-123
```

Use `Fixes:` for the commit that closes the issue, `Refs:` for partial
work or follow-ups. Use the full identifier as Linear returns it
(e.g. `ACT-123`), not the shorthand.

## Reference the commit in Linear

The mirror image: a Linear comment about landed work must cite the
landed commit id, not just the subject line. Subjects can be retitled
at landing (ACT-231 landed as #123 under a different subject than the
local commit), so the id is the only durable link. Read it from
`main@origin` — the user will usually have fetched already — e.g.:

```
Landed as 6153950a171e, `model: record tombstones for changed slugs` (#123).
```

## Attribution

Anything you write into Linear on the user's behalf
(issue descriptions, comments) must start with a
`*— from Claude*` line so it isn't mistaken for the user's own words.
Example:

```
*— from Claude*

Split out from A3-81 as "Fix C"...
```

This applies to both the `description` field of `save_issue` and the `body` field of `save_comment`.

## Status conventions

When creating issues:

- Assigned to someone → `Todo`.
- Unassigned → `Backlog`.

New issues default to project `Beta Launch` unless the user says otherwise
or the work clearly belongs outside that project's scope.
