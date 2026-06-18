package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"ily.dev/act3/model"
)

func (c *Config) doUpload(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	file, _, err := req.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	medID := req.FormValue("med-id")
	sedID := req.FormValue("sed-id")
	epID := req.FormValue("ep-id")
	colID := req.FormValue("col-id")

	var kind model.ImageKind
	switch {
	case medID != "", sedID != "":
		kind = model.ImagePoster
	case epID != "":
		kind = model.ImageThumbnail
	case colID != "":
		kind = model.ImageBanner
	default:
		return nil, &model.ValidationError{
			Op:  "params",
			Err: fmt.Errorf("missing param med-id, sed-id, ep-id, or col-id"),
		}
	}

	originalID, err := c.Model.ImageCreate(ctx, file, kind)
	if err != nil {
		return nil, err
	}

	switch {
	case medID != "":
		_, err = c.withTxRW(ctx, func(tx *model.TxRW) (node, error) {
			return nil, tx.MovieEditionPosterIDSet(ctx, medID, originalID)
		})
	case sedID != "":
		_, err = c.withTxRW(ctx, func(tx *model.TxRW) (node, error) {
			return nil, tx.SeriesEditionPosterIDSet(ctx, sedID, originalID)
		})
	case epID != "":
		_, err = c.withTxRW(ctx, func(tx *model.TxRW) (node, error) {
			return nil, tx.EpisodeThumbnailIDSet(ctx, epID, originalID)
		})
	case colID != "":
		_, err = c.withTxRW(ctx, func(tx *model.TxRW) (node, error) {
			return nil, tx.CollectionBannerIDSet(ctx, colID, originalID)
		})
	}
	if err != nil {
		return nil, err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil, nil
}

// maxUploadFormField caps each non-file multipart text part. Form
// fields here are ID strings; anything larger is a client bug or an
// abuse attempt and we'd rather fail fast than buffer.
const maxUploadFormField = 1 << 10

// doVideoUpload streams a video file multipart upload directly into
// the CAS store via model.VideoUploadCreate. Registered on the
// streaming path so the global request-body cap doesn't apply; we
// rely on req.MultipartReader to avoid spooling the upload through
// req.FormFile's default /tmp file.
//
// The form is expected to emit small text parts (sed-id / med-id /
// ep-id) before the file part named "video".
func (c *Config) doVideoUpload(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	mr, err := req.MultipartReader()
	if err != nil {
		return nil, &model.ValidationError{Op: "multipart", Err: err}
	}

	var medID, epID string
	size := req.ContentLength // file size upper bound: includes multipart framing
	var fileSeen bool
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch part.FormName() {
		case "med-id":
			medID, err = readField(part)
		case "ep-id":
			epID, err = readField(part)
		case "size":
			// The exact file size, sent by our own upload form just
			// ahead of the file.
			var s string
			s, err = readField(part)
			if err == nil {
				if n, perr := strconv.ParseInt(s, 10, 64); perr == nil && n > 0 {
					size = n
				}
			}
		case "video":
			if fileSeen {
				err = &model.ValidationError{
					Op:  "video",
					Err: fmt.Errorf("multiple video parts"),
				}
				break
			}
			fileSeen = true
			var medp, epp *string
			if medID != "" {
				medp = &medID
			}
			if epID != "" {
				epp = &epID
			}
			_, err = c.Model.VideoUploadCreate(ctx, part, part.FileName(), size, medp, epp)
		}
		part.Close()
		if err != nil {
			return nil, err
		}
	}
	if !fileSeen {
		return nil, &model.ValidationError{
			Op:  "params",
			Err: fmt.Errorf("missing video file part"),
		}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil, nil
}

func readField(r io.Reader) (string, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxUploadFormField+1))
	if err != nil {
		return "", err
	}
	if len(b) > maxUploadFormField {
		return "", &model.ValidationError{
			Op:  "form field",
			Err: fmt.Errorf("value exceeds %d bytes", maxUploadFormField),
		}
	}
	return string(b), nil
}
