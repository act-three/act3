package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"ily.dev/act3/model"
	"ily.dev/act3/model/kind"
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

	if _, err := c.Model.ImageUploadCreate(ctx, file, medID, sedID, epID, colID); err != nil {
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
// the blob store via model.VideoUploadCreate. Registered on the
// streaming path so the global request-body cap doesn't apply; we
// rely on req.MultipartReader to avoid spooling the upload through
// req.FormFile's default /tmp file.
//
// The form is expected to emit the kind and id text parts before the
// file part named "video".
func (c *Config) doVideoUpload(w http.ResponseWriter, req *http.Request) (node, error) {
	ctx := req.Context()
	mr, err := req.MultipartReader()
	if err != nil {
		return nil, &model.ValidationError{Op: "multipart", Err: err}
	}

	var k kind.VideoUpload
	var id string
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
		case "kind":
			var s string
			s, err = readField(part)
			if err == nil {
				k, err = kind.ParseVideoUpload(s)
			}
		case "id":
			id, err = readField(part)
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
			_, err = c.Model.VideoUploadCreate(ctx, part, part.FileName(), size, k, id)
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
