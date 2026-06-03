package video

import (
	"testing"

	"ily.dev/act3/priority"
	"ily.dev/act3/video/ffmpeg"
)

func TestPlanAudioRenditions_noAudio(t *testing.T) {
	got := PlanAudioRenditions(&ffmpeg.ProbeResult{})
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestPlanAudioRenditions_aacStereo(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Audio: []ffmpeg.AudioStream{{
			Index:     0,
			CodecName: "aac",
			Channels:  2,
		}},
	}
	got := PlanAudioRenditions(probe)
	if len(got) != 1 {
		t.Fatalf("expected 1 rendition, got %d", len(got))
	}
	want := AudioRendition{
		SourceStreamIndex: 0,
		Channels:          2,
		Bitrate:           128,
		Codec:             "aac",
		Priority:          priority.EncodeAudio,
	}
	if got[0] != want {
		t.Errorf("got %+v, want %+v", got[0], want)
	}
}

func TestPlanAudioRenditions_ac3Surround(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Audio: []ffmpeg.AudioStream{{
			Index:     0,
			CodecName: "ac3",
			Channels:  6,
		}},
	}
	got := PlanAudioRenditions(probe)
	if len(got) != 2 {
		t.Fatalf("expected 2 renditions, got %d", len(got))
	}
	if got[0].Channels != 6 || got[0].Bitrate != 384 {
		t.Errorf("surround: got %+v", got[0])
	}
	if got[1].Channels != 2 || got[1].Bitrate != 128 {
		t.Errorf("stereo downmix: got %+v", got[1])
	}
	// The generated downmix must sort after the original it derives from.
	if got[0].SortKey != 0 || got[1].SortKey != 1 {
		t.Errorf("SortKey: got %d, %d; want 0, 1 (downmix after original)",
			got[0].SortKey, got[1].SortKey)
	}
	for i, r := range got {
		if r.SourceStreamIndex != 0 {
			t.Errorf("got[%d].SourceStreamIndex = %d, want 0", i, r.SourceStreamIndex)
		}
		if r.Codec != "aac" {
			t.Errorf("got[%d].Codec = %q, want aac", i, r.Codec)
		}
	}
}

func TestPlanAudioRenditions_multipleTracks(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Audio: []ffmpeg.AudioStream{
			{Index: 0, CodecName: "aac", Channels: 2, Language: "eng"},
			{Index: 1, CodecName: "ac3", Channels: 6, Language: "eng", Title: "Commentary"},
		},
	}
	got := PlanAudioRenditions(probe)
	if len(got) != 3 {
		t.Fatalf("expected 3 renditions, got %d", len(got))
	}
	wantIndices := []int{0, 1, 1}
	wantChannels := []int{2, 6, 2}
	for i, r := range got {
		if r.SourceStreamIndex != wantIndices[i] {
			t.Errorf("got[%d].SourceStreamIndex = %d, want %d", i, r.SourceStreamIndex, wantIndices[i])
		}
		if r.Channels != wantChannels[i] {
			t.Errorf("got[%d].Channels = %d, want %d", i, r.Channels, wantChannels[i])
		}
		// SortKey follows plan order: source-track order, downmix last.
		if r.SortKey != i {
			t.Errorf("got[%d].SortKey = %d, want %d", i, r.SortKey, i)
		}
	}
}

func TestPlanAudioRenditions_surroundSkipsDownmixWithStereoTwin(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Audio: []ffmpeg.AudioStream{
			{Index: 0, CodecName: "ac3", Channels: 6, Language: "eng", Title: "English"},
			{Index: 1, CodecName: "aac", Channels: 2, Language: "eng", Title: "English"},
		},
	}
	got := PlanAudioRenditions(probe)
	// Stream 0 keeps only its 5.1 rendition (the stereo downmix is
	// covered by stream 1); stream 1 keeps its stereo rendition.
	if len(got) != 2 {
		t.Fatalf("expected 2 renditions, got %d: %+v", len(got), got)
	}
	if got[0].SourceStreamIndex != 0 || got[0].Channels != 6 {
		t.Errorf("got[0] = %+v, want 5.1 from stream 0", got[0])
	}
	if got[1].SourceStreamIndex != 1 || got[1].Channels != 2 {
		t.Errorf("got[1] = %+v, want stereo from stream 1", got[1])
	}
}

func TestPlanAudioRenditions_surroundKeepsDownmixWhenStereoDiffers(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Audio: []ffmpeg.AudioStream{
			{Index: 0, CodecName: "ac3", Channels: 6, Language: "eng", Title: "English"},
			{Index: 1, CodecName: "aac", Channels: 2, Language: "eng", Title: "Commentary"},
		},
	}
	got := PlanAudioRenditions(probe)
	// Titles differ, so the surround track still gets its downmix.
	if len(got) != 3 {
		t.Fatalf("expected 3 renditions, got %d: %+v", len(got), got)
	}
}

func TestPlanAudioRenditions_preservesChannelCount(t *testing.T) {
	cases := []struct {
		name        string
		channels    int
		wantAsis    int   // as-is rendition channels
		wantBitrate int64 // as-is rendition bitrate
	}{
		{"5.0", 5, 5, 320},
		{"7.1", 8, 8, 512},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			probe := &ffmpeg.ProbeResult{
				Audio: []ffmpeg.AudioStream{{Index: 0, CodecName: "dts", Channels: c.channels}},
			}
			got := PlanAudioRenditions(probe)
			// Surround source with no stereo twin: as-is + downmix.
			if len(got) != 2 {
				t.Fatalf("expected 2 renditions, got %d: %+v", len(got), got)
			}
			if got[0].Channels != c.wantAsis || got[0].Bitrate != c.wantBitrate {
				t.Errorf("as-is: got %dch/%dk, want %dch/%dk",
					got[0].Channels, got[0].Bitrate, c.wantAsis, c.wantBitrate)
			}
			if got[1].Channels != 2 || got[1].Bitrate != 128 {
				t.Errorf("downmix: got %+v, want 2ch/128k", got[1])
			}
		})
	}
}

func TestPlanAudioRenditions_mono(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Audio: []ffmpeg.AudioStream{{
			Index:     0,
			CodecName: "aac",
			Channels:  1,
		}},
	}
	got := PlanAudioRenditions(probe)
	if len(got) != 1 {
		t.Fatalf("expected 1 rendition, got %d", len(got))
	}
	if got[0].Channels != 1 {
		t.Errorf("expected mono passthrough, got Channels=%d", got[0].Channels)
	}
	if got[0].Bitrate != 64 {
		t.Errorf("expected 64 kbit/s for mono, got %d", got[0].Bitrate)
	}
}
