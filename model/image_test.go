package model

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"ily.dev/act3/storage"
)

// TestImageCreateNRGBARoundTrip is a regression test for a
// gen2brain/webp encoder bug that produced visibly corrupted output
// when the input *image.NRGBA had non-zero-origin bounds — which is
// exactly what centerCrop's old SubImage path produced. The fix
// copies the cropped region into a fresh zero-origin NRGBA before
// encoding; this test asserts the round-trip is faithful.
func TestImageCreateNRGBARoundTrip(t *testing.T) {
	dir := t.TempDir()

	// 600x600 NRGBA with four solid-color quadrants. Solid colors are
	// near-lossless under WebP q=80, so any pixel-shift bug shows up
	// as a wrong hue and the per-channel error explodes.
	const W, H = 600, 600
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
	// decode → centerCrop → webp.Encode path.
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, src); err != nil {
		t.Fatal(err)
	}

	storeDir := filepath.Join(dir, "cas")
	if err := os.Mkdir(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sd, err := storage.Open(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	m := &Model{store: sd}

	key, err := m.ImageCreate(&pngBuf, ImagePoster)
	if err != nil {
		t.Fatal(err)
	}

	// Decode the stored WebP back through the same image.Decode path
	// the rest of the codebase uses.
	r, err := sd.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got, _, err := image.Decode(r)
	if err != nil {
		t.Fatal(err)
	}

	// Build the expected center-cropped reference. With src 600x600
	// and target 2:3, src is wider than target so we keep height
	// (600) and crop width to 600*2/3 = 400, centered at x=100.
	const cw, ch = 400, 600
	const x0, y0 = 100, 0
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
	// crop, swapping a 100-column-wide stripe of red for green and
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
	t.Logf("round-trip MAE per channel = (%.2f, %.2f, %.2f)", maeR, maeG, maeB)
}

func absDiff(a, b uint32) uint64 {
	if a > b {
		return uint64(a - b)
	}
	return uint64(b - a)
}
