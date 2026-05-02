package video

import (
	"strings"
	"testing"
	"time"
)

func TestScaleResolution(t *testing.T) {
	tests := []struct {
		name         string
		srcW, srcH   int
		maxH         int
		wantW, wantH int
	}{
		{"passthrough zero", 1920, 1080, 0, 1920, 1080},
		{"passthrough larger", 1920, 1080, 1200, 1920, 1080},
		{"passthrough equal", 1920, 1080, 1080, 1920, 1080},
		{"4K to 1080p", 3840, 2160, 1080, 1920, 1080},
		{"1080p to 720p", 1920, 1080, 720, 1280, 720},
		{"1080p to 540p", 1920, 1080, 540, 960, 540},
		{"odd width rounds down", 1918, 1080, 720, 1278, 720},
		{"ultrawide", 2560, 1080, 720, 1706, 720},
		{"4K to 540p", 3840, 2160, 540, 960, 540},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := ScaleResolution(tt.srcW, tt.srcH, tt.maxH)
			if w != tt.wantW || h != tt.wantH {
				t.Errorf("ScaleResolution(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.srcW, tt.srcH, tt.maxH, w, h, tt.wantW, tt.wantH)
			}
			if w%2 != 0 {
				t.Errorf("width %d is not even", w)
			}
		})
	}
}

func TestResolutionString(t *testing.T) {
	if got := ResolutionString(1920, 1080); got != "1920x1080" {
		t.Errorf("got %q", got)
	}
}

