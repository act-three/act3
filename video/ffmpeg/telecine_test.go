package ffmpeg

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// MPEG-2 PES timestamp helpers
// ---------------------------------------------------------------------------

// encodePTS writes a 33-bit PTS/DTS value into a 5-byte buffer using
// the MPEG PES timestamp encoding. markerHigh is the upper 4 bits of
// the first byte (0010 for PTS-only, 0011 for PTS in PTS+DTS, 0001
// for DTS).
func encodePTS(buf []byte, val uint64, markerHigh byte) {
	buf[0] = (markerHigh << 4) | byte((val>>29)&0x0E) | 0x01
	buf[1] = byte((val >> 22) & 0xFF)
	buf[2] = byte((val>>14)&0xFE) | 0x01
	buf[3] = byte((val >> 7) & 0xFF)
	buf[4] = byte((val<<1)&0xFE) | 0x01
}

// decodePTS reads a 33-bit PTS/DTS value from a 5-byte PES timestamp
// field.
func decodePTS(buf []byte) uint64 {
	var v uint64
	v |= (uint64(buf[0]) & 0x0E) << 29
	v |= uint64(buf[1]) << 22
	v |= (uint64(buf[2]) & 0xFE) << 14
	v |= uint64(buf[3]) << 7
	v |= (uint64(buf[4]) & 0xFE) >> 1
	return v
}

// ---------------------------------------------------------------------------
// patchTelecine — MPEG-2 PS bitstream patcher
// ---------------------------------------------------------------------------

