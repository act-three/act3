package ffmpeg

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

// ProbeKeyframes returns the display-order frame indices of every
// video keyframe in r. Indices are 0-based; the first keyframe is
// always present and starts segment 0.
//
// The implementation enumerates video stream packets via ffprobe
// (each video packet carries one frame) and counts them in stream
// order. For closed-GOP encodings — which all standard HLS inputs
// use — a keyframe's stream-order packet index equals its
// display-order frame index, so the returned indices line up with
// the encoder's `n` variable in `-force_key_frames "expr:..."`.
//
// HLS segment boundaries can only fall on keyframes (each fMP4
// segment must be independently decodable), so the keyframe list is
// what determines achievable cut points for any rendition produced
// from this source — both stream-copy and re-encode.
func ProbeKeyframes(ctx context.Context, r *os.File, format string) ([]int64, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	c := newCmd(ctx, "ffprobe",
		"-v", "error",
		"-protocol_whitelist", "file",
		"-f", format,
		"-select_streams", "v:0",
		"-show_packets",
		"-show_entries", "packet=flags",
		"-of", "csv=p=0",
		"/dev/stdin",
	)
	c.Stdin = r
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return nil, errors.Join(err, errors.New(stderr.String()))
	}
	var out []int64
	var idx int64
	for line := range strings.SplitSeq(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "K") {
			out = append(out, idx)
		}
		idx++
	}
	return out, nil
}

// SegmentBoundaries picks a subset of source keyframes to use as HLS
// segment cuts, with at least minFrames between consecutive
// boundaries. Inputs and outputs are display-order frame indices.
// The first keyframe (start of segment 0) is not included — only
// the interior cut points between segments.
//
// The selection algorithm matches what ffmpeg's HLS muxer does on
// stream-copy: walk keyframes in order, take the first one whose
// distance from the previous cut is at least minFrames. Re-encodes
// can be forced (via -force_key_frames "expr:eq(n,...)") to put
// keyframes at exactly the returned frame indices so every variant
// cuts at the same boundaries.
func SegmentBoundaries(keyframes []int64, minFrames int64) []int64 {
	if len(keyframes) <= 1 {
		return nil
	}
	var out []int64
	last := keyframes[0]
	for _, k := range keyframes[1:] {
		if k-last >= minFrames {
			out = append(out, k)
			last = k
		}
	}
	return out
}

// MinFramesPerSegment returns the minimum integer frame count that
// covers target at the given frame rate. Returns 0 if fps is
// degenerate.
func MinFramesPerSegment(fps FrameRate, target time.Duration) int64 {
	if fps.Num <= 0 || fps.Den <= 0 {
		return 0
	}
	num := int64(target) * int64(fps.Num)
	den := int64(fps.Den) * int64(time.Second)
	// ceil(num/den)
	return (num + den - 1) / den
}

// UniformSegmentBoundaries returns synthetic HLS segment cut points
// at minFrames intervals up to duration, expressed as display-order
// frame indices. Used when every rendition for a source is a
// re-encode: with no stream-copy variant in the mix, segment cuts
// don't have to land on source keyframes, so we pick a uniform grid
// the encoder can hit exactly.
//
// Like SegmentBoundaries, the implicit cut at frame 0 is omitted —
// only interior boundaries are returned.
func UniformSegmentBoundaries(fps FrameRate, duration time.Duration, minFrames int64) []int64 {
	if minFrames <= 0 || fps.Num <= 0 || fps.Den <= 0 {
		return nil
	}
	num := int64(duration) * int64(fps.Num)
	den := int64(fps.Den) * int64(time.Second)
	totalFrames := num / den
	var out []int64
	for k := minFrames; k < totalFrames; k += minFrames {
		out = append(out, k)
	}
	return out
}

// MaxKeyframeGap returns the largest distance (in display-order
// frames) between consecutive entries in keyframes. Returns 0 for
// fewer than two entries.
func MaxKeyframeGap(keyframes []int64) int64 {
	var max int64
	for i := 1; i < len(keyframes); i++ {
		if g := keyframes[i] - keyframes[i-1]; g > max {
			max = g
		}
	}
	return max
}
