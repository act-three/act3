package ffmpeg

import (
	"encoding/binary"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// fixEXTINF corrects EXTINF durations in an HLS playlist by
// parsing the actual fMP4 segment data. ffmpeg's HLS muxer
// computes EXTINF from raw encoder output packet timestamps,
// which include a B-frame pipeline delay that inflates the first
// segment's duration by ~117ms (libx265 medium preset with VFR
// telecine input). This causes "Playlist vs segment duration
// mismatch" errors in mediastreamvalidator.
//
// Per RFC 8216, EXTINF for fMP4 segments is the sum of sample
// durations in the track run. For each segment byte range, we
// parse the moof's video traf to sum the sample durations from
// the trun (or the tfhd default_sample_duration when per-sample
// durations are absent), then divide by the track timescale from
// the sidx box.
//
// If parsing fails for any segment, the original EXTINF is kept —
// a slightly wrong duration is better than a missing playlist.
func fixEXTINF(playlist string, media []byte) string {
	// Parse the playlist to find segment byte ranges and their
	// EXTINF line indices.
	type segment struct {
		extinfIdx int     // index into lines
		origDur   float64 // original EXTINF value
		offset    int64   // byte offset into media
		size      int64   // byte count
	}

	lines := strings.Split(playlist, "\n")
	var segs []segment

	for i := range lines {
		dur, ok := parseEXTINF(lines[i])
		if !ok {
			continue
		}
		// The next non-empty lines should be an optional
		// EXT-X-BYTERANGE and then the media URI.
		off, sz, found := findByteRange(lines, i+1)
		if !found {
			continue
		}
		segs = append(segs, segment{
			extinfIdx: i,
			origDur:   dur,
			offset:    off,
			size:      sz,
		})
	}

	if len(segs) == 0 {
		return playlist
	}

	// For each segment, parse the fMP4 fragment to compute the
	// video track's total sample duration (DTS span).
	var maxDur float64
	for _, seg := range segs {
		// Written as subtraction so seg.offset+seg.size cannot
		// overflow int64 for near-2^63 values from a crafted
		// EXT-X-BYTERANGE line.
		if seg.offset < 0 || seg.size < 0 ||
			seg.offset > int64(len(media))-seg.size {
			if seg.origDur > maxDur {
				maxDur = seg.origDur
			}
			continue
		}
		chunk := media[seg.offset : seg.offset+seg.size]
		dur, err := fmp4VideoDuration(chunk)
		if err != nil || math.IsNaN(dur) || math.IsInf(dur, 0) {
			if seg.origDur > maxDur {
				maxDur = seg.origDur
			}
			continue
		}
		// Replace the EXTINF line with the corrected duration.
		lines[seg.extinfIdx] = fmt.Sprintf("#EXTINF:%f,", dur)
		if dur > maxDur {
			maxDur = dur
		}
	}

	// Update EXT-X-TARGETDURATION to be the rounded-up integer of
	// the longest corrected segment. Per the HLS spec, each EXTINF
	// rounded to the nearest integer must be ≤ TARGETDURATION.
	if maxDur > 0 {
		target := int(math.Ceil(maxDur))
		for i, line := range lines {
			if strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") {
				lines[i] = fmt.Sprintf("#EXT-X-TARGETDURATION:%d", target)
				break
			}
		}
	}

	return strings.Join(lines, "\n")
}

// parseEXTINF extracts the duration from an #EXTINF: line.
var extinfRe = regexp.MustCompile(`^#EXTINF:([\d.]+),`)

func parseEXTINF(line string) (float64, bool) {
	m := extinfRe.FindStringSubmatch(line)
	if m == nil {
		return 0, false
	}
	d, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return d, true
}

// findByteRange scans lines starting at idx for an EXT-X-BYTERANGE
// directive and returns (offset, size, true). Returns false if no
// byte range is found before the next EXTINF or end of playlist.
var byterangeRe = regexp.MustCompile(
	`^#EXT-X-BYTERANGE:(\d+)@(\d+)`)

