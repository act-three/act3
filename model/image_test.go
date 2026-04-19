package model

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"golang.org/x/image/webp"

	"ily.dev/act3/database"
	"ily.dev/act3/database/schema"
	"ily.dev/act3/storage"
)

// newTestModel returns a minimal in-memory Model suitable for
// tests that need to call ImageCreate or other methods that touch
// both the storage and the database. Placeholder images are
// installed before returning so parent inserts that rely on the
// per-kind FK DEFAULTs can succeed.
func newTestModel(t *testing.T) *Model {
	t.Helper()
	dbr, dbw, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		dbr.Close()
		dbw.Close()
	})
	sd, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := &Model{store: sd, dbr: dbr, dbw: dbw}
	if err := m.insertPlaceholderImages(context.Background()); err != nil {
		t.Fatal(err)
	}
	return m
}

// TestImageCreateNRGBARoundTrip is a regression test for a
// gen2brain/webp encoder bug that produced visibly corrupted output
// when the input *image.NRGBA had non-zero-origin bounds — which is
// exactly what centerCrop's old SubImage path produced. The fix
// copies the cropped region into a fresh zero-origin NRGBA before
// encoding; this test asserts the round-trip is faithful by
// per-pixel comparison against the largest variant.
func TestImageCreateNRGBARoundTrip(t *testing.T) {
	ctx := t.Context()

	// 900x900 NRGBA with four solid-color quadrants. Cropped to 2:3
	// this gives 600x900, which lets the poster spec generate both
	// configured variants (300 and 600) without any "native" extra
	// — the 600 variant is unscaled and we can compare it directly
	// against the cropped reference.
	const W, H = 900, 900
	src := image.NewNRGBA(image.Rect(0, 0, W, H))
	quads := [4]color.NRGBA{
		{200, 30, 30, 255},  // top-left red
		{30, 200, 30, 255},  // top-right green
		{30, 30, 200, 255},  // bottom-left blue
		{200, 200, 30, 255}, // bottom-right yellow
	}
	for y := range H {
		for x := range W {
			i := 0
			if x >= W/2 {
				i |= 1
			}
			if y >= H/2 {
				i |= 2
			}
			src.SetNRGBA(x, y, quads[i])
		}
	}

	// Encode src to PNG so ImageCreate exercises the full
	// decode → centerCrop → resize → webp.Encode path.
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, src); err != nil {
		t.Fatal(err)
	}

	m := newTestModel(t)
	originalID, err := m.ImageCreate(ctx, &pngBuf, ImagePoster)
	if err != nil {
		t.Fatal(err)
	}

	// Confirm two variants were stored at the configured widths
	// (300 and 600). The 600 variant is unscaled native, which we
	// then decode and pixel-compare against the centered crop.
	var stored []schema.ImageRendition
	if err := m.WithTxR(func(tx *TxR) error {
		stored, err = tx.q.ImageRenditionListByImageID(ctx, originalID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 {
		t.Fatalf("variant count = %d, want 2", len(stored))
	}
	largest := stored[len(stored)-1]
	if largest.Width != 600 || largest.Height != 900 {
		t.Fatalf("largest variant = %dx%d, want 600x900", largest.Width, largest.Height)
	}

	var largestKey string
	if err := m.WithTxR(func(tx *TxR) error {
		var err error
		largestKey, err = tx.ImageVariantKey(ctx, originalID, 600)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	f, err := m.store.Open(largestKey)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := webp.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	// Build the expected center-cropped reference. With src 900x900
	// and target 2:3, src is wider than target so we keep height
	// (900) and crop width to 900*2/3 = 600, centered at x=150.
	const cw, ch = 600, 900
	const x0, y0 = 150, 0
	expected := image.NewNRGBA(image.Rect(0, 0, cw, ch))
	draw.Draw(expected, expected.Bounds(), src, image.Pt(x0, y0), draw.Src)

	if b := got.Bounds(); b.Dx() != cw || b.Dy() != ch {
		t.Fatalf("decoded size = %dx%d, want %dx%d", b.Dx(), b.Dy(), cw, ch)
	}

	// Mean absolute error per RGB channel over the whole image.
	// q=80 WebP with 4:2:0 chroma subsampling bleeds a few pixels
	// across the quadrant seams, so a clean round-trip lands around
	// MAE ~10 per channel. The pre-fix non-zero-origin bug encoded
	// the top-left cw×ch region of the parent instead of the centered
	// crop, swapping a 150-column-wide stripe of red for green and
	// pushing MAE above ~40 on R and G — well clear of the threshold.
	var sumR, sumG, sumB uint64
	for y := range ch {
		for x := range cw {
			er, eg, eb, _ := expected.At(x, y).RGBA()
			gr, gg, gb, _ := got.At(x, y).RGBA()
			sumR += absDiff(er>>8, gr>>8)
			sumG += absDiff(eg>>8, gg>>8)
			sumB += absDiff(eb>>8, gb>>8)
		}
	}
	n := uint64(cw * ch)
	maeR := float64(sumR) / float64(n)
	maeG := float64(sumG) / float64(n)
	maeB := float64(sumB) / float64(n)
	const maxMAE = 25.0
	if maeR > maxMAE || maeG > maxMAE || maeB > maxMAE {
		t.Errorf("ImageCreate round-trip MAE = (%.2f, %.2f, %.2f), want <= %.2f per channel "+
			"(likely the gen2brain/webp non-zero-origin encode bug has regressed)",
			maeR, maeG, maeB, maxMAE)
	}
	im := Image{ID: originalID, Kind: ImagePoster}
	t.Logf("round-trip MAE per channel = (%.2f, %.2f, %.2f); srcset=%q",
		maeR, maeG, maeB, im.Srcset())
}

func absDiff(a, b uint32) uint64 {
	if a > b {
		return uint64(a - b)
	}
	return uint64(b - a)
}
