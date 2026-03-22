package model

import (
	"fmt"
	"path"
	"sort"

	"ily.dev/act3/database/schema"
)

// QualityOption describes one entry in the player quality menu.
type QualityOption struct {
	Label string // e.g. "Auto", "1080p", "720p", "540p"
	URL   string // playlist URL
}

// QualityOptions returns the quality menu entries for a video.
// The first entry is "Auto" (the multivariant playlist);
// the rest are individual rendition playlists sorted by bitrate
// (highest first).
func (tx *TxR) QualityOptions(ctx Context, v *Video) ([]QualityOption, error) {
	rends, err := tx.q.RenditionForStreamingListDirectByVideoID(ctx, v.ID())
	if err != nil {
		return nil, err
	}

	// Sort by bitrate descending.
	sort.Slice(rends, func(i, j int) bool {
		return rends[i].TargetBitrate > rends[j].TargetBitrate
	})

	opts := []QualityOption{
		{Label: "Auto", URL: v.PlaylistURL()},
	}
	for _, r := range rends {
		if r.Playlist == "" {
			continue // not yet encoded
		}
		opts = append(opts, QualityOption{
			Label: qualityLabel(r),
			URL:   "/-/pls/" + r.ID + ".m3u8",
		})
	}
	return opts, nil
}

func qualityLabel(r schema.RenditionForStreaming) string {
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
	url      string
	filename string
	label    string
}

func (r *RenditionForDownload) URL() string      { return r.url }
func (r *RenditionForDownload) Filename() string { return r.filename }
func (r *RenditionForDownload) Label() string    { return r.label }

func (tx *TxR) RenditionForDownloadList(ctx Context, epID string) ([]*RenditionForDownload, error) {
	vids, err := tx.q.VideoListByEpisodeID(ctx, epID)
	if err != nil {
		return nil, err
	}
	var rends []*RenditionForDownload
	for _, vid := range vids {

		// TODO(april): make the filename good:
		// [series title] [year] [edition if not main] s01e01 [episode title] [resolution] [sdr hdr] .mkv
		filename := "episode.mkv"

		rends = append(rends, &RenditionForDownload{
			url:      path.Join("/-/dl", vid.OriginalHash, filename),
			filename: filename,
			label:    fmt.Sprintf("Original (%s)", vid.ReleasePath),
		})
	}
	return rends, nil
}

func (tx *TxR) RenditionForStreaming(ctx Context, id string) (schema.RenditionForStreaming, error) {
	return tx.q.RenditionForStreamingGet(ctx, id)
}

func (tx *TxR) RenditionForStreamingListByEpisodeID(ctx Context, epID string) ([]schema.RenditionForStreaming, error) {
	return tx.q.RenditionForStreamingListByVideoID(ctx, epID)
}
