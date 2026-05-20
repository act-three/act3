package ffmpeg

import (
	"math/bits"
	"time"
)

// SegmentBoundaries picks a subset of source keyframes to use as HLS
// segment cuts, matching the schedule ffmpeg's HLS muxer follows on
// stream-copy. Inputs and outputs are display-order frame indices.
// The first keyframe (start of segment 0) is omitted — only the
// interior cut points between segments are returned.
//
// Cuts follow ffmpeg's absolute schedule exactly: segment N ends at
// the first keyframe with display PTS ≥ N × target. The target is
// expressed as a duration (rather than a frame count) so the
// comparison happens in time, with no fps-quantization drift — for
// 24000/1001fps content a 4s target yields ~95.904 frames per
// segment, and rounding that up to 96 cumulatively overshoots by
// ~0.096 frames per segment, enough to walk past a scene-change
// keyframe ffmpeg would have picked. Comparing k × fps.Den against
// N × target_ticks_per_seg in 1/fps.Num-second ticks keeps the
// arithmetic integer-exact.
//
// Re-encodes can be forced (via -force_key_frames "expr:eq(n,...)")
// to put keyframes at exactly the returned frame indices.
func SegmentBoundaries(keyframes []int64, fps FrameRate, target time.Duration) []int64 {
	if len(keyframes) <= 1 || !fps.Positive() || target <= 0 {
		return nil
	}
	// In 1/fps.Num-second ticks: a keyframe at display index k has
	// PTS k × fps.Den; one segment of `target` covers
	// target × fps.Num / time.Second ticks. Both are exact integers
	// for the rational frame rates HLS sources use.
	targetTicks := int64(target) * int64(fps.Num) / int64(time.Second)
	if targetTicks <= 0 {
		return nil
	}
	var out []int64
	cur := targetTicks
	den := int64(fps.Den)
	for _, k := range keyframes[1:] {
		if k*den >= cur {
			out = append(out, k)
			cur += targetTicks
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
	if minFrames <= 0 || fps.Num <= 0 || fps.Den <= 0 || duration <= 0 {
		return nil
	}
	// totalFrames = floor(duration × fps.Num / (fps.Den × time.Second)).
	// duration × fps.Num overflows int64 when fps is a large irreducible
	// fraction (e.g. CodedFrameRate derived from a multi-million-packet
	// source with a fine timebase: Num is the packet count scaled by
	// the timebase denominator, easily ~5×10⁷, and at a 39-minute
	// duration the product reaches ~10²³). Promote to a 128-bit
	// product so the division remains exact.
	hi, lo := bits.Mul64(uint64(duration), uint64(fps.Num))
	den := uint64(fps.Den) * uint64(time.Second)
	if hi >= den {
		// Would need a multi-century video to reach this. Bail out
		// rather than panic in bits.Div64.
		return nil
	}
	q, _ := bits.Div64(hi, lo, den)
	totalFrames := int64(q)
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
