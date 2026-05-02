package model

import (
	"strings"
	"testing"

	"ily.dev/act3/database/schema"
)

func TestBuildMVPlaylist(t *testing.T) {
	vid := schema.Video{ID: "vid1", Width: 1920, Height: 1080}
	rendFull := schema.Rendition{
		ID:            "rend1080",
		VideoID:       "vid1",
		Purpose:       "streaming",
		Codec:         "h264",
		TargetBitrate: 5000,
		MaxHeight:     1080,
		Key:           "k1",
		Playlist:      "#EXTM3U\n",
	}
	rendHalf := schema.Rendition{
		ID:            "rend720",
		VideoID:       "vid1",
		Purpose:       "streaming",
		Codec:         "h264",
		TargetBitrate: 3000,
		MaxHeight:     720,
		Key:           "k2",
		Playlist:      "#EXTM3U\n",
	}
	track := schema.AudioTrack{ID: "at1", VideoID: "vid1", Language: "eng", StreamIndex: 0}
	encAudio := schema.AudioRendition{ID: "ar1", VideoID: "vid1", AudioTrackID: "at1", Channels: 2, Key: "ka1"}

	tests := []struct {
		name   string
		rends  []schema.Rendition
		audio  []schema.AudioRendition
		tracks []schema.AudioTrack
		want   string // "" means expect empty; "contains:..." means expect a substring.
	}{
		{
			name:   "no video renditions returns empty",
			rends:  nil,
			audio:  []schema.AudioRendition{encAudio},
			tracks: []schema.AudioTrack{track},
			want:   "",
		},
		{
			name:   "audio not yet ready returns empty",
			rends:  []schema.Rendition{rendFull},
			audio:  nil,
			tracks: []schema.AudioTrack{track},
			want:   "",
		},
		{
			name:   "full playlist when both ready",
			rends:  []schema.Rendition{rendFull, rendHalf},
			audio:  []schema.AudioRendition{encAudio},
			tracks: []schema.AudioTrack{track},
			want:   "contains:/-/pls/rend1080.m3u8",
		},
		{
			name:   "single-variant playlist contains exactly the picked rendition",
			rends:  []schema.Rendition{rendHalf},
			audio:  []schema.AudioRendition{encAudio},
			tracks: []schema.AudioTrack{track},
			want:   "contains:/-/pls/rend720.m3u8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildMVPlaylist(vid, tt.rends, tt.audio, tt.tracks, nil)
			switch {
			case tt.want == "":
				if got != "" {
					t.Errorf("expected empty, got %d bytes:\n%s", len(got), got)
				}
			case strings.HasPrefix(tt.want, "contains:"):
				sub := strings.TrimPrefix(tt.want, "contains:")
				if got == "" {
					t.Fatalf("expected non-empty playlist containing %q, got empty", sub)
				}
				if !strings.Contains(got, sub) {
					t.Errorf("playlist missing %q:\n%s", sub, got)
				}
			}
		})
	}

	t.Run("single-variant excludes other rendition URI", func(t *testing.T) {
		got := buildMVPlaylist(vid,
			[]schema.Rendition{rendHalf},
			[]schema.AudioRendition{encAudio},
			[]schema.AudioTrack{track},
			nil,
		)
		if strings.Contains(got, "rend1080.m3u8") {
			t.Errorf("single-variant playlist must not reference the unpicked rendition:\n%s", got)
		}
	})

	t.Run("audio group present in single-variant", func(t *testing.T) {
		got := buildMVPlaylist(vid,
			[]schema.Rendition{rendHalf},
			[]schema.AudioRendition{encAudio},
			[]schema.AudioTrack{track},
			nil,
		)
		if !strings.Contains(got, "/-/audpls/ar1.m3u8") {
			t.Errorf("expected audio rendition URI in playlist:\n%s", got)
		}
		if !strings.Contains(got, "TYPE=AUDIO") {
			t.Errorf("expected EXT-X-MEDIA TYPE=AUDIO entry:\n%s", got)
		}
	})
}
