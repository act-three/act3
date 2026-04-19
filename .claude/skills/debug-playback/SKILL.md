---
name: debug-playback
description: Guidelines for debugging playback issues. Provides debugging suggestions and follow-up steps. Use when user asks to debug playback and provides an episode URL or playlist URL.
compatibility: This skill works best on macOS with mediastreamvalidator and avmediainfo available.
---

# Debug Playback

Note that current Chrome DOES support native HLS playback.

## How to use subagents in this workflow

Playback debugging burns context fast: Playwright snapshots, curl
bodies, ffmpeg logs, mediastreamvalidator output, and downloaded
segment data all pile up. To keep the main conversation coherent and
to enforce the phase boundaries, **delegate each investigative phase
to its own subagent**, then **spawn a separate auditor subagent** to
independently verify the previous phase's conclusions before
proceeding.

Rules for using subagents here:

- **One phase, one subagent.** Don't combine phases — the point is
  that Phase 2 can't start fixing, Phase 3 can't skip reproduction,
  etc. A narrow brief enforces the boundary.
- **Audit independently.** An auditor must re-run the key check
  itself, not just read the prior agent's notes. Treat the earlier
  subagent's claims as unverified until the auditor confirms them.
- **Ask for concise reports.** Each subagent should return a short
  findings summary (what it observed, what it concluded, what
  artifacts it left on disk), not a transcript. The main agent
  synthesizes across phases.
- **Pass artifacts by path, not by content.** Have subagents write
  downloaded segments, playlists, logs, and test inputs to disk and
  report the paths. The next subagent reads from disk.
- Use `subagent_type: "general-purpose"` unless another type fits
  better (e.g. `Explore` for pure code reading in Phase 5 prep).

## Instructions

### Phase 1. Clarify (main agent)

Clarify if necessary. What is the playback problem? Black screen,
glitchy audio, glitchy video, buffering, etc.

Do this in the main conversation — it needs the user in the loop.

### Phase 2. Investigate (subagent)

Spawn an **investigation subagent**. Brief it with:

- The episode/playlist URL and the user's description of the symptom.
- Permission to use Playwright MCP and Chrome Devtools MCP to load
  the page and observe playback, to check the browser console, and
  to run curl, mediastreamvalidator, and avmediainfo.
- Explicit instruction: **do not attempt a fix, do not modify source
  code.** Its only job is to identify the likely cause.
- Guidance that when the issue looks like it's in the encoded output,
  it should distinguish encoding *args* vs encoding *environment*:
  - Download the production fMP4 + playlist and validate locally
    (`mediastreamvalidator` accepts `file://`, no HTTP needed).
  - If the production file fails locally too, the issue is in the
    encoded file itself, not in how we serve it.
  - Check whether encoding the same source with the *local* ffmpeg
    produces valid output. If local works but production doesn't,
    the bug might be in the Docker ffmpeg version, not our code.
- Instruction to note any incidental findings but stay focused on
  the reported playback problem.

Ask the subagent to return: the suspected cause, the evidence for it,
and paths to any saved artifacts (playlists, segments, logs).

Then spawn an **audit subagent**. Brief it with the original symptom,
the investigator's suspected cause, and the artifact paths. Its job:
independently re-run the decisive check (e.g. re-validate the saved
segment, re-fetch the playlist) and confirm or refute the
investigator's conclusion. If the audit refutes it, loop back.

### Phase 3. Build Small Test Case (subagent)

Spawn a **reproduction subagent**. Brief it with the confirmed cause
from Phase 2 and the relevant artifact paths. Explicit instruction:
**do not attempt a fix.** Its job is to produce a failing unit test.

Useful assets it can fetch:

1. Source video (from the episode details page).
2. Playlists.
3. HLS segment data.

Process the source video locally:

1. Extract a short clip (about 4 segments worth) from the source.
2. Process the clip using package `video/ffmpeg` to produce a
   valid HLS package like what we generate for production.
3. Verify that this output yields the same error as the generated
   production HLS package.

If these steps don't yield the same error, try variations:

- Use `docker run act3-ffmpeg /out/ffmpeg ...` to encode with
  the production ffmpeg instead of the local one.
- Try longer clips, the full video, or different encode settings.
- Try 2-pass encoding (Pass1Combined + Pass2Single) vs 1-pass.

Ask the subagent to return: the test path, how to run it, and the
exact failure output — which must match the symptom from Phase 1.

Then spawn an **audit subagent** with the test path and the original
symptom. Its job: run the test itself and confirm the failure mode
matches what the user originally reported. A test that fails for a
*different* reason is not a valid reproduction — send it back.

### Phase 4. Minimize Test Case (subagent)

Spawn a **minimization subagent** with the Phase 3 test as input.
Explicit instruction: **do not attempt a fix.**

Prefer generating synthetic sources over checking in large files.
A good pattern:

1. Use `docker run act3-ffmpeg /out/ffmpeg` to generate a tiny
   synthetic source (e.g. `testsrc2` + `sine`, AV1+EAC3, ~2s).
2. Encode it through the HLS pipeline in the same Docker container.
3. Validate with `mediastreamvalidator` on the host.

This keeps the test self-contained (no fixtures to check in, no
server dependency) while using the exact production ffmpeg.

See `video/ffmpeg/ffmpeg_test.go` for an example.

Ask the subagent to return: the minimized test path, its size vs the
Phase 3 test, and the failure output.

Then spawn an **audit subagent** with the minimized test and the
original symptom. Its job: run the minimized test and confirm the
failure still matches the original reported problem — not a
degenerate failure introduced by over-minimization.

### Phase 5. Fix (main agent, optionally with a subagent)

Now the main agent has a small, audited, reproducing test and a
confirmed root cause. Write the fix and verify the test passes.

For fixes that require broad code reading (tracing call graphs,
finding all usages), spawn an `Explore` subagent to gather context,
then apply the fix in the main agent so the edits are visible in the
conversation.

After the fix compiles and the new test passes, spawn a **final audit
subagent** with: the original symptom, the fix diff, and the test
path. Its job: run `go test ./...`, re-run the minimized test, and
(if feasible) reload the dev server and re-check the original
episode in Playwright/Devtools to confirm the user-visible symptom
is gone.
