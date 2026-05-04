package view

import (
	"reflect"
	"slices"
	"testing"

	"ily.dev/act3/model"
)

// TestQualityMenuLabels exercises the full label pipeline as the
// player menu renders it: model returns the options with Auto first;
// the view moves Auto to the end and formats labels, appending FPS
// when frame rates vary and bitrate within runs of duplicates.
func TestQualityMenuLabels(t *testing.T) {
	auto := model.QualityOption{}
	rend := func(id string, height, fps, kbps int) model.QualityOption {
		return model.QualityOption{
			RenditionID:   id,
			Height:        height,
			FPS:           fps,
			TargetBitrate: kbps,
		}
	}
	tests := []struct {
		name string
		opts []model.QualityOption // as the model returns them
		want []string              // labels in render order
	}{
		{
			name: "all distinct heights, uniform fps",
			opts: []model.QualityOption{
				auto,
				rend("a", 1080, 24, 5000),
				rend("b", 720, 24, 3000),
				rend("c", 540, 24, 1500),
			},
			want: []string{"1080p", "720p", "540p", "Auto"},
		},
		{
			name: "adjacent duplicate heights get bitrate suffix",
			opts: []model.QualityOption{
				auto,
				rend("a", 720, 24, 2780),
				rend("b", 720, 24, 2400),
				rend("c", 540, 24, 1200),
				rend("d", 540, 24, 336),
			},
			want: []string{
				"720p — 2.8 MB/s",
				"720p — 2.4 MB/s",
				"540p — 1.2 MB/s",
				"540p — 336 kB/s",
				"Auto",
			},
		},
		{
			name: "run of three gets bitrate on all three",
			opts: []model.QualityOption{
				auto,
				rend("a", 1080, 24, 6000),
				rend("b", 1080, 24, 5000),
				rend("c", 1080, 24, 4000),
				rend("d", 720, 24, 3000),
			},
			want: []string{
				"1080p — 6.0 MB/s",
				"1080p — 5.0 MB/s",
				"1080p — 4.0 MB/s",
				"720p",
				"Auto",
			},
		},
		{
			name: "fps varies — all renditions show fps",
			opts: []model.QualityOption{
				auto,
				rend("a", 1080, 60, 6000),
				rend("b", 720, 60, 3000),
				rend("c", 540, 30, 1500),
				rend("d", 540, 30, 420),
			},
			want: []string{
				"1080p 60fps",
				"720p 60fps",
				"540p 30fps — 1.5 MB/s",
				"540p 30fps — 420 kB/s",
				"Auto",
			},
		},
		{
			name: "non-adjacent dupes don't trigger bitrate",
			opts: []model.QualityOption{
				auto,
				rend("a", 1080, 60, 6000),
				rend("b", 720, 60, 3000),
				rend("c", 1080, 30, 1500),
			},
			want: []string{
				"1080p 60fps",
				"720p 60fps",
				"1080p 30fps",
				"Auto",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := qualityLabels(autoLast(tt.opts))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("labels mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestFPSVaries(t *testing.T) {
	auto := model.QualityOption{}
	rend := func(fps int) model.QualityOption {
		return model.QualityOption{RenditionID: "x", FPS: fps}
	}
	tests := []struct {
		name string
		opts []model.QualityOption
		want bool
	}{
		{"empty", nil, false},
		{"only auto", []model.QualityOption{auto}, false},
		{"single rendition", []model.QualityOption{rend(24)}, false},
		{"two same", []model.QualityOption{rend(24), rend(24)}, false},
		{"two different", []model.QualityOption{rend(24), rend(30)}, true},
		{"three same with auto first", []model.QualityOption{auto, rend(60), rend(60), rend(60)}, false},
		{"three same with auto last", []model.QualityOption{rend(60), rend(60), rend(60), auto}, false},
		{"varies with auto first", []model.QualityOption{auto, rend(60), rend(60), rend(30)}, true},
		{"varies with auto last", []model.QualityOption{rend(60), rend(30), rend(30), auto}, true},
		{"varies with auto sandwiched", []model.QualityOption{rend(24), auto, rend(30)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := slices.Clone(tt.opts)
			got := fpsVaries(tt.opts)
			if got != tt.want {
				t.Errorf("fpsVaries = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.opts, before) {
				t.Errorf("fpsVaries mutated input\n got: %+v\nwant: %+v", tt.opts, before)
			}
		})
	}
}