// patchTelecine patches an MPEG-2 Program Stream (VOB format) to
// simulate soft-telecine 2:3 pulldown. It performs three modifications:
//
//  1. Clears progressive_sequence in every sequence extension header
//     (signals interlaced/telecine content to decoders).
//  2. Sets repeat_first_field=1 on alternate picture coding extensions
//     (L-frames display for 3 fields ≈50ms; S-frames for 2 ≈33ms).
//  3. Rewrites PES PTS/DTS timestamps on video packets with
//     alternating VFR deltas so that the presentation timeline
//     matches real 2:3 pulldown cadence:
//     L-frame (repeat_first_field=1): ~50ms  (+4504 ticks at 90kHz)
//     S-frame (repeat_first_field=0): ~33ms  (+3003 ticks at 90kHz)
//
// The input must be generated with -f vob (MPEG-2 PS, not MPEG-1 PS)
// and -bf 0 (no B-frames) at 24000/1001 fps (matching real telecine
// content, where the coded frame rate is ~24fps). With no B-frames,
// PTS and DTS in the VOB still differ by one frame period due to the
// muxer's reorder buffer; we preserve this relationship while shifting
// both to the VFR cadence.
func patchTelecine(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)

	// ---- Pass 1: patch MPEG-2 ES headers (inside PES payloads) ----
	//
	// We scan for MPEG-2 start codes (00 00 01 xx). Extension start
	// code 0xB5 carries both sequence-extension (ext_id 1) and
	// picture-coding-extension (ext_id 8).
	frameCount := 0
	for i := 0; i < len(out)-10; {
		if out[i] != 0 || out[i+1] != 0 || out[i+2] != 1 {
			i++
			continue
		}

		if out[i+3] == 0xB5 {
			extID := (out[i+4] >> 4) & 0x0F
			switch extID {
			case 1: // Sequence extension
				// byte i+5 bit 3 = progressive_sequence — clear it.
				out[i+5] &^= 1 << 3
			case 8: // Picture coding extension
				if frameCount%2 == 1 {
					// L-frame: set repeat_first_field (bit 1
					// of the byte at offset +7 from start code).
					out[i+7] |= 0x02
				}
				frameCount++
			}
		}
		i++
	}

	// ---- Pass 2: rewrite video PES PTS/DTS ----
	//
	// MPEG-2 PS (VOB) PES header layout for video (stream 0xE0–0xEF):
	//
	//   [0..2]  00 00 01          start code prefix
	//   [3]     stream_id         0xE0–0xEF for video
	//   [4..5]  PES packet length (big-endian)
	//   [6]     flags byte 1     (10xx xxxx for MPEG-2)
	//   [7]     flags byte 2     bits 7:6 = pts_dts_flags
	//   [8]     PES header data length
	//   [9..]   optional PTS (5 bytes) then optional DTS (5 bytes)
	//
	// pts_dts_flags: 0b10 = PTS only, 0b11 = PTS + DTS.
	//
	// The VOB muxer with -bf 0 writes PTS = DTS + 3003 (one frame
	// reorder delay at 29.97fps). We preserve that offset while
	// shifting both PTS and DTS to follow the VFR cadence.

	const (
		ticksL        = 4504  // ~50.04ms at 90kHz — L-frame (3 fields)
		ticksS        = 3003  // ~33.37ms at 90kHz — S-frame (2 fields)
		reorderOffset = 3754  // PTS-DTS offset: ~1 frame at 24fps
		startDTS      = 45000 // 0.5s — matches typical VOB start
	)

	dts := uint64(startDTS)
	videoIdx := 0

	for i := 0; i < len(out)-19; {
		if out[i] != 0 || out[i+1] != 0 || out[i+2] != 1 {
			i++
			continue
		}

		streamID := out[i+3]
		if streamID < 0xE0 || streamID > 0xEF {
			i++
			continue
		}

		// Verify this looks like an MPEG-2 PES header: byte 6
		// must have bits 7:6 == 0b10.
		if i+9 > len(out) || (out[i+6]>>6) != 2 {
			i += 4
			continue
		}

		ptsDTSFlags := (out[i+7] >> 6) & 0x03
		headerLen := int(out[i+8])

		if ptsDTSFlags >= 2 && i+9+5 <= len(out) && headerLen >= 5 {
			pts := dts + reorderOffset

			// Write PTS.
			ptsMarker := byte(0x02) // PTS-only marker
			if ptsDTSFlags == 3 {
				ptsMarker = 0x03 // PTS marker when DTS also present
			}
			encodePTS(out[i+9:i+14], pts, ptsMarker)

			// Write DTS if present.
			if ptsDTSFlags == 3 && i+14+5 <= len(out) && headerLen >= 10 {
				encodePTS(out[i+14:i+19], dts, 0x01)
			}

			// Advance DTS by the telecine delta for this frame.
			// Frame 0 is S (even index), frame 1 is L (odd), etc.
			if videoIdx%2 == 1 {
				dts += ticksL
			} else {
				dts += ticksS
			}
			videoIdx++
		}

		// Skip past PES header to avoid re-matching embedded start
		// codes in the header extension bytes.
		i += 9 + headerLen
	}

	return out
}

// ---------------------------------------------------------------------------
// Unit tests — no Docker / ffmpeg required
// ---------------------------------------------------------------------------

// TestEncodePTSRoundTrip verifies that encodePTS/decodePTS are
// inverses for a range of PTS values.
func TestEncodePTSRoundTrip(t *testing.T) {
	values := []uint64{
		0,
		90000,
		93003,         // 90000 + 3003
		94504,         // 90000 + 4504
		1 << 32,       // large value near 33-bit limit
		(1 << 33) - 1, // max 33-bit value
		270000,        // 3 seconds at 90kHz
	}

	for _, want := range values {
		buf := make([]byte, 5)
		encodePTS(buf, want, 0x02) // marker=0010 (PTS-only)
		got := decodePTS(buf)
		if got != want {
			t.Errorf("round-trip failed: encoded %d, decoded %d "+
				"(buf=%02x %02x %02x %02x %02x)",
				want, got, buf[0], buf[1], buf[2], buf[3], buf[4])
		}
	}
}

