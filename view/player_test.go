package view

import (
	"reflect"
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
