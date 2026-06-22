package model

import (
	"context"
	"reflect"
	"testing"

	"ily.dev/act3/database/schema"
)

// TestQualityOptions covers the resolution and frame-rate fields
// returned for the player quality menu. The Auto entry is always
// first; the others mirror the renditions sorted by bitrate, with
// Height resolved from MaxHeight (falling back to source) and FPS
// resolved from MaxFPS (falling back to rounded source FPS).
func TestQualityOptions(t *testing.T) {
	type rend struct {
		bitrate, maxHeight, maxFPS int64
	}
	cases := []struct {
		name      string
		srcHeight int64
		srcFRNum  int64
		srcFRDen  int64
		rends     []rend
		want      []QualityOption // first element is Auto
	}{
		{
			name:      "1080p 24fps source — no caps fire",
			srcHeight: 1080, srcFRNum: 24000, srcFRDen: 1001,
			rends: []rend{
				{bitrate: 6000, maxHeight: 0, maxFPS: 0}, // best (remux)
				{bitrate: 5000, maxHeight: 0, maxFPS: 0}, // 1080p ladder, no caps fire
				{bitrate: 3000, maxHeight: 720, maxFPS: 0},
				{bitrate: 1500, maxHeight: 540, maxFPS: 0}, // 30fps cap doesn't fire (src=24)
			},
			want: []QualityOption{
				{},
				{Height: 1080, FPS: 24, TargetBitrate: 6000},
				{Height: 1080, FPS: 24, TargetBitrate: 5000},
				{Height: 720, FPS: 24, TargetBitrate: 3000},
				{Height: 540, FPS: 24, TargetBitrate: 1500},
			},
		},
		{
			name:      "1080p 60fps source — fps cap fires on 540p",
			srcHeight: 1080, srcFRNum: 60000, srcFRDen: 1001,
			rends: []rend{
				{bitrate: 6000, maxHeight: 0, maxFPS: 0}, // best
				{bitrate: 3000, maxHeight: 720, maxFPS: 0},
				{bitrate: 1500, maxHeight: 540, maxFPS: 30}, // src 60 > 30
				{bitrate: 420, maxHeight: 540, maxFPS: 30},
			},
			want: []QualityOption{
				{},
				{Height: 1080, FPS: 60, TargetBitrate: 6000},
				{Height: 720, FPS: 60, TargetBitrate: 3000},
				{Height: 540, FPS: 30, TargetBitrate: 1500},
				{Height: 540, FPS: 30, TargetBitrate: 420},
			},
		},
		{
			name:      "540p 30fps source — height caps don't fire",
			srcHeight: 540, srcFRNum: 30, srcFRDen: 1,
			rends: []rend{
				{bitrate: 1500, maxHeight: 0, maxFPS: 0},
				{bitrate: 420, maxHeight: 0, maxFPS: 0},
			},
			want: []QualityOption{
				{},
				{Height: 540, FPS: 30, TargetBitrate: 1500},
				{Height: 540, FPS: 30, TargetBitrate: 420},
			},
		},
		{
			name:      "23.976fps rounds to 24",
			srcHeight: 1080, srcFRNum: 24000, srcFRDen: 1001,
			rends: []rend{
				{bitrate: 5000, maxHeight: 0, maxFPS: 0},
			},
			want: []QualityOption{
				{},
				{Height: 1080, FPS: 24, TargetBitrate: 5000},
			},
		},
		{
			name:      "29.97fps rounds to 30",
			srcHeight: 1080, srcFRNum: 30000, srcFRDen: 1001,
			rends: []rend{
				{bitrate: 5000, maxHeight: 0, maxFPS: 0},
			},
			want: []QualityOption{
				{},
				{Height: 1080, FPS: 30, TargetBitrate: 5000},
			},
		},
		{
			name:      "unknown source frame rate stays 0",
			srcHeight: 1080, srcFRNum: 0, srcFRDen: 0,
			rends: []rend{
				{bitrate: 5000, maxHeight: 0, maxFPS: 0},
			},
			want: []QualityOption{
				{},
				{Height: 1080, FPS: 0, TargetBitrate: 5000},
			},
		},
	}

	ctx := context.Background()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newTestModel(t)
			var v *Video
			err := m.WithTxRW(ctx, func(tx *TxRW) error {
				vid, err := tx.q.VideoCreate(schema.VideoCreateParams{Name: c.name})
				if err != nil {
					return err
				}
				err = tx.q.VideoUpdateProbe(schema.VideoUpdateProbeParams{
					ID:           vid.ID,
					Width:        1920,
					Height:       c.srcHeight,
					FrameRateNum: c.srcFRNum,
					FrameRateDen: c.srcFRDen,
				})

				if err != nil {
					return err
				}
				for _, r := range c.rends {
					rend, err := tx.q.RenditionCreate(schema.RenditionCreateParams{
						VideoID:       vid.ID,
						Purpose:       "streaming",
						Codec:         "h264",
						TargetBitrate: r.bitrate,
						MaxHeight:     r.maxHeight,
						MaxFPS:        r.maxFPS,
					})

					if err != nil {
						return err
					}
					_, err = tx.q.RenditionUpdateEncode(schema.RenditionUpdateEncodeParams{
						ID: rend.ID, Key: "k", Playlist: "#EXTM3U\n",
					})

					if err != nil {
						return err
					}
				}
				vid, err = tx.q.VideoGet(vid.ID)
				if err != nil {
					return err
				}
				v = &Video{v: vid}
				return nil
			})

			if err != nil {
				t.Fatal(err)
			}

			var got []QualityOption
			err = m.WithTxR(ctx, func(tx *TxR) error {
				got = tx.QualityOptions(v)
				return nil
			})

			if err != nil {
				t.Fatal(err)
			}

			// Strip RenditionID for comparison; we only care about the
			// computed fields.
			for i := range got {
				got[i].RenditionID = ""
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("QualityOptions mismatch\n got: %+v\nwant: %+v", got, c.want)
			}
		})
	}
}
