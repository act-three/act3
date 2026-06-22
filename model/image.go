package model

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/webp"
	"golang.org/x/image/draw"
	xwebp "golang.org/x/image/webp"
	"kr.dev/errorfmt"

	"ily.dev/act3/assets"
	"ily.dev/act3/database/schema"
)

// ImageKind selects the target aspect ratio and per-density physical
// pixel widths for stored images.
type ImageKind int

const (
	ImagePoster    ImageKind = iota // 2:3 — movie/series poster
	ImageThumbnail                  // 16:9 — episode thumbnail
	ImageBanner                     // 1000:185 — collection banner
)

type imageKindSpec struct {
	aspectW int   // target aspect ratio numerator
	aspectH int   // target aspect ratio denominator
	widths  []int // physical pixel widths to generate, sorted ascending
}

var imageKindSpecs = map[ImageKind]imageKindSpec{
	ImagePoster:    {aspectW: 2, aspectH: 3, widths: []int{300, 600}},
	ImageThumbnail: {aspectW: 16, aspectH: 9, widths: []int{400, 800}},
	ImageBanner:    {aspectW: 1000, aspectH: 185, widths: []int{1224, 2448}},
}

func (k ImageKind) spec() imageKindSpec {
	s, ok := imageKindSpecs[k]
	if !ok {
		panic(fmt.Sprintf("bad ImageKind %d", k))
	}
	return s
}

// Aspect returns the target aspect ratio for the kind.
func (k ImageKind) Aspect() (w, h int) {
	s := k.spec()
	return s.aspectW, s.aspectH
}

// maxImageBytes caps the size of an input image. Tighter than the
// HTTP-edge MaxBytesHandler so background fetches don't blow up
// memory either.
const maxImageBytes = 10 << 20

// webpQuality is the lossy WebP quality for stored variants.
// q=80 produces SSIM ~0.985 on a 6 MP source while staying under
// ~600 KB; downscaled variants are proportionally smaller.
const webpQuality = 80

// decodeAllowedImage tries each allowed decoder in turn and
// returns the decoded image with its canonical MIME type. Going
// through image.Decode would instead consult the process-wide
// format registry, which any imported package can extend — we
// don't want a direct or transitive dependency to silently
// broaden the set of accepted formats. WebP decode goes through
// golang.org/x/image/webp because gen2brain/webp's decoder has
// known bugs; gen2brain is kept for encoding only.
func decodeAllowedImage(raw []byte) (image.Image, string, error) {
	for _, d := range []struct {
		decode func(io.Reader) (image.Image, error)
		mime   string
	}{
		{png.Decode, "image/png"},
		{jpeg.Decode, "image/jpeg"},
		{xwebp.Decode, "image/webp"},
	} {
		if img, err := d.decode(bytes.NewReader(raw)); err == nil {
			return img, d.mime, nil
		}
	}
	return nil, "", errors.New("unsupported format")
}

// Placeholder Image IDs. These rows are created at boot
// from embedded fallback PNGs and are referenced by the per-kind
// DEFAULT clauses on the parent FK columns, so every parent row
// always has a non-NULL image to render. The IDs intentionally
// avoid the ('i'||newID()) format used for real uploads so they
// can never collide.
const (
	imagePosterPlaceholderID    = "iplaceholderposter"
	imageThumbnailPlaceholderID = "iplaceholderthumbnail"
	imageBannerPlaceholderID    = "iplaceholderbanner"
)

func isPlaceholderImageID(id string) bool {
	switch id {
	case imagePosterPlaceholderID, imageThumbnailPlaceholderID, imageBannerPlaceholderID:
		return true
	}
	return false
}

// Image is the renderable form of an Image row. It carries only
// the Image table's primary key and the kind whose aspect
// ratio and configured widths the variants were generated from.
// Every URL and dimension a view needs is a pure derivation of
// those two fields — no DB or storage lookup happens at render
// time. The image handler at /-/img/{id}/{width} resolves the
// requested logical width to the best matching variant blob at
// request time.
type Image struct {
	ID   string
	Kind ImageKind
}

