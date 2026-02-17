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
