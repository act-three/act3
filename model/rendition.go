package model

import (
	"database/sql"
	"errors"
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
			Path:  "/-/qpls/" + r.ID + ".m3u8",
		})
	}
	return opts, nil
}

func qualityLabel(r schema.Rendition) string {
	if r.MaxHeight > 0 {
		return fmt.Sprintf("%dp", r.MaxHeight)
	}
	// MaxHeight 0 means source resolution.
	mbps := float64(r.TargetBitrate) / 1000
	if mbps >= 1 {
		return fmt.Sprintf("%.0f Mbps", mbps)
	}
	return fmt.Sprintf("%d kbps", r.TargetBitrate)
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

func (tx *TxR) Rendition(ctx Context, id string) (schema.Rendition, error) {
	return tx.q.RenditionGet(ctx, id)
}

// VariantMVPlaylist returns a single-variant MV playlist that pins
// the given rendition as the only quality option but still carries
// the full audio rendition group and subtitle group. The fixed-quality
// menu items use this so picking "1080p" doesn't drop audio (post
// chunk-3, video renditions are encoded -an; the only place audio
// lives is in the AUDIO group on an MV playlist). Returns "" when the
// rendition isn't yet encoded or the audio side isn't ready.
func (tx *TxR) VariantMVPlaylist(ctx Context, rendID string) (string, error) {
	rend, err := tx.q.RenditionGet(ctx, rendID)
	if err != nil {
		return "", err
	}
	if rend.Key == "" || rend.Playlist == "" {
		return "", nil
	}
	vid, err := tx.q.VideoGet(ctx, rend.VideoID)
	if err != nil {
		return "", err
	}
	encodedAudio, err := tx.q.AudioRenditionListEncodedForMV(ctx, vid.ID)
	if err != nil {
		return "", err
	}
	tracks, err := tx.q.AudioTrackListByVideoID(ctx, vid.ID)
	if err != nil {
		return "", err
	}
	subTracks, err := tx.q.SubtitleTrackListByVideoID(ctx, vid.ID)
	if err != nil {
		return "", err
	}
	return buildMVPlaylist(vid, []schema.Rendition{rend}, encodedAudio, tracks, subTracks), nil
}

func (tx *TxR) RenditionListStreamingByEpisodeID(ctx Context, epID string) ([]schema.Rendition, error) {
	return tx.q.RenditionListStreamingByEpisodeID(ctx, epID)
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
func (tx *TxR) VideoDownloadForEpisode(ctx Context, id, epID, sedID string) (VideoDownload, error) {
	ep, err := tx.EpisodeInEdition(ctx, epID, sedID)
	if err != nil {
		return VideoDownload{}, err
	}
	return tx.videoDownloadFor(ctx, id, ep.basename())
}

// VideoDownloadForMovieEdition is the movie-edition counterpart to
// VideoDownloadForEpisode.
func (tx *TxR) VideoDownloadForMovieEdition(ctx Context, id, medID string) (VideoDownload, error) {
	med, err := tx.MovieEditionHead(ctx, medID)
	if err != nil {
		return VideoDownload{}, err
	}
	return tx.videoDownloadFor(ctx, id, med.basename())
}

// videoDownloadFor tries the given id as a Rendition first, then as
// a Video, and assembles the download response using the caller's
// owner-derived basename. Returns sql.ErrNoRows if neither table
// has a matching row, or if a matching Rendition exists but has not
// yet been encoded.
func (tx *TxR) videoDownloadFor(ctx Context, id, basename string) (VideoDownload, error) {
	rend, err := tx.q.RenditionGet(ctx, id)
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
	vid, err := tx.q.VideoGet(ctx, id)
	if err != nil {
		return VideoDownload{}, err
	}
	return VideoDownload{
		Key:         vid.OriginalKey,
		ContentType: vid.OriginalType,
		Filename:    basename + "." + videoExtensionForContentType(vid.OriginalType),
	}, nil
}