// TestPatchTelecineTimestamps verifies that patchTelecine produces
// correct PES PTS/DTS values with the expected VFR cadence using a
// minimal synthetic MPEG-2 PS byte sequence (no ffmpeg needed).
func TestPatchTelecineTimestamps(t *testing.T) {
	const reorderOffset = 3754 // must match patchTelecine

	// Build a minimal PES packet for video stream 0xE0 with PTS+DTS
	// and embedded sequence-extension + picture-coding-extension.
	buildPES := func(pts, dts uint64) []byte {
		// PES header:
		//   [0..3]  00 00 01 E0   start code + stream ID
		//   [4..5]  length (filled in at end)
		//   [6]     0x80          MPEG-2 marker (10 00 0000)
		//   [7]     0xC0          pts_dts_flags = 0b11 (both)
		//   [8]     0x0A          header data length = 10
		//   [9..13] PTS (5 bytes, marker_high=0x03)
		//   [14..18] DTS (5 bytes, marker_high=0x01)
		var pkt []byte
		pkt = append(pkt, 0x00, 0x00, 0x01, 0xE0)
		pkt = append(pkt, 0x00, 0x00) // length placeholder
		pkt = append(pkt, 0x80, 0xC0, 0x0A)

		ptsBuf := make([]byte, 5)
		encodePTS(ptsBuf, pts, 0x03)
		pkt = append(pkt, ptsBuf...)

		dtsBuf := make([]byte, 5)
		encodePTS(dtsBuf, dts, 0x01)
		pkt = append(pkt, dtsBuf...)

		// Payload: sequence extension (ext_id=1) then picture
		// coding extension (ext_id=8).
		pkt = append(pkt, 0x00, 0x00, 0x01, 0xB5)
		pkt = append(pkt, 0x10)       // ext_id=1 high nibble
		pkt = append(pkt, 0x08)       // progressive_sequence=1 (bit 3)
		pkt = append(pkt, 0x00, 0x00) // padding

		pkt = append(pkt, 0x00, 0x00, 0x01, 0xB5)
		pkt = append(pkt, 0x80)                   // ext_id=8
		pkt = append(pkt, 0x00, 0x00)             // padding
		pkt = append(pkt, 0x00)                   // repeat_first_field lives in bit 1
		pkt = append(pkt, 0x00, 0x00, 0x00, 0x00) // padding

		binary.BigEndian.PutUint16(pkt[4:6], uint16(len(pkt)-6))
		return pkt
	}

	// Two PES packets at original CFR timing (24fps, ~3754 ticks/frame).
	var stream []byte
	stream = append(stream, buildPES(48754, 45000)...)
	stream = append(stream, buildPES(52508, 48754)...)

	patched := patchTelecine(stream)

	// Extract results.
	type frameInfo struct {
		pts, dts    uint64
		repeatFirst bool
		progressive bool
	}
	var frames []frameInfo

	for i := 0; i < len(patched)-19; {
		if patched[i] != 0 || patched[i+1] != 0 || patched[i+2] != 1 {
			i++
			continue
		}
		if patched[i+3] < 0xE0 || patched[i+3] > 0xEF {
			i++
			continue
		}
		if (patched[i+6] >> 6) != 2 {
			i += 4
			continue
		}
		ptsDTSFlags := (patched[i+7] >> 6) & 0x03
		headerLen := int(patched[i+8])
		if ptsDTSFlags < 2 || headerLen < 5 {
			i += 4
			continue
		}

		pts := decodePTS(patched[i+9 : i+14])
		dts := pts
		if ptsDTSFlags == 3 && headerLen >= 10 {
			dts = decodePTS(patched[i+14 : i+19])
		}

		fi := frameInfo{pts: pts, dts: dts}
		payloadStart := i + 9 + headerLen
		for j := payloadStart; j < len(patched)-8; j++ {
			if patched[j] != 0 || patched[j+1] != 0 || patched[j+2] != 1 || patched[j+3] != 0xB5 {
				continue
			}
			extID := (patched[j+4] >> 4) & 0x0F
			if extID == 1 {
				fi.progressive = (patched[j+5] & 0x08) != 0
			}
			if extID == 8 {
				fi.repeatFirst = (patched[j+7] & 0x02) != 0
				break
			}
		}
		frames = append(frames, fi)
		i += 9 + headerLen
	}

	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}

	// Frame 0: S-frame (even), repeat_first_field=0.
	if frames[0].repeatFirst {
		t.Error("frame 0 should not have repeat_first_field")
	}
	// Frame 1: L-frame (odd), repeat_first_field=1.
	if !frames[1].repeatFirst {
		t.Error("frame 1 should have repeat_first_field")
	}

	// progressive_sequence should be cleared.
	for i, f := range frames {
		if f.progressive {
			t.Errorf("frame %d: progressive_sequence should be cleared", i)
		}
	}

	// DTS values: start at 45000, frame 0 S-delta (3003),
	// frame 1 at 45000+3003=48003.
	// PTS = DTS + reorderOffset (3754).
	wantDTS := []uint64{45000, 48003}
	wantPTS := []uint64{45000 + reorderOffset, 48003 + reorderOffset}
	for i, f := range frames {
		if f.dts != wantDTS[i] {
			t.Errorf("frame %d DTS = %d, want %d", i, f.dts, wantDTS[i])
		}
		if f.pts != wantPTS[i] {
			t.Errorf("frame %d PTS = %d, want %d", i, f.pts, wantPTS[i])
		}
	}

	t.Logf("frame 0: PTS=%d DTS=%d repeat=%v prog=%v",
		frames[0].pts, frames[0].dts, frames[0].repeatFirst, frames[0].progressive)
	t.Logf("frame 1: PTS=%d DTS=%d repeat=%v prog=%v",
		frames[1].pts, frames[1].dts, frames[1].repeatFirst, frames[1].progressive)
}

