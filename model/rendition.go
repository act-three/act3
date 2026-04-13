package model

import (
	"fmt"
	"sort"

	"ily.dev/act3/database/schema"
)

// QualityOption describes one entry in the player quality menu.
type QualityOption struct {
	Label string // e.g. "Auto", "1080p", "720p", "540p"
	Path  string // playlist URL path
}

// QualityOptions returns the quality menu entries for a video.
// The first entry is "Auto" (the multivariant playlist);
// the rest are individual rendition playlists sorted by bitrate
// (highest first).
func (tx *TxR) QualityOptions(ctx Context, v *Video) ([]QualityOption, error) {
	rends, err := tx.q.RenditionListStreamingByVideoID(ctx, v.ID())
	if err != nil {
		return nil, err
	}

	// Sort by bitrate descending.
	sort.Slice(rends, func(i, j int) bool {
		return rends[i].TargetBitrate > rends[j].TargetBitrate
	})

	opts := []QualityOption{
		{Label: "Auto", Path: v.PlaylistPath()},
	}
	for _, r := range rends {
		if r.Playlist == "" {
			continue // not yet encoded
		}
		opts = append(opts, QualityOption{
			Label: qualityLabel(r),
			Path:  "/-/pls/" + r.ID + ".m3u8",
		})
	}
	return opts, nil
}

func qualityLabel(r schema.Rendition) string {
	suffix := ""
	if r.SurroundAudio != 0 {
		suffix = " 5.1"
	}
	if r.MaxHeight > 0 {
		return fmt.Sprintf("%dp%s", r.MaxHeight, suffix)
	}
	// MaxHeight 0 means source resolution.
	mbps := float64(r.TargetBitrate) / 1000
	if mbps >= 1 {
		return fmt.Sprintf("%.0f Mbps%s", mbps, suffix)
	}
	return fmt.Sprintf("%d kbps%s", r.TargetBitrate, suffix)
}

type RenditionForDownload struct {
	path     string
	filename string
	label    string
}

func (r *RenditionForDownload) Path() string     { return r.path }
func (r *RenditionForDownload) Filename() string { return r.filename }
func (r *RenditionForDownload) Label() string    { return r.label }

// videoExtensionForContentType returns the file extension (with dot)
// for a video MIME type.
func videoExtensionForContentType(ct string) string {
	switch ct {
	case "video/x-matroska":
		return ".mkv"
	case "video/mp4":
		return ".mp4"
	case "video/x-msvideo":
		return ".avi"
	case "video/mp2t":
		return ".ts"
	case "video/x-flv":
		return ".flv"
	case "video/ogg":
		return ".ogv"
	case "video/webm":
		return ".webm"
	default:
		return ".bin"
	}
}

func (tx *TxR) Rendition(ctx Context, id string) (schema.Rendition, error) {
	return tx.q.RenditionGet(ctx, id)
}

func (tx *TxR) RenditionListStreamingByEpisodeID(ctx Context, epID string) ([]schema.Rendition, error) {
	return tx.q.RenditionListStreamingByEpisodeID(ctx, epID)
}
