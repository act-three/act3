package model

import (
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
func (v *Video) MVPlaylist() string         { return v.v.MVPlaylist }
func (v *Video) State() string              { return v.v.State }
func (v *Video) PlaylistPath() string       { return "/-/plr/" + v.ID() + ".m3u8" }
func (v *Video) AudioTracks() []*AudioTrack { return v.audioTracks }

// Active reports whether this Video is the active one for the work
// (episode or movie edition) it was loaded for. False when the Video
// was loaded outside a work context.
func (v *Video) Active() bool { return v.active }

// Playable reports whether this Video has a multivariant playlist and
// is therefore ready to stream. Pending/importing/re-encoding videos
// return false.
func (v *Video) Playable() bool { return v.v.MVPlaylist != "" }

func (tx *TxR) Video(ctx Context, id string) (*Video, error) {
	v, err := tx.q.VideoGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Video{v: v}, nil
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
