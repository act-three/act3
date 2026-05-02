package model

import (
	"ily.dev/act3/database/schema"
)

type AudioRendition struct {
	ar schema.AudioRendition
}

// AudioRendition returns the AudioRendition row for the given ID.
func (tx *TxR) AudioRendition(ctx Context, id string) (schema.AudioRendition, error) {
	return tx.q.AudioRenditionGet(ctx, id)
}

func (a *AudioRendition) ID() string           { return a.ar.ID }
func (a *AudioRendition) VideoID() string      { return a.ar.VideoID }
func (a *AudioRendition) AudioTrackID() string { return a.ar.AudioTrackID }
func (a *AudioRendition) Channels() int        { return int(a.ar.Channels) }
func (a *AudioRendition) Bitrate() int64       { return a.ar.Bitrate }
func (a *AudioRendition) Codec() string        { return a.ar.Codec }
func (a *AudioRendition) Key() string          { return a.ar.Key }
func (a *AudioRendition) Playlist() string     { return a.ar.Playlist }
func (a *AudioRendition) Priority() int        { return int(a.ar.Priority) }
