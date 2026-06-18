package model

import (
	"ily.dev/act3/database/schema"
)

type AudioRendition struct {
	ar schema.AudioRendition
}

// AudioRendition returns the AudioRendition row for the given ID.
func (tx *TxR) AudioRendition(ctx Context, id string) (schema.AudioRendition, error) {
	return tx.q.AudioRenditionGet(id)
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

// AudioOption describes one entry in the player audio-track menu.
// Title, Language, and Channels are kept separate — the UI composes
// the visible label from these inputs. Title is the source-track name
// with no channel-layout suffix (a 5.1 source's stereo and 5.1
// renditions share the same Title; the menu disambiguates via
// Channels).
type AudioOption struct {
	ID       string // AudioRendition ID
	Title    string // source-track name from AudioTrack.Name()
	Language string // ISO 639-2 as stored
	Channels int    // 1 (mono), 2 (stereo), 6 (5.1)
	Default  bool   // whether this is the HLS DEFAULT
}

// AudioOptions returns the audio-menu entries for a video, in the same
// order as the MV playlist's EXT-X-MEDIA AUDIO group. The first entry
// is the HLS DEFAULT.
func (tx *TxR) AudioOptions(ctx Context, v *Video) ([]AudioOption, error) {
	rends, err := tx.q.AudioRenditionListEncodedForMV(v.ID())
	if err != nil {
		return nil, err
	}
	tracks, err := tx.q.AudioTrackListByVideoID(v.ID())
	if err != nil {
		return nil, err
	}
	tracksByID := make(map[string]schema.AudioTrack, len(tracks))
	for _, at := range tracks {
		tracksByID[at.ID] = at
	}
	var opts []AudioOption
	for i, ar := range rends {
		at := tracksByID[ar.AudioTrackID]
		opts = append(opts, AudioOption{
			ID:       ar.ID,
			Title:    (&AudioTrack{at: at}).Name(),
			Language: at.Language,
			Channels: int(ar.Channels),
			Default:  i == 0,
		})
	}
	return opts, nil
}
