package model

import (
	"ily.dev/act3/database/schema"
)

// AudioRenditionMediaKey returns the blob key of the fMP4 media for the
// audio rendition with the given ID. The key is empty until the
// rendition has been encoded.
func (tx *TxR) AudioRenditionMediaKey(id string) (string, error) {
	ar, err := tx.q.AudioRenditionGet(id)
	if err != nil {
		return "", err
	}
	return ar.Key, nil
}

// AudioRenditionPlaylist returns the HLS media playlist for the audio
// rendition with the given ID. The playlist is empty until the
// rendition has been encoded.
func (tx *TxR) AudioRenditionPlaylist(id string) (string, error) {
	ar, err := tx.q.AudioRenditionGet(id)
	if err != nil {
		return "", err
	}
	return ar.Playlist, nil
}

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
func (tx *TxR) AudioOptions(v *Video) ([]AudioOption, error) {
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
