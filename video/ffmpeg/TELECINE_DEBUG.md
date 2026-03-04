## S02E11 Telecine EXTINF Mismatch — Debug Notes

### Bug

S02E11 ("Maneuvers") stutters/jumps back in time around 30:20–30:26.

Root cause: MPEG-2 soft-telecine (2:3 pulldown) source causes
ffmpeg's HLS muxer to write EXTINF values that don't match actual
fMP4 segment PTS spans by ~117ms. Triggers mediastreamvalidator
"Playlist vs segment duration mismatch" errors. Only manifests with
`medium` preset (B-frame delay introduces timing offset).

### Phase 4 — Minimize Test Case ✓

Self-contained synthetic test (`TestMPEG2TelecineEXTINFMismatch_Synthetic`)
reproduces the 117ms EXTINF mismatch without any production clip.

#### Go Bitstream Patcher

`patchTelecine()` in `telecine_test.go`:
- Clears `progressive_sequence` in MPEG-2 sequence extension
- Sets `repeat_first_field=1` on alternate picture coding extensions
- Rewrites PES PTS/DTS timestamps with alternating VFR deltas:
  - S-frame: +3003 ticks at 90kHz (~33ms)
  - L-frame: +4504 ticks at 90kHz (~50ms)
- Input must be MPEG-2 PS in VOB format (`-f vob`, not `-f mpeg`
  which produces MPEG-1 PS without PES PTS/DTS headers)
- Generated with `-bf 0` (no B-frames) so each PES packet has
  exactly one picture and the DTS/PTS relationship is simple

#### Pipeline

1. Generate 65s MPEG-2 VOB at **24fps** (`-bf 0 -g 12`)
2. Patch telecine flags + VFR PTS/DTS in Go
3. Remux via MPEG-TS intermediate (regenerates clean container
   timestamps from PES PTS/DTS) with `mpeg2_metadata` BSF to set
   sequence-header `frame_rate=60000/1001` (≈59.94fps)
4. Remux TS + fresh audio → MKV with `-default_mode passthrough`
5. Encode with medium preset, validate with mediastreamvalidator

Result: mediastreamvalidator detects "Playlist vs segment duration
mismatch" — segment duration 10.3103, playlist duration 10.4271
(difference ≈ 117ms), matching the production bug exactly.

#### Critical Detail: 24fps Not 30fps

The source MUST be generated at **24fps** (24000/1001), not 30fps.
Real telecine content has ~24fps coded pictures with field-doubling
for display; the MPEG-2 decoder outputs one frame per coded picture
(~24fps). The libx265 encoder's B-frame delay pattern depends on
the input frame rate — at 30fps the delay is different and the
EXTINF mismatch doesn't manifest.

#### Root Cause Analysis

The HLS muxer computes EXTINF as `(keyframe_pkt.pts - vs.end_pts)`
in output packet timebase. `vs.end_pts` is set from the first
video ref packet's PTS — which includes the B-frame encoder delay
offset. But the fMP4 segment's actual PTS span is measured from
the first sample in the segment (after edit-list adjustment). The
difference equals the encoder delay: ~117ms with medium preset's
bframes=4 at ~24fps VFR input.

### Phase 5 — Fix ✓

`fixEXTINF()` in `hlsfix.go` post-processes the HLS playlist
after encoding. For each segment byte range it:

1. Parses the fMP4 fragment's `sidx` box for the track timescale
2. Finds the video `traf` (identified by `trun` with CTS offsets)
3. Reads `tfdt` for base decode time, scans `trun` samples to
   compute `(max_PTS + duration) - min_PTS` in timescale ticks
4. Replaces the `#EXTINF:` line with the corrected duration
5. Updates `#EXT-X-TARGETDURATION` to match the longest segment

Called at the end of `Pass2Single` before returning the playlist.
The fix is invisible to callers — they receive a correct playlist.

mediastreamvalidator confirms: no more "Playlist vs segment
duration mismatch" errors on either the synthetic or production
telecine sources.

### Key Files

- `video/ffmpeg/hlsfix.go` — fixEXTINF, fMP4 box parsers
  (sidx, moof, traf, tfdt, trun)
- `video/ffmpeg/telecine_test.go` — patchTelecine, encodePTS,
  decodePTS, all telecine tests
- `video/ffmpeg/ffmpeg_test.go` — other encode tests,
  Docker/preset setup
- `video/ffmpeg/ffmpeg.go` — Pass1Combined, Pass2Single,
  fpsPassthrough, hlsOutputArgs
