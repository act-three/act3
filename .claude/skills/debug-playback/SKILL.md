---
name: debug-playback
description: Guidelines for debugging playback issues. Provides debugging suggestions and follow-up steps. Use when user asks to debug playback and provides an episode URL or playlist URL.
compatibility: This skill works best on macOS with mediastreamvalidator and avmediainfo available.
---

# Debug Playback

Note that current Chrome DOES support native HLS playback.

## Instructions

### Phase 1. Clarify

Clarify if necessary. What is the playback problem? Black screen,
glitchy audio, glitchy video, buffering, etc.

### Phase 2. Investigate

Use curl, mediastreamvalidator, and avmediainfo at your discretion to
figure out what is going wrong.

Do not attempt to implement a fix until Phase 5.

When the issue is in the encoded output (e.g. mediastreamvalidator
reports errors on the fMP4 segments), check whether the problem is
in our encoding *args* or the encoding *environment*:

- Download the production fMP4 + playlist and validate locally
  (`mediastreamvalidator` works with `file://` paths, no HTTP
  server needed).
- If the production file fails locally too, the issue is in the
  encoded file itself, not in how we serve it.
- Check whether encoding the same source with the *local* ffmpeg
  produces valid output. If local works but production doesn't,
  the bug might be in the Docker ffmpeg version, not our code.

Sometimes these tools will reveal other problems that aren't
expected to break playback. Document those but stay focused on
fixing playback.

### Phase 3. Build Small Test Case

Once a cause is found, fetch the relevant assets and write a unit test
to reliably reproduce the failure. We won't necessarily keep all this
video data around long term, but it's useful to have it on hand during
debugging.

Do not attempt to implement a fix until Phase 5.

Useful assets:

1. Source video. Available to download from the episode details page.
2. Playlists.
3. HLS segment data.

Process the source video locally:

1. Extract a short clip (about 4 segments worth) from the source
   video.
2. Process the clip using package `video/ffmpeg` to produce a
   valid HLS package like what we generate for production.
3. Verify that this output yields the same error as the generated
   production HLS package.

If these steps don't yield the same error, try variations:

- Use `docker run act3-ffmpeg /out/ffmpeg ...` to encode with
  the production ffmpeg instead of the local one.
- Try longer clips, the full video, or different encode settings.
- Try 2-pass encoding (Pass1Combined + Pass2Single) vs 1-pass.

Verify that the test reproduces the same problem that was
originally reported or observed in production.

### Phase 4. Minimize Test Case

Try to minimize the size of the test data.

Do not attempt to implement a fix until Phase 5.

Prefer generating synthetic sources over checking in large files.
A good pattern:

1. Use `docker run act3-ffmpeg /out/ffmpeg` to generate a tiny
   synthetic source (e.g. `testsrc2` + `sine`, AV1+EAC3, ~2s).
2. Encode it through the HLS pipeline in the same Docker
   container.
3. Validate with `mediastreamvalidator` on the host.

This keeps the test self-contained (no fixtures to check in, no
server dependency) while using the exact production ffmpeg.

Verify that the minimized testcase still reproduces the same
problem originally reported or observed in production.

See `video/ffmpeg/ffmpeg_test.go` for an example.

### Phase 5. Fix

Write a fix and verify that the unit test passes.