// IsPlaceholder reports whether the Image is one of the
// boot-time placeholder rows shared by every parent that has no
// user-uploaded image yet.
func (im Image) IsPlaceholder() bool { return isPlaceholderImageID(im.ID) }

// URL returns the image handler URL for this image at the given
// logical pixel width. The handler resolves "logical width" to
// the best matching stored variant when the request lands.
func (im Image) URL(width int) string {
	return "/-/img/" + im.ID + "/" + strconv.Itoa(width)
}

// SmallestURL returns the handler URL for the smallest configured
// variant. Used as the canonical <img src>. Layout size is set by
// CSS via the u-poster class's aspect-ratio rules, so no width or
// height attribute is needed on the rendered element.
func (im Image) SmallestURL() string {
	s := im.Kind.spec()
	return im.URL(s.widths[0])
}

// LargestURL returns the handler URL for the largest configured
// variant. Used by image editing dialogs where the rendered <img>
// should always show the highest-fidelity rendition regardless of
// viewport size.
func (im Image) LargestURL() string {
	s := im.Kind.spec()
	return im.URL(s.widths[len(s.widths)-1])
}

// Srcset returns a "url1 300w, url2 600w" string suitable for an
// <img srcset> attribute, covering every configured width for the
// kind.
func (im Image) Srcset() string {
	s := im.Kind.spec()
	parts := make([]string, 0, len(s.widths))
	for _, w := range s.widths {
		parts = append(parts, im.URL(w)+" "+strconv.Itoa(w)+"w")
	}
	return strings.Join(parts, ", ")
}

// FindImageVariantKey returns the blob key of the best stored variant
// for the requested logical width. "Best" is the largest variant
// whose physical width is at most the requested width; if every
// stored variant is wider than requested (e.g. when requested is
// smaller than the smallest stored variant), the smallest variant
// is returned.
func (tx *TxR) FindImageVariantKey(originalID string, width int) (key string, found bool) {
	vs, ok := txfind1(tx.q.ImageRenditionListByImageID(originalID))
	if !ok || len(vs) == 0 {
		return "", false
	}
	// vs is sorted ASC by Width. Walk forward and keep the last
	// variant that still fits inside the requested width.
	chosen := vs[0]
	for _, v := range vs {
		if int(v.Width) > width {
			break
		}
		chosen = v
	}
	return chosen.Key, true
}

// ImageCreate reads an image from r, validates that it decodes
// as one of the allowed formats (png/jpeg/webp), center-crops it
// to the target aspect ratio for kind, and writes both the
// original bytes and a set of downscaled WebP variants to the
// blob store and the Image / ImageRendition tables. Returns
// the new Image ID.
//
// Variants are generated at the per-density physical widths
// declared in imageKindSpecs. Configured widths that would
// upscale the cropped source are skipped. If the cropped source's
// native width falls strictly inside the configured range, a
// variant at the native size is added too so the best variant we
// offer reflects the actual fidelity of the source. The output
// never includes a width larger than the largest configured
// width. If even the smallest configured width would upscale, a
// single fallback variant at the cropped source's native size is
// generated so every Image has at least one rendition.
//
// The "original" row keeps the bytes byte-for-byte as uploaded
// so we can re-derive variants in the future without re-fetching
// from TMDB/TVmaze or asking the user to re-upload. It is not
// served to browsers — only the WebP variants are.
//
// Decode failures are returned as ValidationError so callers at
// the HTTP edge can return 400.
func (m *Model) ImageCreate(ctx context.Context, r io.ReadCloser, kind ImageKind) (originalID string, err error) {
	defer errorfmt.Handlef("image create: %w", &err)

	// Buffer the input so we can both store it byte-for-byte and
	// hand it to the decoder. MaxBytesReader makes the cap
	// overflow distinguishable from a generic read error.
	raw, err := io.ReadAll(http.MaxBytesReader(nil, r, maxImageBytes))
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return "", &ValidationError{
				Op:  "image too large",
				Err: fmt.Errorf("max %d bytes", maxImageBytes),
			}
		}
		return "", err
	}

	src, mime, err := decodeAllowedImage(raw)
	if err != nil {
		return "", &ValidationError{Op: "decode image", Err: err}
	}

	cropped := centerCrop(src, kind)
	encoded := encodeVariants(cropped, kind)
	return m.insertImage(ctx, "", mime, raw, encoded)
}

