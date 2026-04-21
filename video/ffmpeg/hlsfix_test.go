package ffmpeg

import (
	"encoding/binary"
	"fmt"
	"math"
	"testing"
	"time"
)

// ---------------------------------------------------------------
// fMP4 box construction helpers
// ---------------------------------------------------------------

// box builds a top-level or nested ISO BMFF box.
func box(typ string, payload []byte) []byte {
	b := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(b, uint32(len(b)))
	copy(b[4:8], typ)
	copy(b[8:], payload)
	return b
}

// fullbox builds a full-box (version + flags prefix).
func fullbox(typ string, version uint8, flags uint32, payload []byte) []byte {
	p := make([]byte, 4+len(payload))
	p[0] = version
	p[1] = byte(flags >> 16)
	p[2] = byte(flags >> 8)
	p[3] = byte(flags)
	copy(p[4:], payload)
	return box(typ, p)
}

func put32(v uint32) []byte { b := make([]byte, 4); binary.BigEndian.PutUint32(b, v); return b }

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// buildSIDX builds a minimal sidx box with the given timescale.
func buildSIDX(refID, timescale uint32) []byte {
	// sidx v0: ref_id(4) + timescale(4) + earliest_pres(4) +
	//          first_offset(4) + reserved(2) + ref_count(2) +
	//          one reference entry(12)
	payload := concat(
		put32(refID),
		put32(timescale),
		put32(0),           // earliest_presentation_time
		put32(0),           // first_offset
		[]byte{0, 0, 0, 1}, // reserved(2) + reference_count(2)=1
		put32(0),           // reference (type+size)
		put32(0),           // subsegment_duration
		put32(0),           // SAP info
	)
	return fullbox("sidx", 0, 0, payload)
}

// buildTFHD builds a tfhd box.
func buildTFHD(trackID uint32, defaultDur *uint32) []byte {
	flags := uint32(0x020000) // default-base-is-moof
	var extra []byte
	if defaultDur != nil {
		flags |= 0x08 // default-sample-duration-present
		extra = put32(*defaultDur)
	}
	return fullbox("tfhd", 0, flags, concat(put32(trackID), extra))
}

// buildTFDT builds a tfdt v0 box.
func buildTFDT(baseDTS uint32) []byte {
	return fullbox("tfdt", 0, 0, put32(baseDTS))
}

// trunSample holds per-sample fields for trun construction.
type trunSample struct {
	Duration uint32
	Size     uint32
	CTS      int32
}

// buildTRUN builds a trun box. If hasDur is false, durations are
// omitted from the per-sample records (the caller should provide
// a default via tfhd). CTS offsets are included when hasCTS is
// true; signed CTS requires version=1.
func buildTRUN(version uint8, hasDur, hasCTS bool, samples []trunSample) []byte {
	flags := uint32(0x001) // data_offset always present
	if hasDur {
		flags |= 0x100
	}
	flags |= 0x200 // sample_size always present
	if hasCTS {
		flags |= 0x800
	}

	var payload []byte
	payload = append(payload, put32(uint32(len(samples)))...) // sample_count
	payload = append(payload, put32(0)...)                    // data_offset

	for _, s := range samples {
		if hasDur {
			payload = append(payload, put32(s.Duration)...)
		}
		payload = append(payload, put32(s.Size)...) // sample_size
		if hasCTS {
			payload = append(payload, put32(uint32(s.CTS))...)
		}
	}

	return fullbox("trun", version, flags, payload)
}

// buildFragment assembles sidx + moof(traf(tfhd,tfdt,trun)) +
// a minimal mdat.
func buildFragment(
	timescale uint32,
	trackID uint32,
	baseDTS uint32,
	defaultDur *uint32,
	trunVersion uint8,
	hasDur, hasCTS bool,
	samples []trunSample,
) []byte {
	sidx := buildSIDX(1, timescale)
	traf := box("traf", concat(
		buildTFHD(trackID, defaultDur),
		buildTFDT(baseDTS),
		buildTRUN(trunVersion, hasDur, hasCTS, samples),
	))
	moof := box("moof", traf)
	mdat := box("mdat", []byte{0}) // minimal mdat
	return concat(sidx, moof, mdat)
}

