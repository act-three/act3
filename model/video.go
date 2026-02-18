package model

import "ily.dev/act3/database/schema"

type Video struct {
	v schema.Video
}

func (v *Video) ID() string           { return v.v.ID }
func (v *Video) ReleaseID() string    { return v.v.ReleaseID }
func (v *Video) ReleasePath() string  { return v.v.ReleasePath }
func (v *Video) OriginalHash() string { return v.v.OriginalHash }
func (v *Video) MVPlaylist() string   { return v.v.MVPlaylist }
func (v *Video) PlaylistURL() string  { return "/vid/" + v.ID() + ".m3u8" }
func (v *Video) PlayerURL() string    { return "/player/" + v.ID() }

func (tx *TxR) Video(ctx Context, id string) (*Video, error) {
	v, err := tx.q.VideoGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Video{v}, nil
}

func (tx *TxR) VideoListByEpisodeID(ctx Context, epID string) ([]schema.Video, error) {
	return tx.q.VideoListByEpisodeID(ctx, epID)
}

func vidMapByID(vids []schema.Video) map[string]*Video {
	m := map[string]*Video{}
	for i := range vids {
		m[vids[i].ID] = &Video{vids[i]}
	}
	return m
}

func vidMapByEpisodeID(evs []schema.EpisodeVideo, vidByID map[string]*Video) map[string][]*Video {
	m := map[string][]*Video{}
	for _, ev := range evs {
		if v := vidByID[ev.VideoID]; v != nil {
			m[ev.EpisodeID] = append(m[ev.EpisodeID], v)
		}
	}
	return m
}
