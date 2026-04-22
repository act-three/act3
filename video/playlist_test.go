package video

import (
	"strings"
	"testing"
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

func TestGenerateMVPlaylist(t *testing.T) {
	entries := []MVEntry{
		{URI: "best.m3u8", Bandwidth: 5_000_000, Resolution: "1920x1080", Codecs: "avc1.640028,mp4a.40.2"},
		{URI: "low.m3u8", Bandwidth: 500_000, Resolution: "640x360"},
		{URI: "bare.m3u8", Bandwidth: 100_000},
	}
	playlist := GenerateMVPlaylist(entries)
	if !strings.Contains(playlist, "#EXTM3U") {
		t.Error("missing #EXTM3U header")
	}
	if !strings.Contains(playlist, "best.m3u8") {
		t.Error("missing best.m3u8 URI")
	}
}
