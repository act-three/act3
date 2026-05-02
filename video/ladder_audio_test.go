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
	if got[0].Channels != 2 || got[0].Bitrate != 128 {
		t.Errorf("stereo: got %+v", got[0])
	}
	if got[1].Channels != 6 || got[1].Bitrate != 384 {
		t.Errorf("surround: got %+v", got[1])
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
	wantChannels := []int{2, 2, 6}
	for i, r := range got {
		if r.SourceStreamIndex != wantIndices[i] {
			t.Errorf("got[%d].SourceStreamIndex = %d, want %d", i, r.SourceStreamIndex, wantIndices[i])
		}
		if r.Channels != wantChannels[i] {
			t.Errorf("got[%d].Channels = %d, want %d", i, r.Channels, wantChannels[i])
		}
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
