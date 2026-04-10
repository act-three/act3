// Package priority defines task priority constants.
//
// Lower values run first. Constants are grouped by purpose;
// some share the same numeric value — that's intentional
// (e.g. Default and Encode1st are both 0).
package priority

const (
	// Default is the default for tasks that don't specify
	// a priority.
	Default = 0

	// Image fetch priorities. Posters are more visible than
	// thumbnails, but both are less important than other IO
	// tasks (metadata fetches, downloads) that run at Default.
	FetchPoster    = 10
	FetchThumbnail = 20

	// Encoding priorities control the order renditions are
	// produced across all videos. Best rendition first, then
	// pass-1 analysis, then remaining tiers from most to
	// least useful.
	Encode1st = 0 // best rendition
	Pass1     = 1 // ffmpeg pass-1 + surround variant
	Encode2nd = 2 // lowest bitrate (quick preview)
	Encode3rd = 3 // 720p
	Encode4th = 4 // 1080p
	Encode5th = 5 // 540p30
	Encode6th = 6 // near-source 20 Mbps
)