// ---------------------------------------------------------------------------
// Integration tests — require Docker (ffmpeg/ffprobe) and possibly
// mediastreamvalidator
// ---------------------------------------------------------------------------

// parseProbeLine parses a single line of ffprobe -of flat output and
// returns (key, value). The flat format is:
//
//	frames.frame.N.key="value"   (strings)
//	frames.frame.N.key=value     (numbers)
func parseProbeLine(line string) (key, value string) {
	before, after, ok := strings.Cut(line, "=")
	if !ok {
		return "", ""
	}
	// The key is the last dot-separated component before '='.
	fullKey := before
	dot := strings.LastIndexByte(fullKey, '.')
	if dot >= 0 {
		key = fullKey[dot+1:]
	} else {
		key = fullKey
	}
	value = strings.Trim(after, "\"")
	return key, value
}

// probeFrame holds the fields we care about from ffprobe -show_frames.
type probeFrame struct {
	dtsTime    float64
	dtsValid   bool
	repeatPict int
}

// parseProbeFlat parses ffprobe -of flat -show_frames output into a
// slice of probeFrame, filtering only the fields we need. This avoids
// the CSV parsing issues caused by side_data contamination.
func parseProbeFlat(output string) []probeFrame {
	var frames []probeFrame
	var cur probeFrame
	curIdx := -1

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Determine frame index from the flat key prefix:
		//   frames.frame.N.key=value
		// We only care about lines starting with "frames.frame.".
		if !strings.HasPrefix(line, "frames.frame.") {
			continue
		}

		// Skip side_data lines.
		if strings.Contains(line, "side_data") {
			continue
		}

		// Extract frame index.
		rest := line[len("frames.frame."):]
		before, _, ok := strings.Cut(rest, ".")
		if !ok {
			continue
		}
		idx, err := strconv.Atoi(before)
		if err != nil {
			continue
		}

		// Start a new frame if the index advanced.
		if idx != curIdx {
			if curIdx >= 0 {
				frames = append(frames, cur)
			}
			cur = probeFrame{}
			curIdx = idx
		}

		key, value := parseProbeLine(line)
		switch key {
		case "pkt_dts_time":
			if value != "N/A" {
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					cur.dtsTime = v
					cur.dtsValid = true
				}
			}
		case "repeat_pict":
			if v, err := strconv.Atoi(value); err == nil {
				cur.repeatPict = v
			}
		}
	}
	// Don't forget the last frame.
	if curIdx >= 0 {
		frames = append(frames, cur)
	}
	return frames
}