func findByteRange(lines []string, idx int) (offset, size int64, ok bool) {
	for i := idx; i < len(lines) && i < idx+3; i++ {
		m := byterangeRe.FindStringSubmatch(lines[i])
		if m != nil {
			sz, _ := strconv.ParseInt(m[1], 10, 64)
			off, _ := strconv.ParseInt(m[2], 10, 64)
			return off, sz, true
		}
		// Stop if we hit another EXTINF or EXT-X-ENDLIST.
		if strings.HasPrefix(lines[i], "#EXTINF:") ||
			strings.HasPrefix(lines[i], "#EXT-X-ENDLIST") {
			break
		}
	}
	return 0, 0, false
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

// fmp4VideoDuration parses an fMP4 fragment (moof+mdat) and
// returns the video track's decode duration in seconds: the sum
// of all sample durations divided by the track timescale.
//
// Sample durations come from the trun box when the
// sample-duration-present flag (0x100) is set, otherwise from
// tfhd.default_sample_duration. The timescale comes from the
// sidx box that precedes the moof.
func fmp4VideoDuration(fragment []byte) (float64, error) {
	timescale, found := parseSIDXTimescale(fragment)
	if !found {
		return 0, fmt.Errorf("no sidx box found in fragment")
	}

	moofData, err := findBox(fragment, "moof")
	if err != nil {
		return 0, fmt.Errorf("no moof: %w", err)
	}

	trafs := findAllBoxes(moofData, "traf")
	if len(trafs) == 0 {
		return 0, fmt.Errorf("no traf in moof")
	}

	videoTraf := findVideoTraf(trafs)

	// Read default_sample_duration from tfhd (used when the
	// trun omits per-sample durations).
	defaultDur := parseTFHDDefaultDuration(videoTraf)

	trunData, err := findBox(videoTraf, "trun")
	if err != nil {
		return 0, fmt.Errorf("no trun: %w", err)
	}
	totalTicks, err := trunTotalDuration(trunData, defaultDur)
	if err != nil {
		return 0, err
	}
	if totalTicks <= 0 {
		return 0, fmt.Errorf("non-positive duration: %d", totalTicks)
	}

	return float64(totalTicks) / float64(timescale), nil
}

// findVideoTraf picks the video traf from a list of traf
// payloads. It prefers a traf whose trun has CTS offsets
// (flag 0x800 — video tracks have composition offsets, audio
// typically does not). Failing that, it picks the traf with
// tfhd track_id=1 (the conventional video track). As a last
// resort it returns the first traf.
func findVideoTraf(trafs [][]byte) []byte {
	// Prefer traf with CTS offsets in trun.
	for _, traf := range trafs {
		trunData, err := findBox(traf, "trun")
		if err != nil || len(trunData) < 4 {
			continue
		}
		flags := uint32(trunData[1])<<16 |
			uint32(trunData[2])<<8 |
			uint32(trunData[3])
		if flags&0x800 != 0 {
			return traf
		}
	}
	// Fall back to track_id 1.
	for _, traf := range trafs {
		tfhdData, err := findBox(traf, "tfhd")
		if err != nil || len(tfhdData) < 8 {
			continue
		}
		trackID := binary.BigEndian.Uint32(tfhdData[4:8])
		if trackID == 1 {
			return traf
		}
	}
	return trafs[0]
}

// parseSIDXTimescale finds the first sidx box in data and returns
// its timescale field. The sidx (segment index) box layout:
//
//	[size:4][type:4][version:1][flags:3]
//	[reference_ID:4][timescale:4]...
func parseSIDXTimescale(data []byte) (uint32, bool) {
	offset := 0
	for offset+8 <= len(data) {
		size := int(binary.BigEndian.Uint32(data[offset:]))
		if size < 8 || offset+size > len(data) {
			break
		}
		boxType := string(data[offset+4 : offset+8])
		if boxType == "sidx" && size >= 20 {
			// version(1) + flags(3) + reference_ID(4) + timescale(4)
			ts := binary.BigEndian.Uint32(data[offset+16 : offset+20])
			if ts == 0 {
				// Zero timescale would yield +Inf durations; treat
				// as unparseable and let fixEXTINF fall back to the
				// original EXTINF.
				return 0, false
			}
			return ts, true
		}
		// Stop if we hit moof — sidx comes before moof.
		if boxType == "moof" {
			break
		}
		offset += size
	}
	return 0, false
}

// findBox searches for a box of the given type inside a container
// box's payload. Returns the box's payload (after the 8-byte header).
func findBox(container []byte, boxType string) ([]byte, error) {
	offset := 0
	for offset+8 <= len(container) {
		size := int(binary.BigEndian.Uint32(container[offset:]))
		if size < 8 {
			break
		}
		if offset+size > len(container) {
			break
		}
		bt := string(container[offset+4 : offset+8])
		if bt == boxType {
			return container[offset+8 : offset+size], nil
		}
		offset += size
	}
	return nil, fmt.Errorf("box %q not found", boxType)
}

// findAllBoxes returns all boxes of the given type inside a
// container box's payload.
func findAllBoxes(container []byte, boxType string) [][]byte {
	var result [][]byte
	offset := 0
	for offset+8 <= len(container) {
		size := int(binary.BigEndian.Uint32(container[offset:]))
		if size < 8 || offset+size > len(container) {
			break
		}
		bt := string(container[offset+4 : offset+8])
		if bt == boxType {
			result = append(result, container[offset+8:offset+size])
		}
		offset += size
	}
	return result
}

// parseTFHDDefaultDuration extracts default_sample_duration from
// a tfhd box inside the given traf payload. Returns 0 if the
// field is absent.
//
// tfhd payload layout:
//
//	[version:1][flags:3][track_ID:4]
//	[base_data_offset:8]?          (if flags & 0x000001)
//	[sample_description_index:4]?  (if flags & 0x000002)
//	[default_sample_duration:4]?   (if flags & 0x000008)
//	[default_sample_size:4]?       (if flags & 0x000010)
//	[default_sample_flags:4]?      (if flags & 0x000020)
func parseTFHDDefaultDuration(traf []byte) uint32 {
	tfhdData, err := findBox(traf, "tfhd")
	if err != nil || len(tfhdData) < 8 {
		return 0
	}
	flags := uint32(tfhdData[1])<<16 |
		uint32(tfhdData[2])<<8 |
		uint32(tfhdData[3])
	if flags&0x08 == 0 {
		return 0 // no default_sample_duration
	}
	pos := 8 // past version(1)+flags(3)+track_ID(4)
	if flags&0x01 != 0 {
		pos += 8 // base_data_offset
	}
	if flags&0x02 != 0 {
		pos += 4 // sample_description_index
	}
	if pos+4 > len(tfhdData) {
		return 0
	}
	return binary.BigEndian.Uint32(tfhdData[pos : pos+4])
}

// trunTotalDuration sums sample durations from a trun box and
// returns the total in timescale ticks. When the trun does not
// carry per-sample durations (flag 0x100 absent), defaultDur is
// used for every sample.
//
// trun payload layout:
//
//	[version:1][flags:3][sample_count:4]
//	[data_offset:4]?            (if flags & 0x001)
//	[first_sample_flags:4]?     (if flags & 0x004)
//	per sample:
//	  [sample_duration:4]?      (if flags & 0x100)
//	  [sample_size:4]?          (if flags & 0x200)
//	  [sample_flags:4]?         (if flags & 0x400)
//	  [sample_composition_time_offset:4]? (if flags & 0x800)
func trunTotalDuration(data []byte, defaultDur uint32) (int64, error) {
	if len(data) < 8 {
		return 0, fmt.Errorf("trun too short")
	}

	flags := uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	sampleCount := int(binary.BigEndian.Uint32(data[4:8]))

	if sampleCount == 0 {
		return 0, fmt.Errorf("trun has 0 samples")
	}

	// Cap sampleCount to protect against crafted truns that omit
	// all per-sample fields (recordSize == 0) but rely on
	// tfhd.default_sample_duration — in that case the needed-bytes
	// check below collapses to a tautology and the loop would
	// otherwise run up to 2^32 iterations.  A real HLS segment has
	// at most a few thousand samples.
	const maxSampleCount = 1 << 20
	if sampleCount > maxSampleCount {
		return 0, fmt.Errorf("trun sample_count %d exceeds cap %d",
			sampleCount, maxSampleCount)
	}

	hasDuration := flags&0x100 != 0
	hasSize := flags&0x200 != 0
	hasFlags := flags&0x400 != 0
	hasCTSOffset := flags&0x800 != 0

	// When the trun omits per-sample durations, use the tfhd
	// default. If neither source provides a duration, the
	// segment duration is unknowable.
	if !hasDuration && defaultDur == 0 {
		return 0, fmt.Errorf("trun has no per-sample durations " +
			"and no default_sample_duration")
	}

	pos := 8
	if flags&0x001 != 0 {
		pos += 4 // data_offset
	}
	if flags&0x004 != 0 {
		pos += 4 // first_sample_flags
	}

	recordSize := 0
	if hasDuration {
		recordSize += 4
	}
	if hasSize {
		recordSize += 4
	}
	if hasFlags {
		recordSize += 4
	}
	if hasCTSOffset {
		recordSize += 4
	}

	needed := pos + sampleCount*recordSize
	if needed > len(data) {
		return 0, fmt.Errorf("trun data too short: need %d, have %d",
			needed, len(data))
	}

	var total int64
	for range sampleCount {
		dur := defaultDur
		if hasDuration {
			dur = binary.BigEndian.Uint32(data[pos:])
			pos += 4
		}
		if hasSize {
			pos += 4
		}
		if hasFlags {
			pos += 4
		}
		if hasCTSOffset {
			pos += 4
		}
		total += int64(dur)
	}

	return total, nil
}
