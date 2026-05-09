package ffmpeg

import (
	"time"
)

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
