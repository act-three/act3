package model

import (
	"fmt"
	"path"

	"ily.dev/act3/database/schema"
)

type RenditionForStreaming struct {
	url string
}

func (r *RenditionForStreaming) URL() string { return r.url }

func (tx *TxR) RenditionForStreamingList(ctx Context, epID string) ([]*RenditionForStreaming, error) {
	a, err := tx.q.RenditionForStreamingListByVideoID(ctx, epID)
	if err != nil {
		return nil, err
	}
	var rends []*RenditionForStreaming
	for _, r := range a {
		rends = append(rends, &RenditionForStreaming{
			url: path.Join("/stream", r.Hash),
		})
	}
	return rends, nil
}

type RenditionForDownload struct {
	url      string
	filename string
	label    string
}

func (r *RenditionForDownload) URL() string      { return r.url }
func (r *RenditionForDownload) Filename() string { return r.filename }
func (r *RenditionForDownload) Label() string    { return r.label }

func (tx *TxR) RenditionForDownloadList(ctx Context, epID string) ([]*RenditionForDownload, error) {
	vids, err := tx.q.VideoListByEpisodeID(ctx, epID)
	if err != nil {
		return nil, err
	}
	var rends []*RenditionForDownload
	for _, vid := range vids {

		// TODO(april): make the filename good:
		// [series title] [year] [edition if not main] s01e01 [episode title] [resolution] [sdr hdr] .mkv
		filename := "episode.mkv"

		rends = append(rends, &RenditionForDownload{
			url:      path.Join("/dl", vid.OriginalHash, filename),
			filename: filename,
			label:    fmt.Sprintf("Original (%s)", vid.ReleasePath),
		})
	}
	return rends, nil
}

func (tx *TxR) VideoListByEpisodeID(ctx Context, epID string) ([]schema.Video, error) {
	return tx.q.VideoListByEpisodeID(ctx, epID)
}

func (tx *TxR) RenditionForStreamingListByEpisodeID(ctx Context, epID string) ([]schema.RenditionForStreaming, error) {
	return tx.q.RenditionForStreamingListByVideoID(ctx, epID)
}
