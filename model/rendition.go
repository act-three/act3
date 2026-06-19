package model

import (
	"cmp"
	"database/sql"
	"errors"
	"sort"

	"ily.dev/act3/database/schema"
)

// QualityOption describes one entry in the player quality menu.
// RenditionID is empty for the "Auto" entry (full MV playlist with
// all renditions); for other entries it pins to a specific rendition.
// The player JS composes a combined-pin URL from RenditionID and the
// current audio selection. Display formatting lives in the view layer.
type QualityOption struct {
	RenditionID   string // empty for Auto
	Height        int    // actual output pixel height; 0 for Auto
	FPS           int    // actual output frame rate, rounded; 0 for Auto
	TargetBitrate int    // kbit/s; 0 for Auto
}

// QualityOptions returns the quality menu entries for a video.
// The first entry is "Auto" (the multivariant playlist);
// the rest are individual rendition playlists sorted by bitrate
// (highest first). Each rendition's Height is the actual output
// pixel height — the planner stores the cap as MaxHeight (0 when
// the source already satisfies the cap), so the actual output is
// MaxHeight when set and the source height otherwise.
func (tx *TxR) QualityOptions(v *Video) ([]QualityOption, error) {
	rends, err := tx.q.RenditionListStreamingByVideoID(v.ID())
	if err != nil {
		return nil, err
	}

	// Sort by bitrate descending.
	sort.Slice(rends, func(i, j int) bool {
		return rends[i].TargetBitrate > rends[j].TargetBitrate
	})

	var srcFPS int64
	if v.v.FrameRateDen > 0 {
		srcFPS = (v.v.FrameRateNum + v.v.FrameRateDen/2) / v.v.FrameRateDen
	}

	opts := []QualityOption{{}} // Auto
	for _, r := range rends {
		if r.Playlist == "" {
			continue // not yet encoded
		}
		opts = append(opts, QualityOption{
			RenditionID:   r.ID,
			Height:        int(cmp.Or(r.MaxHeight, v.v.Height)),
			FPS:           int(cmp.Or(r.MaxFPS, srcFPS)),
			TargetBitrate: int(r.TargetBitrate),
		})
	}
	return opts, nil
}

type RenditionForDownload struct {
	path  string
	label string
}

func (r *RenditionForDownload) Path() string  { return r.path }
func (r *RenditionForDownload) Label() string { return r.label }

// videoExtensionForContentType returns the file extension
// (without leading dot) for a video MIME type.
func videoExtensionForContentType(ct string) string {
	switch ct {
	case "video/x-matroska":
		return "mkv"
	case "video/mp4":
		return "mp4"
	case "video/x-msvideo":
		return "avi"
	case "video/mp2t":
		return "ts"
	case "video/x-flv":
		return "flv"
	case "video/ogg":
		return "ogv"
	case "video/webm":
		return "webm"
	default:
		return "bin"
	}
}

func (tx *TxR) Rendition(id string) (schema.Rendition, error) {
	return tx.q.RenditionGet(id)
}

func (tx *TxR) RenditionListStreamingByEpisodeID(epID string) ([]schema.Rendition, error) {
	return tx.q.RenditionListStreamingByEpisodeID(epID)
}

// VideoDownload bundles everything the download handler needs to
// serve a response: the CAS blob key, the Content-Type to pin, and
// the server-generated filename for Content-Disposition.
type VideoDownload struct {
	Key         string
	ContentType string
	Filename    string
}

// VideoDownloadForEpisode resolves the download response for id
// (a Rendition ID or a Video ID) in the context of the episode
// identified by epID within the series edition identified by sedID.
// The filename is derived from the episode's basename in that
// edition, matching the browse-page download listing.
func (tx *TxR) VideoDownloadForEpisode(id, epID, sedID string) (VideoDownload, error) {
	ep, err := tx.EpisodeInEdition(epID, sedID)
	if err != nil {
		return VideoDownload{}, err
	}
	return tx.videoDownloadFor(id, ep.basename())
}

// VideoDownloadForMovieEdition is the movie-edition counterpart to
// VideoDownloadForEpisode.
func (tx *TxR) VideoDownloadForMovieEdition(id, medID string) (VideoDownload, error) {
	med, err := tx.MovieEditionHead(medID)
	if err != nil {
		return VideoDownload{}, err
	}
	return tx.videoDownloadFor(id, med.basename())
}

// videoDownloadFor tries the given id as a Rendition first, then as
// a Video, and assembles the download response using the caller's
// owner-derived basename. Returns sql.ErrNoRows if neither table
// has a matching row, or if a matching Rendition exists but has not
// yet been encoded.
func (tx *TxR) videoDownloadFor(id, basename string) (VideoDownload, error) {
	rend, err := tx.q.RenditionGet(id)
	if err == nil {
		if rend.Key == "" {
			return VideoDownload{}, sql.ErrNoRows
		}
		return VideoDownload{
			Key:         rend.Key,
			ContentType: "video/mp4",
			Filename:    basename + ".mp4",
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return VideoDownload{}, err
	}
	vid, err := tx.q.VideoGet(id)
	if err != nil {
		return VideoDownload{}, err
	}
	return VideoDownload{
		Key:         vid.OriginalKey,
		ContentType: vid.OriginalType,
		Filename:    basename + "." + videoExtensionForContentType(vid.OriginalType),
	}, nil
}