// TestPatchTelecineIntegration generates a real MPEG-2 stream via
// ffmpeg, patches it, and verifies that ffprobe sees the correct
// repeat_pict and VFR timestamps.
func TestPatchTelecineIntegration(t *testing.T) {
	dir := setupDocker(t)
	ctx := t.Context()

	// Generate a short MPEG-2 PS (VOB format) at 24fps, no
	// B-frames. VOB gives us MPEG-2 PS with proper PES PTS/DTS.
	// 24fps matches the coded frame rate of real telecine content.
	rawPath := filepath.Join(dir, "raw.mpg")
	t.Log("generating MPEG-2 VOB source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=3:size=720x480:rate=24000/1001",
		"-an",
		"-c:v", "mpeg2video", "-b:v", "5000k",
		"-bf", "0",
		"-g", "12",
		"-f", "vob",
		rawPath,
	)

	rawData, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("read raw: %v", err)
	}

	patched := patchTelecine(rawData)
	patchedPath := filepath.Join(dir, "patched.mpg")
	if err := os.WriteFile(patchedPath, patched, 0o644); err != nil {
		t.Fatalf("write patched: %v", err)
	}

	// Probe the patched file for repeat_pict and timestamps using
	// the flat output format to avoid CSV side_data contamination.
	probeCmd := newCmd(ctx, "ffprobe",
		"-show_frames", "-select_streams", "v",
		"-read_intervals", "%+#20",
		"-show_entries", "frame=pkt_dts_time,repeat_pict",
		"-of", "flat",
		patchedPath,
	)
	out, err := probeCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("probe patched: %v\n%s", err, out)
	}

	frames := parseProbeFlat(string(out))
	if len(frames) < 4 {
		t.Fatalf("expected at least 4 frames, got %d\nraw:\n%s",
			len(frames), out)
	}

	for i, f := range frames {
		if i >= 8 {
			break
		}
		t.Logf("frame %2d: dts=%.6f repeat_pict=%d",
			i, f.dtsTime, f.repeatPict)
	}

	// Verify alternating repeat_pict.
	var sawR0, sawR1 bool
	for _, f := range frames {
		if f.repeatPict == 0 {
			sawR0 = true
		}
		if f.repeatPict == 1 {
			sawR1 = true
		}
	}
	if !sawR0 || !sawR1 {
		t.Error("expected both repeat_pict=0 and repeat_pict=1")
	}

	// Verify VFR timestamps: should see both short (~33ms) and
	// long (~50ms) gaps, not constant CFR.
	var shortGaps, longGaps int
	for i := 1; i < len(frames); i++ {
		if !frames[i].dtsValid || !frames[i-1].dtsValid {
			continue
		}
		gap := (frames[i].dtsTime - frames[i-1].dtsTime) * 1000
		if gap > 28 && gap < 40 {
			shortGaps++
		} else if gap > 44 && gap < 56 {
			longGaps++
		}
		if i <= 6 {
			t.Logf("  gap %d→%d: %.1fms", i-1, i, gap)
		}
	}

	if shortGaps == 0 || longGaps == 0 {
		t.Errorf("expected both short (~33ms) and long (~50ms) gaps; "+
			"got %d short, %d long", shortGaps, longGaps)
	}
	t.Logf("VFR gaps: %d short (~33ms), %d long (~50ms)",
		shortGaps, longGaps)
}