func TestPeakBitrate(t *testing.T) {
	tests := []struct {
		name     string
		playlist string
		want     int64
	}{
		{
			"empty",
			"",
			0,
		},
		{
			"no segments",
			"#EXTM3U\n#EXT-X-VERSION:7\n#EXT-X-ENDLIST\n",
			0,
		},
		{
			"single segment",
			"#EXTM3U\n#EXTINF:4.0,\n#EXT-X-BYTERANGE:1000000@0\nseg.m4s\n",
			// 1000000 * 8 / 4.0 = 2000000 bps
			2_000_000,
		},
		{
			"peak is second segment",
			strings.Join([]string{
				"#EXTM3U",
				"#EXTINF:4.0,",
				"#EXT-X-BYTERANGE:1000000@0",
				"seg0.m4s",
				"#EXTINF:2.0,",
				"#EXT-X-BYTERANGE:2000000@1000000",
				"seg1.m4s",
			}, "\n"),
			// seg0: 1000000*8/4=2000000, seg1: 2000000*8/2=8000000
			8_000_000,
		},
		{
			"missing byterange",
			"#EXTM3U\n#EXTINF:4.0,\nseg.ts\n",
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PeakBitrate(tt.playlist)
			if got != tt.want {
				t.Errorf("PeakBitrate() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFixupMediaPlaylist(t *testing.T) {
	playlist := "#EXTM3U\nseg-tmp.m4s\nseg-tmp.m4s\n"
	got := FixupMediaPlaylist(playlist, "seg-tmp.m4s", "abc123.m4s")
	if strings.Contains(got, "seg-tmp") {
		t.Errorf("old name still present: %s", got)
	}
	if !strings.Contains(got, "abc123.m4s") {
		t.Errorf("new name not found: %s", got)
	}
}

func TestFixupMediaPlaylistAudio(t *testing.T) {
	playlist := strings.Join([]string{
		"#EXTM3U",
		"#EXT-X-VERSION:7",
		"#EXT-X-TARGETDURATION:6",
		"#EXT-X-MEDIA-SEQUENCE:0",
		"#EXT-X-PLAYLIST-TYPE:VOD",
		`#EXT-X-MAP:URI="media0.mp4",BYTERANGE="744@0"`,
		"#EXTINF:6.000000,",
		"#EXT-X-BYTERANGE:131072@744",
		"media0.mp4",
		"#EXT-X-ENDLIST",
		"",
	}, "\n")
	got := FixupMediaPlaylist(playlist, "media0.mp4", "/-/aud/arend123.mp4")
	if strings.Contains(got, "media0.mp4") {
		t.Errorf("old name still present: %s", got)
	}
	if !strings.Contains(got, `#EXT-X-MAP:URI="/-/aud/arend123.mp4"`) {
		t.Errorf("EXT-X-MAP URI not substituted: %s", got)
	}
	// Segment URI line (the one not preceded by #EXT-X-MAP).
	if !strings.Contains(got, "\n/-/aud/arend123.mp4\n") {
		t.Errorf("segment URI not substituted: %s", got)
	}
}

func TestGenerateMVPlaylist(t *testing.T) {
	entries := []MVEntry{
		{URI: "best.m3u8", Bandwidth: 5_000_000, Resolution: "1920x1080", Codecs: "avc1.640028,mp4a.40.2"},
		{URI: "low.m3u8", Bandwidth: 500_000, Resolution: "640x360"},
		{URI: "bare.m3u8", Bandwidth: 100_000},
	}
	playlist := GenerateMVPlaylist(entries, nil, nil)
	if !strings.Contains(playlist, "#EXTM3U") {
		t.Error("missing #EXTM3U header")
	}
	if !strings.Contains(playlist, "best.m3u8") {
		t.Error("missing best.m3u8 URI")
	}
	// No alternatives: no EXT-X-MEDIA lines and no AUDIO=/SUBTITLES= attributes.
	if strings.Contains(playlist, "EXT-X-MEDIA") {
		t.Errorf("unexpected EXT-X-MEDIA line: %s", playlist)
	}
	if strings.Contains(playlist, "SUBTITLES=") {
		t.Errorf("unexpected SUBTITLES= attribute: %s", playlist)
	}
	if strings.Contains(playlist, "AUDIO=") {
		t.Errorf("unexpected AUDIO= attribute: %s", playlist)
	}
}

func TestGenerateMVPlaylistWithSubtitles(t *testing.T) {
	entries := []MVEntry{
		{URI: "best.m3u8", Bandwidth: 5_000_000, Resolution: "1920x1080", Codecs: "avc1.640028,mp4a.40.2"},
		{URI: "low.m3u8", Bandwidth: 500_000, Resolution: "640x360"},
	}
	t.Run("single", func(t *testing.T) {
		subs := []MVSubtitle{
			{URI: "/sub/1.m3u8", Name: "English", Language: "eng", Default: true},
		}
		got := GenerateMVPlaylist(entries, nil, subs)
		if !strings.Contains(got, `#EXT-X-MEDIA:TYPE=SUBTITLES`) {
			t.Errorf("missing EXT-X-MEDIA SUBTITLES tag: %s", got)
		}
		if !strings.Contains(got, `GROUP-ID="subs"`) {
			t.Errorf("missing GROUP-ID: %s", got)
		}
		if !strings.Contains(got, `NAME="English"`) {
			t.Errorf("missing NAME: %s", got)
		}
		if !strings.Contains(got, `LANGUAGE="eng"`) {
			t.Errorf("missing LANGUAGE: %s", got)
		}
		if !strings.Contains(got, `DEFAULT=YES`) {
			t.Errorf("missing DEFAULT=YES: %s", got)
		}
		if !strings.Contains(got, `AUTOSELECT=YES`) {
			t.Errorf("missing AUTOSELECT=YES: %s", got)
		}
		if strings.Contains(got, `FORCED=YES`) {
			t.Errorf("unexpected FORCED=YES: %s", got)
		}
		if !strings.Contains(got, `URI="/sub/1.m3u8"`) {
			t.Errorf("missing subtitle URI: %s", got)
		}
		// Every variant must carry SUBTITLES="subs".
		count := strings.Count(got, `SUBTITLES="subs"`)
		if count != len(entries) {
			t.Errorf("SUBTITLES=\"subs\" appeared %d times, want %d: %s",
				count, len(entries), got)
		}
	})
	t.Run("multiple with forced", func(t *testing.T) {
		subs := []MVSubtitle{
			{URI: "/sub/1.m3u8", Name: "English", Language: "eng", Default: true},
			{URI: "/sub/2.m3u8", Name: "English (Forced)", Language: "eng", Forced: true},
			{URI: "/sub/3.m3u8", Name: "Japanese", Language: "jpn"},
		}
		got := GenerateMVPlaylist(entries, nil, subs)
		// One EXT-X-MEDIA line per subtitle.
		if c := strings.Count(got, "#EXT-X-MEDIA:TYPE=SUBTITLES"); c != 3 {
			t.Errorf("got %d EXT-X-MEDIA lines, want 3: %s", c, got)
		}
		if !strings.Contains(got, `FORCED=YES`) {
			t.Errorf("missing FORCED=YES on forced track: %s", got)
		}
		// Forced tracks must not be auto-selected — clients should
		// only show forced narrative when the user opts in.
		if c := strings.Count(got, `AUTOSELECT=YES`); c != 2 {
			t.Errorf("got %d AUTOSELECT=YES (want 2: non-forced only): %s", c, got)
		}
		// At most one DEFAULT=YES.
		if c := strings.Count(got, "DEFAULT=YES"); c != 1 {
			t.Errorf("got %d DEFAULT=YES, want 1: %s", c, got)
		}
	})
	t.Run("name without language", func(t *testing.T) {
		subs := []MVSubtitle{
			{URI: "/sub/x.m3u8", Name: "Track 3"},
		}
		got := GenerateMVPlaylist(entries, nil, subs)
		if strings.Contains(got, "LANGUAGE=") {
			t.Errorf("unexpected LANGUAGE attribute: %s", got)
		}
		if !strings.Contains(got, `NAME="Track 3"`) {
			t.Errorf("missing NAME: %s", got)
		}
	})
}

func TestGenerateMVPlaylistWithAudio(t *testing.T) {
	entries := []MVEntry{
		{URI: "best.m3u8", Bandwidth: 5_000_000, Resolution: "1920x1080", Codecs: "hvc1.1.6.L150.90,mp4a.40.2"},
		{URI: "low.m3u8", Bandwidth: 500_000, Resolution: "640x360"},
	}
	auds := []MVAudio{
		{URI: "/-/audpls/a1.m3u8", Name: "English (Stereo)", Language: "eng", Channels: 2, Default: true},
		{URI: "/-/audpls/a2.m3u8", Name: "English (5.1)", Language: "eng", Channels: 6},
	}
	got := GenerateMVPlaylist(entries, auds, nil)
	if !strings.Contains(got, `#EXT-X-MEDIA:TYPE=AUDIO`) {
		t.Errorf("missing EXT-X-MEDIA AUDIO tag: %s", got)
	}
	if !strings.Contains(got, `GROUP-ID="aud"`) {
		t.Errorf("missing GROUP-ID: %s", got)
	}
	if !strings.Contains(got, `CHANNELS="2"`) {
		t.Errorf("missing CHANNELS=\"2\": %s", got)
	}
	if !strings.Contains(got, `CHANNELS="6"`) {
		t.Errorf("missing CHANNELS=\"6\": %s", got)
	}
	if !strings.Contains(got, `URI="/-/audpls/a1.m3u8"`) {
		t.Errorf("missing audio URI: %s", got)
	}
	// Exactly one DEFAULT=YES across the AUDIO group entries.
	mediaAudio := 0
	defaults := 0
	for line := range strings.SplitSeq(got, "\n") {
		if !strings.HasPrefix(line, "#EXT-X-MEDIA:TYPE=AUDIO") {
			continue
		}
		mediaAudio++
		if strings.Contains(line, "DEFAULT=YES") {
			defaults++
		}
	}
	if mediaAudio != 2 {
		t.Errorf("got %d EXT-X-MEDIA AUDIO lines, want 2: %s", mediaAudio, got)
	}
	if defaults != 1 {
		t.Errorf("got %d DEFAULT=YES on AUDIO entries, want 1: %s", defaults, got)
	}
	// Every variant must carry AUDIO="aud".
	if c := strings.Count(got, `AUDIO="aud"`); c != len(entries) {
		t.Errorf(`AUDIO="aud" appeared %d times, want %d: %s`, c, len(entries), got)
	}
	if strings.Contains(got, "SUBTITLES=") {
		t.Errorf("unexpected SUBTITLES= attribute: %s", got)
	}
}

func TestGenerateMVPlaylistWithAudioAndSubtitles(t *testing.T) {
	entries := []MVEntry{
		{URI: "best.m3u8", Bandwidth: 5_000_000, Resolution: "1920x1080"},
		{URI: "low.m3u8", Bandwidth: 500_000, Resolution: "640x360"},
	}
	auds := []MVAudio{
		{URI: "/-/audpls/a1.m3u8", Name: "English", Language: "eng", Channels: 2, Default: true},
	}
	subs := []MVSubtitle{
		{URI: "/-/subpls/s1.m3u8", Name: "English", Language: "eng", Default: true},
	}
	got := GenerateMVPlaylist(entries, auds, subs)
	if c := strings.Count(got, `AUDIO="aud"`); c != len(entries) {
		t.Errorf(`AUDIO="aud" appeared %d times, want %d: %s`, c, len(entries), got)
	}
	if c := strings.Count(got, `SUBTITLES="subs"`); c != len(entries) {
		t.Errorf(`SUBTITLES="subs" appeared %d times, want %d: %s`, c, len(entries), got)
	}
	if c := strings.Count(got, `#EXT-X-MEDIA:TYPE=AUDIO`); c != 1 {
		t.Errorf("got %d AUDIO EXT-X-MEDIA lines, want 1: %s", c, got)
	}
	if c := strings.Count(got, `#EXT-X-MEDIA:TYPE=SUBTITLES`); c != 1 {
		t.Errorf("got %d SUBTITLES EXT-X-MEDIA lines, want 1: %s", c, got)
	}
}

func TestGenerateSubtitleMediaPlaylist(t *testing.T) {
	got := GenerateSubtitleMediaPlaylist(90*time.Minute, "/-/sub/abc.vtt")
	want := "#EXTM3U\n" +
		"#EXT-X-VERSION:3\n" +
		"#EXT-X-PLAYLIST-TYPE:VOD\n" +
		"#EXT-X-TARGETDURATION:5400\n" +
		"#EXT-X-MEDIA-SEQUENCE:0\n" +
		"#EXTINF:5400,\n" +
		"/-/sub/abc.vtt\n" +
		"#EXT-X-ENDLIST\n"
	if got != want {
		t.Errorf("playlist mismatch\n got=%q\nwant=%q", got, want)
	}
}

func TestGenerateSubtitleMediaPlaylistFractional(t *testing.T) {
	// 1.5s -> ceil to 2 for TARGETDURATION.
	got := GenerateSubtitleMediaPlaylist(1500*time.Millisecond, "/-/sub/x.vtt")
	if !strings.Contains(got, "#EXT-X-TARGETDURATION:2\n") {
		t.Errorf("expected TARGETDURATION:2, got: %s", got)
	}
	if !strings.Contains(got, "#EXTINF:1.5,\n") {
		t.Errorf("expected EXTINF:1.5, got: %s", got)
	}
}
