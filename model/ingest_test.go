package model

import (
	"context"
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

	t.Run("native HDR10 source signals PQ variants", func(t *testing.T) {
		hdrVid := vid
		hdrVid.ColorTransfer = "smpte2084"
		rend := rendHalf
		rend.Codec = "hevc"
		got := buildMVPlaylist(hdrVid,
			[]schema.Rendition{rend},
			[]schema.AudioRendition{encAudio},
			[]schema.AudioTrack{track},
			nil,
		)
		if !strings.Contains(got, "VIDEO-RANGE=PQ") {
			t.Errorf("missing VIDEO-RANGE=PQ:\n%s", got)
		}
		if !strings.Contains(got, "hvc1.2.4") {
			t.Errorf("missing Main 10 codec string:\n%s", got)
		}
	})

	t.Run("stereo downmix of surround is not autoselected", func(t *testing.T) {
		surround := schema.AudioTrack{ID: "at6", VideoID: "vid1", Language: "eng", StreamIndex: 1, Channels: 6}
		got := buildMVPlaylist(vid,
			[]schema.Rendition{rendHalf},
			[]schema.AudioRendition{
				{ID: "ar6", VideoID: "vid1", AudioTrackID: "at6", Channels: 6, Key: "ka6"},
				{ID: "ar2", VideoID: "vid1", AudioTrackID: "at6", Channels: 2, Key: "ka2"},
			},
			[]schema.AudioTrack{surround},
			nil,
		)
		for line := range strings.SplitSeq(got, "\n") {
			if !strings.Contains(line, "TYPE=AUDIO") {
				continue
			}
			want := strings.Contains(line, "ar6.m3u8")
			if got := strings.Contains(line, "AUTOSELECT=YES"); got != want {
				t.Errorf("AUTOSELECT=YES is %v, want %v: %s", got, want, line)
			}
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

func TestIsPlayableMV(t *testing.T) {
	rend := schema.Rendition{ID: "rend1", Key: "k1", Playlist: "#EXTM3U\n"}
	track := schema.AudioTrack{ID: "at1", VideoID: "vid1", Language: "eng", StreamIndex: 0}
	encAudio := schema.AudioRendition{ID: "ar1", VideoID: "vid1", AudioTrackID: "at1", Channels: 2, Key: "ka1"}

	tests := []struct {
		name   string
		rends  []schema.Rendition
		audio  []schema.AudioRendition
		tracks []schema.AudioTrack
		want   bool
	}{
		{"no video renditions", nil, []schema.AudioRendition{encAudio}, []schema.AudioTrack{track}, false},
		{"audio not yet ready", []schema.Rendition{rend}, nil, []schema.AudioTrack{track}, false},
		{"no audio tracks at all", []schema.Rendition{rend}, nil, nil, true},
		{"all ready", []schema.Rendition{rend}, []schema.AudioRendition{encAudio}, []schema.AudioTrack{track}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPlayableMV(tt.rends, tt.audio, tt.tracks); got != tt.want {
				t.Errorf("isPlayableMV = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestReencodeReplansAfterDelete is a regression test for ACT-179.
// taskReencode opens an inner WithTxRW to delete the existing
// renditions, then re-plans. Before the fix, planAndCreateRenditions
// ran on the outer (stale) TxR; its existing-rendition guard saw the
// pre-delete snapshot and silently returned, leaving the video without
// renditions and without queued encode tasks. With the fix, planning
// runs in a fresh TxR — the guard sees the empty list and proceeds to
// open the source. We pass a bogus OriginalKey so that step fails
// noisily, which is what we assert on: pre-fix taskReencode returned
// nil.
//
// Uses a file-backed DB because the snapshot pinning the bug depends
// on only manifests in WAL mode; :memory: shared-cache falls back to
// MEMORY journal, where the outer reader blocks the inner writer.
func TestReencodeReplansAfterDelete(t *testing.T) {
	ctx := context.Background()
	m := newFileBackedTestModel(t)

	vidID := createVideoRow(t, m, "v.mkv", "fakekey", []string{"rendkey1"})

	err := m.WithTxR(ctx, func(tx *TxR) error {
		return tx.taskReencode([]string{vidID})
	})

	// The inner WithTxRW commits the delete regardless of what runs
	// afterward; both pre- and post-fix the rendition row is gone here.
	var renditions []schema.Rendition
	if rerr := m.WithTxR(ctx, func(tx *TxR) error {
		var e error
		renditions, e = tx.q.RenditionListByVideoID(vidID)
		return e
	}); rerr != nil {
		t.Fatal(rerr)
	}
	if len(renditions) != 0 {
		t.Fatalf("renditions should be deleted, got %d", len(renditions))
	}

	if err == nil {
		t.Fatal("expected planAndCreateRenditions to attempt the source open and fail; got nil — guard short-circuited on stale snapshot?")
	}
}

// TestEncodeTaskNoOps verifies that encode tasks operate on exactly
// the rendition named in their args: a task whose rendition is gone
// (reencode or trash) or already encoded (retry) must no-op rather
// than draw a different rendition. This pins down the fix for a race
// where concurrent same-video tasks each picked the "next unencoded"
// rendition, encoding one rendition twice and another never.
func TestEncodeTaskNoOps(t *testing.T) {
	ctx := context.Background()
	m := newFileBackedTestModel(t)

	vidID := createVideoRow(t, m, "v.mkv", "fakekey", []string{"rendkey1"})

	var rendID, audioRendID string
	err := m.WithTxRW(ctx, func(tx *TxRW) error {
		var e error
		rends, e := tx.q.RenditionListByVideoID(vidID)
		if e != nil {
			return e
		}
		rendID = rends[0].ID
		at, e := tx.q.AudioTrackCreate(schema.AudioTrackCreateParams{
			VideoID: vidID, Channels: 2, ChannelLayout: "stereo",
			SampleRate: 48000, Codec: "aac", Profile: "LC",
		})

		if e != nil {
			return e
		}
		ar, e := tx.q.AudioRenditionCreate(schema.AudioRenditionCreateParams{
			VideoID: vidID, AudioTrackID: at.ID,
			Channels: 2, Bitrate: 128, Codec: "aac", Priority: 1,
		})

		if e != nil {
			return e
		}
		audioRendID = ar.ID
		_, e = tx.q.AudioRenditionUpdateEncode(schema.AudioRenditionUpdateEncodeParams{
			ID: ar.ID, Key: "audiokey1",
		})

		return e
	})

	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		f    func(*TxR, []string) error
		args []string
	}{
		{"rend already encoded", (*TxR).taskIngestEncodeRend, []string{rendID}},
		{"rend deleted", (*TxR).taskIngestEncodeRend, []string{"rendgone"}},
		{"dl-rend already encoded", (*TxR).taskIngestEncodeDownloadRend, []string{rendID}},
		{"dl-rend deleted", (*TxR).taskIngestEncodeDownloadRend, []string{"rendgone"}},
		{"audio already encoded", (*TxR).taskIngestEncodeAudio, []string{audioRendID}},
		{"audio deleted", (*TxR).taskIngestEncodeAudio, []string{"argone"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.WithTxR(ctx, func(tx *TxR) error {
				return tt.f(tx, tt.args)
			})

			if err != nil {
				t.Fatalf("want no-op, got error: %v", err)
			}
		})
	}
}
