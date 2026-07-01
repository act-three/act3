package kind

import "fmt"

// A VideoUpload is the kind of object an uploaded video is attached to.
//
//sumtype:decl
type VideoUpload interface {
	fmt.Stringer
	videoUpload()
}

func (Episode) videoUpload()      {}
func (MovieEdition) videoUpload() {}

var videoUploads = []VideoUpload{Episode{}, MovieEdition{}}

// ParseVideoUpload returns the VideoUpload named by s.
func ParseVideoUpload(s string) (VideoUpload, error) {
	for _, k := range videoUploads {
		if k.String() == s {
			return k, nil
		}
	}
	return nil, fmt.Errorf("kind: bad VideoUpload %q", s)
}