// insertImage writes raw + variants to the blob store and the
// Image / ImageRendition tables in a single transaction. If id
// is empty the Image row gets a freshly generated ID; otherwise
// the row is created with id (used for boot-time placeholders,
// whose IDs are fixed so parent FK DEFAULTs can reference them).
func (m *Model) insertImage(ctx context.Context, id, mime string, raw []byte, variants []encodedVariant) (originalID string, err error) {
	err = m.WithTxRW(ctx, func(tx *TxRW) error {
		originalKey, err := m.store.Copy(bytes.NewReader(raw))
		if err != nil {
			return err
		}
		if id == "" {
			io_, err := tx.q.ImageCreate(schema.ImageCreateParams{
				OriginalKey: originalKey,
				Type:        mime,
			})

			if err != nil {
				return err
			}
			originalID = io_.ID
		} else {
			if err := tx.q.ImageCreateWithID(schema.ImageCreateWithIDParams{
				ID:          id,
				OriginalKey: originalKey,
				Type:        mime,
			}); err != nil {
				return err
			}
			originalID = id
		}
		for _, v := range variants {
			variantKey, err := m.store.Copy(bytes.NewReader(v.bytes))
			if err != nil {
				return err
			}
			if err := tx.q.ImageRenditionCreate(schema.ImageRenditionCreateParams{
				Key:     variantKey,
				ImageID: originalID,
				Type:    "image/webp",
				Width:   int64(v.width),
				Height:  int64(v.height),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return originalID, nil
}

// imageFetch GETs url and hands the body to ImageCreate. The ctx
// timeout bounds the whole fetch so a slow or unresponsive upstream
// cannot wedge the worker.
func (m *Model) imageFetch(ctx context.Context, url string, kind ImageKind) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("bad status %d", resp.StatusCode)
	}
	return m.ImageCreate(ctx, resp.Body, kind)
}

// encodedVariant holds an encoded WebP variant in memory before
// it's committed to the blob store.
type encodedVariant struct {
	bytes  []byte
	width  int
	height int
}

// encodeVariants encodes cropped as lossy WebP at each configured
// width that doesn't require upscaling. If the cropped source's
// native width falls strictly inside the configured range — i.e.
// it's smaller than the largest configured width and isn't
// already a fitting configured width — a variant at the native
// size is added too, so the best variant we offer reflects the
// actual fidelity of the source. The output never includes a
// width larger than the largest configured width.
//
// Always returns at least one variant: when even the smallest
// configured width would upscale, the cropped source at its
// native size is returned as a single fallback variant.
func encodeVariants(cropped image.Image, kind ImageKind) []encodedVariant {
	cw := cropped.Bounds().Dx()
	ch := cropped.Bounds().Dy()
	spec := kind.spec()

	widths := slices.Clone(spec.widths)
	widths = slices.DeleteFunc(widths, func(w int) bool { return w > cw })
	if cw < spec.widths[len(spec.widths)-1] && !slices.Contains(widths, cw) {
		widths = append(widths, cw)
		slices.Sort(widths)
	}
	if len(widths) == 0 {
		// Source is smaller than the smallest configured width.
		widths = []int{cw}
	}

	variants := make([]encodedVariant, 0, len(widths))
	for _, w := range widths {
		h := w * ch / cw
		var img image.Image = cropped
		if w != cw {
			scaled := image.NewNRGBA(image.Rect(0, 0, w, h))
			draw.CatmullRom.Scale(scaled, scaled.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)
			img = scaled
		}
		variants = append(variants, encodeWebP(img, w, h))
	}
	return variants
}

func encodeWebP(img image.Image, w, h int) encodedVariant {
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: webpQuality}); err != nil {
		// webp.Encode only fails on programmer error (bad input
		// type or quality); panicking is appropriate.
		panic(fmt.Sprintf("webp encode: %v", err))
	}
	return encodedVariant{bytes: buf.Bytes(), width: w, height: h}
}

// placeholderBlobs holds the bytes that a placeholder Image
// inserts into the database and blob store. Each kind's blobs
// are decoded and encoded once at package init from the embedded
// fallback PNGs, so the resize + webp encode cost doesn't recur
// on every Model.New (which the model package tests do per test).
type placeholderBlobs struct {
	id       string
	raw      []byte
	mime     string
	variants []encodedVariant
}

func placeholderVariants(raw []byte, kind ImageKind) []encodedVariant {
	src, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		panic(fmt.Sprintf("placeholder decode: %v", err))
	}
	return encodeVariants(centerCrop(src, kind), kind)
}