// ---------------------------------------------------------------
// Tests
// ---------------------------------------------------------------

// TestFmp4VideoDuration_DefaultSampleDuration verifies that
// fmp4VideoDuration uses tfhd.default_sample_duration when the
// trun has no per-sample durations. This reproduces the bug where
// CFR renditions (e.g. fps=30 output) get ~0.2s EXTINF instead of
// the correct ~13.7s, because parseTRUN treated every sample as
// having duration 0.
func TestFmp4VideoDuration_DefaultSampleDuration(t *testing.T) {
	const (
		timescale   = 15360 // 30fps track
		frameDur    = 512   // 15360/30
		sampleCount = 10
	)
	defaultDur := uint32(frameDur)

	// Build samples with CTS offsets but no per-sample durations.
	// CTS offsets simulate B-frame reordering (range 0–3072).
	samples := make([]trunSample, sampleCount)
	ctsPattern := []int32{1024, 3072, 1536, 0, 512,
		1024, 3072, 1536, 0, 512}
	for i := range samples {
		samples[i] = trunSample{Size: 100, CTS: ctsPattern[i]}
	}

	fragment := buildFragment(
		timescale, 1, 0, &defaultDur,
		0,     // trun version 0 (unsigned CTS)
		false, // no per-sample durations
		true,  // has CTS offsets
		samples,
	)

	got, err := fmp4VideoDuration(fragment)
	if err != nil {
		t.Fatalf("fmp4VideoDuration: %v", err)
	}

	// Correct duration: 10 samples × 512 ticks / 15360 = 0.333333s
	want := float64(sampleCount*frameDur) / float64(timescale)

	// The bug: without reading default_sample_duration, the code
	// computes (max CTS - min CTS) / timescale = 3072/15360 = 0.2s
	buggy := float64(3072) / float64(timescale)

	if math.Abs(got-buggy) < 0.001 {
		t.Errorf("fmp4VideoDuration returned buggy PTS-range "+
			"duration %fs (max_CTS-min_CTS); want DTS span %fs",
			got, want)
	}
	if math.Abs(got-want) > 0.001 {
		t.Errorf("fmp4VideoDuration = %f, want %f", got, want)
	}
}

// TestFmp4VideoDuration_DTSSpanNotPTSSpan verifies that
// fmp4VideoDuration returns the DTS span (sum of sample durations)
// rather than the PTS span. Per RFC 8216, EXTINF for fMP4 is the
// sum of sample durations. Using PTS span instead causes cumulative
// drift at segment boundaries (~70s over a 45-minute video).
func TestFmp4VideoDuration_DTSSpanNotPTSSpan(t *testing.T) {
	const timescale = 90000

	// IBPB pattern where PTS span < DTS span due to B-frame
	// reordering at the segment boundary:
	//
	//   Sample  DTS      CTS     PTS      PTS+dur
	//   I       0        +3003   3003     6006
	//   B       3003     -3003   0        3003
	//   P       6006     +3003   9009     12012
	//   B       9009     -3003   6006     9009
	//
	//   DTS span = 4 × 3003 = 12012
	//   PTS span = max(12012) - min(0) = 12012   … equal here
	//
	// But if the last sample is a B-frame whose PTS+dur doesn't
	// reach the end of the DTS range, PTS span < DTS span:
	//
	//   Sample  DTS      CTS     PTS      PTS+dur
	//   I       0        +3003   3003     6006
	//   P       3003     +3003   6006     9009
	//   B       6006     -3003   3003     6006
	//
	//   DTS span = 3 × 3003 = 9009
	//   PTS span = max(9009) - min(3003) = 6006
	samples := []trunSample{
		{Duration: 3003, Size: 500, CTS: +3003}, // I
		{Duration: 3003, Size: 200, CTS: +3003}, // P
		{Duration: 3003, Size: 100, CTS: -3003}, // B
	}

	fragment := buildFragment(
		timescale, 1, 0, nil,
		1,    // version 1 for signed CTS
		true, // per-sample durations
		true, // CTS offsets
		samples,
	)

	got, err := fmp4VideoDuration(fragment)
	if err != nil {
		t.Fatalf("fmp4VideoDuration: %v", err)
	}

	dtsDur := float64(3*3003) / float64(timescale)    // 0.100100
	ptsDur := float64(9009-3003) / float64(timescale) // 0.066733

	if math.Abs(got-ptsDur) < 0.0001 {
		t.Errorf("fmp4VideoDuration returned PTS span %fs; "+
			"want DTS span %fs (sum of sample durations per "+
			"RFC 8216)", got, dtsDur)
	}
	if math.Abs(got-dtsDur) > 0.0001 {
		t.Errorf("fmp4VideoDuration = %f, want %f", got, dtsDur)
	}
}

