package view

import (
	"reflect"
	"testing"

	"ily.dev/act3/model"
)

func TestQualityLabels(t *testing.T) {
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
		opts []model.QualityOption
		want []string
	}{
		{
			name: "all distinct heights, uniform fps",
			opts: []model.QualityOption{
				auto,
				rend("a", 1080, 24, 5000),
				rend("b", 720, 24, 3000),
				rend("c", 540, 24, 1500),
			},
			want: []string{"Auto", "1080p", "720p", "540p"},
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
				"Auto",
				"720p — 2.8 MB/s",
				"720p — 2.4 MB/s",
				"540p — 1.2 MB/s",
				"540p — 336 kB/s",
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
				"Auto",
				"1080p — 6.0 MB/s",
				"1080p — 5.0 MB/s",
				"1080p — 4.0 MB/s",
				"720p",
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
				"Auto",
				"1080p 60fps",
				"720p 60fps",
				"540p 30fps — 1.5 MB/s",
				"540p 30fps — 420 kB/s",
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
				"Auto",
				"1080p 60fps",
				"720p 60fps",
				"1080p 30fps",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := qualityLabels(tt.opts)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("qualityLabels mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}
