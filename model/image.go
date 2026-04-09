package model

import (
	"bytes"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/gen2brain/webp"
	_ "golang.org/x/image/webp"

	"kr.dev/errorfmt"
)

// ImageKind selects the target aspect ratio for stored images.
type ImageKind int

const (
	ImagePoster    ImageKind = iota // 2:3 — movie/series poster
	ImageThumbnail                  // 16:9 — episode thumbnail
	ImageBanner                     // 1000:185 — collection banner
)

func (k ImageKind) ratio() (w, h int) {
	switch k {
	case ImagePoster:
		return 2, 3
	case ImageThumbnail:
		return 16, 9
	case ImageBanner:
		return 1000, 185
	}
	panic("bad ImageKind")
}

// webpQuality is the lossy WebP quality used for stored images.
// q=80 produces SSIM ~0.985 on a 6 MP source while staying under
// ~600 KB; smaller crops produce proportionally smaller output.
const webpQuality = 80

// ImageCreate decodes r, center-crops it to the target aspect ratio
// for kind, re-encodes as lossy WebP, and writes the result into the
// blob store. Decode failures surface as ValidationError so callers
// at the HTTP edge can return 400.
func (m *Model) ImageCreate(r io.Reader, kind ImageKind) (key string, err error) {
	defer errorfmt.Handlef("image create: %w", &err)

	src, _, err := image.Decode(r)
	if err != nil {
		return "", &ValidationError{Op: "decode image", Err: err}
	}

	cropped := centerCrop(src, kind)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, cropped, webp.Options{Quality: webpQuality}); err != nil {
		return "", err
	}
	return m.store.Copy(&buf)
}

// centerCrop returns the largest rectangle of src whose aspect ratio
// matches kind, anchored at src's center, copied into a fresh
// zero-origin NRGBA. We deliberately copy rather than return a
// SubImage: gen2brain/webp's encoder mishandles non-zero-origin
// bounds (it reads Pix[] without offsetting by Bounds().Min) and
// produces visibly corrupted output.
func centerCrop(src image.Image, kind ImageKind) image.Image {
	tw, th := kind.ratio()
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()

	var cw, ch int
	if sw*th > sh*tw {
		// src is wider than target — keep height, crop width.
		ch = sh
		cw = sh * tw / th
	} else {
		// src is taller than (or equal to) target — keep width, crop height.
		cw = sw
		ch = sw * th / tw
	}
	x0 := b.Min.X + (sw-cw)/2
	y0 := b.Min.Y + (sh-ch)/2

	dst := image.NewNRGBA(image.Rect(0, 0, cw, ch))
	draw.Draw(dst, dst.Bounds(), src, image.Pt(x0, y0), draw.Src)
	return dst
}