// TestFixEXTINF_Corrects verifies the end-to-end fixEXTINF function
// replaces EXTINF values with the DTS span computed from fMP4 data.
func TestFixEXTINF_Corrects(t *testing.T) {
	const timescale = 1000
	defaultDur := uint32(100)

	// 20 samples × 100 ticks / 1000 = 2.0s
	samples := make([]trunSample, 20)
	for i := range samples {
		samples[i] = trunSample{Size: 50, CTS: int32(i % 3 * 100)}
	}

	fragment := buildFragment(
		timescale, 1, 0, &defaultDur,
		0, false, true, samples,
	)

	// Build a playlist whose single segment covers the fragment.
	playlist := fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:7
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:VOD
#EXT-X-MAP:URI="media.mp4",BYTERANGE="100@0"
#EXTINF:0.200000,
#EXT-X-BYTERANGE:%d@0
media.mp4
#EXT-X-ENDLIST`, len(fragment))

	got := fixEXTINF(playlist, fragment)

	// Should contain the corrected duration (2.0s), not 0.2s.
	if dur, ok := parseEXTINF(
		findEXTINFLine(got)); !ok || math.Abs(dur-2.0) > 0.01 {
		t.Errorf("fixEXTINF produced EXTINF=%f, want ~2.0", dur)
	}
}

func findEXTINFLine(playlist string) string {
	for _, line := range splitLines(playlist) {
		if _, ok := parseEXTINF(line); ok {
			return line
		}
	}
	return ""
}

func splitLines(s string) []string {
	var lines []string
	for len(s) > 0 {
		i := 0
		for i < len(s) && s[i] != '\n' {
			i++
		}
		lines = append(lines, s[:i])
		if i < len(s) {
			i++ // skip \n
		}
		s = s[i:]
	}
	return lines
}

// TestTrunTotalDuration_SampleCountCap verifies that a crafted trun
// with an enormous sample_count and no per-sample record fields
// (recordSize == 0) is rejected quickly.  Without the cap the loop
// would run up to 2^32 iterations, burning ~3–5 CPU seconds per
// fragment — a CPU-DoS vector through the ingest path.
func TestTrunTotalDuration_SampleCountCap(t *testing.T) {
	// trun payload: sample_count(4).  flags=0 so no data_offset,
	// no first_sample_flags, no per-sample fields → recordSize==0.
	payload := put32(0xFFFFFFFF)
	trun := fullbox("trun", 0, 0, payload)

	start := time.Now()
	_, err := trunTotalDuration(trun[8:], 1)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("trunTotalDuration accepted 0xFFFFFFFF sample_count")
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("trunTotalDuration took %v on crafted trun; "+
			"want <100ms", elapsed)
	}
}