var placeholders = []*placeholderBlobs{
	{
		id:       imagePosterPlaceholderID,
		raw:      assets.PosterFallbackPNG,
		mime:     "image/png",
		variants: placeholderVariants(assets.PosterFallbackPNG, ImagePoster),
	},
	{
		id:       imageThumbnailPlaceholderID,
		raw:      assets.ThumbnailFallbackPNG,
		mime:     "image/png",
		variants: placeholderVariants(assets.ThumbnailFallbackPNG, ImageThumbnail),
	},
	{
		id:       imageBannerPlaceholderID,
		raw:      assets.BannerFallbackPNG,
		mime:     "image/png",
		variants: placeholderVariants(assets.BannerFallbackPNG, ImageBanner),
	},
}

// insertPlaceholderImages creates the placeholder Image rows
// (and their variant blobs) from embedded fallback PNGs if they
// don't already exist. Called once at boot from model.New so
// the per-kind FK DEFAULTs on parent tables always resolve.
func (m *Model) insertPlaceholderImages(ctx context.Context) error {
	for _, b := range placeholders {
		if err := m.insertPlaceholder(ctx, b); err != nil {
			return fmt.Errorf("placeholder %s: %w", b.id, err)
		}
	}
	return nil
}

// insertPlaceholder is idempotent: if a row with b.id already
// exists it returns nil without touching the store or the
// Image / ImageRendition tables.
func (m *Model) insertPlaceholder(ctx context.Context, b *placeholderBlobs) error {
	err := m.WithTxR(ctx, func(tx *TxR) error {
		_, err := tx.q.ImageGet(b.id)
		return err
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = m.insertImage(ctx, b.id, b.mime, b.raw, b.variants)
	return err
}

// imageDelete deletes an Image and its rendition rows + blobs
// unconditionally. Callers are responsible for not invoking this
// on a placeholder. The cleanup is split between SQL (rows
// deleted in this transaction) and storage (blob keys removed
// in an onCommit hook so a rolled-back tx doesn't lose the
// bytes).
//
// Each user-uploaded image has exactly one owner — when a parent
// row's image FK is updated, the previous image is no longer
// reachable from anywhere and can be removed without ref
// counting.
func (tx *TxRW) imageDelete(imageID string) error {
	renditionKeys, err := tx.q.ImageRenditionDeleteByImageID(imageID)
	if err != nil {
		return err
	}
	imageKey, err := tx.q.ImageDelete(imageID)
	if err != nil {
		return err
	}
	tx.onCommit(func() {
		for _, k := range renditionKeys {
			tx.m.store.Remove(k)
		}
		tx.m.store.Remove(imageKey)
	})
	return nil
}

// centerCrop returns the largest rectangle of src whose aspect
// ratio matches kind, anchored at src's center, copied into a
// fresh zero-origin NRGBA. We deliberately copy rather than
// return a SubImage: gen2brain/webp's encoder mishandles
// non-zero-origin bounds (it reads Pix[] without offsetting by
// Bounds().Min) and produces visibly corrupted output.
func centerCrop(src image.Image, kind ImageKind) *image.NRGBA {
	spec := kind.spec()
	tw, th := spec.aspectW, spec.aspectH
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
