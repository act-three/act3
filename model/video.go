package model

import (
	"time"

	"ily.dev/act3/database/schema"
)

type Video struct {
	v           schema.Video
	audioTracks []*AudioTrack
	active      bool
}

func (v *Video) ID() string                 { return v.v.ID }
func (v *Video) Name() string               { return v.v.Name }
func (v *Video) OriginalKey() string        { return v.v.OriginalKey }
func (v *Video) State() string              { return v.v.State }
func (v *Video) Duration() time.Duration    { return time.Duration(v.v.Duration) * time.Millisecond }
func (v *Video) PlaylistPath() string       { return "/-/plr/" + v.ID() + ".m3u8" }
func (v *Video) AudioTracks() []*AudioTrack { return v.audioTracks }

// Active reports whether this Video is the active one for the work
// (episode or movie edition) it was loaded for. False when the Video
// was loaded outside a work context.
func (v *Video) Active() bool { return v.active }

// Playable reports whether this Video has all the renditions needed
// to assemble a multivariant playlist. Pending/importing/re-encoding
// videos return false. The MV playlist itself is built on demand via
// (TxR).MVPlaylist; this flag is the cheap gate used by SQL filters
// and active-video promotion.
func (v *Video) Playable() bool { return v.v.Playable != 0 }

func (tx *TxR) Video(ctx Context, id string) (*Video, error) {
	v, err := tx.q.VideoGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Video{v: v}, nil
}

// MVPlaylist builds the multivariant HLS playlist for a video on
// demand from the current rendition state. Returns "" when the video
// is not yet playable (no encoded video, or any source audio track
// without an encoded rendition).
func (tx *TxR) MVPlaylist(ctx Context, videoID string) (string, error) {
	vid, err := tx.q.VideoGet(ctx, videoID)
	if err != nil {
		return "", err
	}
	encoded, err := tx.q.RenditionListEncodedStreamingByVideoID(ctx, videoID)
	if err != nil {
		return "", err
	}
	encodedAudio, err := tx.q.AudioRenditionListEncodedForMV(ctx, videoID)
	if err != nil {
		return "", err
	}
	tracks, err := tx.q.AudioTrackListByVideoID(ctx, videoID)
	if err != nil {
		return "", err
	}
	subTracks, err := tx.q.SubtitleTrackListByVideoID(ctx, videoID)
	if err != nil {
		return "", err
	}
	return buildMVPlaylist(vid, encoded, encodedAudio, tracks, subTracks), nil
}

func (tx *TxR) VideoListByEpisodeID(ctx Context, epID string) ([]schema.Video, error) {
	return tx.q.VideoListByEpisodeID(ctx, epID)
}

func vidMapByID(vids []schema.Video) map[string]*Video {
	m := map[string]*Video{}
	for i := range vids {
		m[vids[i].ID] = &Video{v: vids[i]}
	}
	return m
}

func vidMapByEpisodeID(evs []schema.EpisodeVideo, vidByID map[string]*Video) map[string][]*Video {
	m := map[string][]*Video{}
	for _, ev := range evs {
		v := vidByID[ev.VideoID]
		if v == nil {
			continue
		}
		clone := *v
		clone.active = ev.Active != 0
		m[ev.EpisodeID] = append(m[ev.EpisodeID], &clone)
	}
	return m
}
