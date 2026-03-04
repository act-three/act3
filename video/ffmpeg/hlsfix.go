package ffmpeg

import (
	"encoding/binary"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// fixEXTINF corrects EXTINF durations in an HLS playlist by parsing
// the actual fMP4 segment data. ffmpeg's HLS muxer computes EXTINF
// from raw encoder output packet timestamps, which can differ from
// the actual fMP4 segment PTS spans when the video encoder introduces
// a B-frame delay (e.g. libx265 medium preset with VFR telecine
// input). This manifests as ~117ms "Playlist vs segment duration
// mismatch" errors in mediastreamvalidator.
//
// The fix: for each segment byte range in the playlist, parse the
// fMP4 moof/trun boxes to find the video track's actual presentation
// time span (min PTS to max PTS+duration), then replace the EXTINF
// value with the correct duration.
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

	for i := 0; i < len(lines); i++ {
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
	// actual video PTS span.
	var maxDur float64
	for _, seg := range segs {
		if seg.offset < 0 || seg.offset+seg.size > int64(len(media)) {
			if seg.origDur > maxDur {
				maxDur = seg.origDur
			}
			continue
		}
		chunk := media[seg.offset : seg.offset+seg.size]
		dur, err := fmp4VideoDuration(chunk)
		if err != nil {
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

// fmp4VideoDuration parses an fMP4 fragment (moof+mdat) and returns
// the video track's presentation duration in seconds. It finds the
// first video traf (identified by having sample composition time
// offsets or by being track ID 1, the typical video track), reads
// tfdt for the base decode time, and scans trun samples to compute
// the PTS span: (max_sample_PTS + duration) - min_sample_PTS.
//
// The timescale is read from the mdhd box if a moov is present in
// the data; otherwise it's inferred from the trun sample durations
// and the expected ~24fps telecine frame rate. In practice the moov
// is NOT in the segment chunk (it's at the start of the file), so
// we need the caller to handle timescale. However, since we're
// computing a ratio (actual_duration / timescale) and the EXTINF
// is also in seconds, we can compute the duration in timescale
// ticks and convert using the timescale from tfdt+trun data
// combined with the original EXTINF as a reference.
//
// Actually, the simplest correct approach: compute the duration in
// ticks from the trun, then find the timescale by looking at the
// sidx box that precedes the moof. The sidx box contains the
// timescale and the segment duration in that timescale — but the
// sidx duration has the same bug as EXTINF (ffmpeg writes it from
// the same source). So instead we scan the full media for the moov
// to get the timescale.
//
// Revised approach: the caller passes the full media file, and we
// accept a chunk range. But for simplicity, this function accepts
// just the fragment and a timescale parameter.
func fmp4VideoDuration(fragment []byte) (float64, error) {
	// We need the timescale. It's in the moov/trak/mdia/mdhd box,
	// but the fragment doesn't contain moov. We'll infer it:
	// find the trun, compute total decode duration in ticks, and
	// the PTS span in ticks. Then we need a timescale to convert
	// to seconds.
	//
	// Alternative: look for a sidx box before the moof. The sidx
	// contains timescale and reference_duration. While the
	// reference_duration may be wrong (same bug), the timescale
	// field is correct.
	timescale, found := parseSIDXTimescale(fragment)
	if !found {
		return 0, fmt.Errorf("no sidx box found in fragment")
	}

	// Find the moof box.
	moofData, err := findBox(fragment, "moof")
	if err != nil {
		return 0, fmt.Errorf("no moof: %w", err)
	}

	// Find the first video traf. We identify the video track by
	// looking for a trun with composition time offsets (flag 0x800)
	// — audio trun boxes typically don't have CTS offsets.
	// If no traf has CTS offsets, fall back to the first traf.
	trafs := findAllBoxes(moofData, "traf")
	if len(trafs) == 0 {
		return 0, fmt.Errorf("no traf in moof")
	}

	var videoTraf []byte
	for _, traf := range trafs {
		trunData, err := findBox(traf, "trun")
		if err != nil {
			continue
		}
		if len(trunData) < 4 {
			continue
		}
		flags := uint32(trunData[1])<<16 |
			uint32(trunData[2])<<8 |
			uint32(trunData[3])
		if flags&0x800 != 0 { // has CTS offsets → video
			videoTraf = traf
			break
		}
	}
	if videoTraf == nil {
		// Fall back to first traf.
		videoTraf = trafs[0]
	}

	// Parse tfdt for base_decode_time.
	tfdtData, err := findBox(videoTraf, "tfdt")
	if err != nil {
		return 0, fmt.Errorf("no tfdt: %w", err)
	}
	_, baseDTS := parseTFDT(tfdtData)

	// Parse trun for sample durations and CTS offsets.
	trunData, err := findBox(videoTraf, "trun")
	if err != nil {
		return 0, fmt.Errorf("no trun: %w", err)
	}
	minPTS, maxPTSPlusDur, err := parseTRUN(trunData, baseDTS)
	if err != nil {
		return 0, err
	}

	span := maxPTSPlusDur - minPTS
	if span <= 0 {
		return 0, fmt.Errorf("non-positive PTS span: %d", span)
	}

	return float64(span) / float64(timescale), nil
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

// parseTFDT reads a tfdt (track fragment decode time) box and
// returns the base_decode_time. The box payload layout:
//
//	[version:1][flags:3][base_media_decode_time:4 or 8]
func parseTFDT(data []byte) (version uint8, baseDTS int64) {
	if len(data) < 8 {
		return 0, 0
	}
	version = data[0]
	if version == 1 && len(data) >= 12 {
		return version, int64(binary.BigEndian.Uint64(data[4:12]))
	}
	return version, int64(binary.BigEndian.Uint32(data[4:8]))
}

// parseTRUN reads a trun (track run) box and computes the min PTS
// and max PTS+duration across all samples. Returns these as tick
// values in the track's timescale.
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
func parseTRUN(data []byte, baseDTS int64) (minPTS, maxPTSPlusDur int64, err error) {
	if len(data) < 8 {
		return 0, 0, fmt.Errorf("trun too short")
	}

	version := data[0]
	flags := uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	sampleCount := int(binary.BigEndian.Uint32(data[4:8]))

	if sampleCount == 0 {
		return 0, 0, fmt.Errorf("trun has 0 samples")
	}

	hasDuration := flags&0x100 != 0
	hasSize := flags&0x200 != 0
	hasFlags := flags&0x400 != 0
	hasCTSOffset := flags&0x800 != 0

	pos := 8
	if flags&0x001 != 0 { // data_offset
		pos += 4
	}
	if flags&0x004 != 0 { // first_sample_flags
		pos += 4
	}

	// Compute per-sample record size.
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
		return 0, 0, fmt.Errorf("trun data too short: need %d, have %d",
			needed, len(data))
	}

	minPTS = math.MaxInt64
	maxPTSPlusDur = math.MinInt64
	dts := baseDTS

	for i := 0; i < sampleCount; i++ {
		var duration uint32
		var ctsOffset int64

		if hasDuration {
			duration = binary.BigEndian.Uint32(data[pos:])
			pos += 4
		}
		if hasSize {
			pos += 4
		}
		if hasFlags {
			pos += 4
		}
		if hasCTSOffset {
			if version == 0 {
				ctsOffset = int64(binary.BigEndian.Uint32(data[pos:]))
			} else {
				// Signed 32-bit composition offset (version >= 1).
				ctsOffset = int64(int32(binary.BigEndian.Uint32(data[pos:])))
			}
			pos += 4
		}

		pts := dts + ctsOffset
		end := pts + int64(duration)

		if pts < minPTS {
			minPTS = pts
		}
		if end > maxPTSPlusDur {
			maxPTSPlusDur = end
		}

		dts += int64(duration)
	}

	return minPTS, maxPTSPlusDur, nil
}