// TestMPEG2TelecineEXTINFMismatch_Synthetic is a self-contained
// version of TestMPEG2TelecineEXTINFMismatch that generates a fully
// synthetic MPEG-2 soft-telecine source instead of requiring a
// production clip.
//
// The synthetic source has the same key properties as the production
// clip: MPEG-2 video at ~24fps coded frame rate with
// repeat_first_field (2:3 pulldown), VFR timestamps alternating
// ~33ms/~50ms, and r_frame_rate ≈ 59.94fps. Using 24fps (not 30fps)
// is critical: the production source has 24fps coded pictures, and
// the encoder's B-frame delay pattern (which causes the EXTINF
// mismatch) depends on the input frame rate.
//
// Pipeline:
//  1. Generate ~65s MPEG-2 VOB at 24fps (no B-frames).
//  2. Patch the elementary stream: telecine flags + VFR PTS/DTS.
//  3. Remux through MPEG-TS (to regenerate clean container
//     timestamps from PES PTS/DTS) with mpeg2_metadata BSF setting
//     the sequence-header frame_rate to 60000/1001 (≈59.94fps).
//  4. Remux TS + fresh audio → MKV with -default_mode passthrough
//     so the patched codec frame rate flows into DefaultDuration.
//  5. Encode with medium preset (B-frames trigger the bug).
//  6. Validate HLS output with mediastreamvalidator.
func TestMPEG2TelecineEXTINFMismatch_Synthetic(t *testing.T) {
	if _, err := exec.LookPath("mediastreamvalidator"); err != nil {
		t.Skip("mediastreamvalidator not in PATH")
	}

	dir := setupDocker(t)
	// Must use medium preset — ultrafast uses fewer B-frames and
	// does not reproduce this bug.
	setPreset(t, "medium")
	ctx := t.Context()

	// Step 1: Generate video-only MPEG-2 VOB. We keep audio out of
	// the VOB so that patching video PES timestamps doesn't desync
	// from audio PES timestamps. Audio is added later during remux.
	rawPath := filepath.Join(dir, "raw.mpg")
	t.Log("generating 24fps MPEG-2 VOB source (no B-frames)...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=65:size=720x480:rate=24000/1001",
		"-an",
		"-c:v", "mpeg2video", "-b:v", "5000k",
		"-bf", "0",
		"-g", "12",
		"-f", "vob",
		rawPath,
	)

	// Step 2: Patch telecine flags and PTS/DTS.
	t.Log("patching telecine flags and VFR PTS/DTS...")
	rawData, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("read raw: %v", err)
	}
	patched := patchTelecine(rawData)
	patchedPath := filepath.Join(dir, "patched.mpg")
	if err := os.WriteFile(patchedPath, patched, 0o644); err != nil {
		t.Fatalf("write patched: %v", err)
	}

	// Verify the patch produced VFR timestamps.
	t.Log("verifying patch...")
	verifyCmd := newCmd(ctx, "ffprobe",
		"-show_frames", "-select_streams", "v",
		"-read_intervals", "%+#10",
		"-show_entries", "frame=pkt_dts_time,repeat_pict",
		"-of", "flat",
		patchedPath,
	)
	verifyOut, err := verifyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("verify probe: %v\n%s", err, verifyOut)
	}
	verifyFrames := parseProbeFlat(string(verifyOut))
	for i, f := range verifyFrames {
		if i >= 8 {
			break
		}
		t.Logf("patched frame %2d: dts=%.6f repeat_pict=%d",
			i, f.dtsTime, f.repeatPict)
	}

	// Step 3: Remux to MKV in two stages.
	//
	// Stage 3a: video-only MPEG-TS intermediate. The VOB demuxer
	// sometimes emits packets with unset timestamps that the
	// Matroska muxer rejects, so we go through MPEG-TS first
	// (which regenerates container timestamps from PES PTS/DTS).
	// We also apply mpeg2_metadata BSF to set the MPEG-2 sequence
	// header frame_rate to 60000/1001 (≈59.94fps), matching the
	// production scenario where the MKV DefaultDuration claims
	// 59.94fps but actual coded frames are ~24fps with VFR timing.
	tsPath := filepath.Join(dir, "intermediate.ts")
	t.Log("remuxing patched VOB → MPEG-TS (video only, 59.94fps header)...")
	generate(t,
		"-y",
		"-fflags", "+genpts",
		"-i", patchedPath,
		"-c:v", "copy",
		"-bsf:v", "mpeg2_metadata=frame_rate=60000/1001",
		"-an",
		"-f", "mpegts",
		tsPath,
	)

	// Stage 3b: remux TS video + fresh audio into MKV. Use
	// -default_mode passthrough so the muxer copies the (now-
	// patched) codec frame rate into the track's DefaultDuration.
	srcPath := filepath.Join(dir, "source.mkv")
	t.Log("remuxing MPEG-TS + audio → MKV...")
	generate(t,
		"-y",
		"-i", tsPath,
		"-f", "lavfi", "-i",
		"sine=frequency=440:duration=65:sample_rate=48000",
		"-c:v", "copy",
		"-c:a", "ac3", "-ac", "2",
		"-default_mode", "passthrough",
		srcPath,
	)

	// Step 4: Probe and encode. Verify that r_frame_rate > 30fps
	// (should be ~59.94fps from the patched sequence header).
	srcFile, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	t.Log("probing source...")
	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	t.Logf("probe: %dx%d, %s fps, duration %s, %d kbps",
		probe.Video.Width, probe.Video.Height,
		probe.Video.FrameRate, probe.Duration,
		probe.Video.BitRate/1000)

	if probe.Video.FrameRate.Le(30) {
		t.Fatalf("expected r_frame_rate > 30fps (should be ~59.94), "+
			"got %s", probe.Video.FrameRate)
	}

	// Coded-rate sanity: every coded picture lands in one video
	// packet, so packets/duration is the rate the encoder receives
	// frames at under -fps_mode passthrough — ~24fps for a 2:3
	// pulldown source even when the container claims 59.94. If
	// CodedFrameRate ever drifts up to the display rate, segment
	// boundary math (MinFramesPerSegment etc.) will silently
	// over-count frames per segment and produce ~10s segments
	// instead of ≤6s. See ACT-183.
	codedFps := probe.Video.CodedFrameRate()
	t.Logf("coded fps=%s (packets=%d, duration_ticks=%d, "+
		"timebase=%d/%d)",
		codedFps, probe.Video.PacketCount,
		probe.Video.DurationTicks,
		probe.Video.TimebaseNum, probe.Video.TimebaseDen)
	if codedFps.Gt(30) {
		t.Errorf("expected coded fps ≤ 30 on a 2:3-pulldown "+
			"source, got %s", codedFps)
	}
	if MinFramesPerSegment(codedFps, MinSegmentDuration) >
		MinFramesPerSegment(FrameRate{Num: 30, Den: 1},
			MinSegmentDuration) {
		t.Errorf("MinFramesPerSegment(coded fps) = %d exceeds "+
			"the cap for 30fps; soft-telecine source is being "+
			"measured at the display rate",
			MinFramesPerSegment(codedFps, MinSegmentDuration))
	}

	params := EncodeParams{
		Path:    filepath.Join(dir, MediaName(0)),
		Codec:   "libx265",
		Bitrate: 1500,
		Tag:     "hvc1",
		StatsID: "r0",
	}

	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, dir, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	t.Log("running pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		dir, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}

	// Step 5: Validate with mediastreamvalidator.
	plsPath := filepath.Join(dir, "stream0.m3u8")
	if err := os.WriteFile(plsPath, []byte(playlist), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Log("running mediastreamvalidator...")
	out, err := exec.Command("mediastreamvalidator", plsPath).
		CombinedOutput()
	_ = err

	output := string(out)
	t.Logf("mediastreamvalidator output:\n%s", output)

	if strings.Contains(output, "Playlist vs segment duration mismatch") {
		t.Error("EXTINF mismatch: mediastreamvalidator detected " +
			"\"Playlist vs segment duration mismatch\" — " +
			"EXTINF values do not match actual fMP4 segment " +
			"durations")
	}

	// Step 6: Segment decodability check — detect open GOPs.
	//
	// For each segment, compare the trun sample count (frames the
	// muxer wrote) against what ffprobe can decode standalone. With
	// open GOPs, trailing B-frames reference the next segment's IDR
	// and are undecoded standalone (e.g. trun=252 but ffprobe=249).
	// With closed GOPs, the counts always match.
	mediaData, err := os.ReadFile(filepath.Join(dir, MediaName(0)))
	if err != nil {
		t.Fatalf("read media: %v", err)
	}

	// Parse the init segment byte range from #EXT-X-MAP.
	// Format: #EXT-X-MAP:URI="...",BYTERANGE="size@offset"
	var initSeg []byte
	lines := strings.Split(playlist, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "#EXT-X-MAP:") {
			continue
		}
		m := byterangeRe.FindStringSubmatch(line)
		if m == nil {
			// Try the quoted BYTERANGE= form.
			_, after, ok := strings.Cut(line, "BYTERANGE=\"")
			if !ok {
				break
			}
			rest := after
			before, _, ok := strings.Cut(rest, "\"")
			if !ok {
				break
			}
			br := before
			szStr, offStr, ok := strings.Cut(br, "@")
			if !ok {
				break
			}
			sz, _ := strconv.ParseInt(szStr, 10, 64)
			off, _ := strconv.ParseInt(offStr, 10, 64)
			if off >= 0 && off+sz <= int64(len(mediaData)) {
				initSeg = mediaData[off : off+sz]
			}
		}
		break
	}
	if len(initSeg) == 0 {
		t.Fatal("could not parse init segment from EXT-X-MAP")
	}

	var mismatches int
	var segments int
	for i := range lines {
		_, ok := parseEXTINF(lines[i])
		if !ok {
			continue
		}
		off, sz, found := findByteRange(lines, i+1)
		if !found {
			continue
		}
		if off < 0 || off+sz > int64(len(mediaData)) {
			continue
		}
		chunk := mediaData[off : off+sz]

		trunCount, err := fmp4VideoSampleCount(chunk)
		if err != nil {
			t.Logf("segment %d: skip fmp4 parse: %v",
				segments, err)
			segments++
			continue
		}

		// Write init segment + fragment to a temp file so
		// ffprobe can decode it standalone.
		segPath := filepath.Join(dir,
			"seg"+strconv.Itoa(segments)+".mp4")
		segData := make([]byte, len(initSeg)+len(chunk))
		copy(segData, initSeg)
		copy(segData[len(initSeg):], chunk)
		if err := os.WriteFile(segPath, segData, 0o644); err != nil {
			t.Fatalf("write segment: %v", err)
		}

		// Count decoded video frames with ffprobe.
		probeCmd := newCmd(ctx, "ffprobe",
			"-v", "error",
			"-select_streams", "v",
			"-count_frames",
			"-show_entries", "stream=nb_read_frames",
			"-of", "csv=p=0",
			segPath,
		)
		probeOut, err := probeCmd.CombinedOutput()
		if err != nil {
			t.Logf("segment %d: ffprobe error: %v\n%s",
				segments, err, probeOut)
			segments++
			continue
		}
		// Take the last non-empty line — Docker platform
		// warnings may precede the actual output.
		outLines := strings.Split(
			strings.TrimSpace(string(probeOut)), "\n")
		lastLine := outLines[len(outLines)-1]
		decoded, err := strconv.Atoi(
			strings.TrimSpace(lastLine))
		if err != nil {
			t.Logf("segment %d: parse ffprobe output %q: %v",
				segments, string(probeOut), err)
			segments++
			continue
		}

		if trunCount != decoded {
			t.Errorf("segment %d: open GOP detected: "+
				"trun=%d decoded=%d (delta=%d)",
				segments, trunCount, decoded,
				trunCount-decoded)
			mismatches++
		}
		segments++
	}
	if segments == 0 {
		t.Fatal("no segments found in playlist")
	}
	t.Logf("checked %d segments: %d mismatches", segments, mismatches)
}

// fmp4VideoSampleCount parses an fMP4 fragment (moof+mdat) and
// returns the video track's trun sample count.
func fmp4VideoSampleCount(fragment []byte) (int, error) {
	moofData, err := findBox(fragment, "moof")
	if err != nil {
		return 0, fmt.Errorf("no moof: %w", err)
	}
	trafs := findAllBoxes(moofData, "traf")
	if len(trafs) == 0 {
		return 0, fmt.Errorf("no traf in moof")
	}
	videoTraf := findVideoTraf(trafs)
	trunData, err := findBox(videoTraf, "trun")
	if err != nil {
		return 0, fmt.Errorf("no trun: %w", err)
	}
	if len(trunData) < 8 {
		return 0, fmt.Errorf("trun too short")
	}
	count := binary.BigEndian.Uint32(trunData[4:8])
	return int(count), nil
}
