package shared

import (
	"image/color"
	"strings"
	"testing"
)

func TestRenderBitmapUsesEmbeddedFont(t *testing.T) {
	img := &Image{
		Width:   1,
		Height:  1,
		Palette: DefaultPalette,
		Font: &Font{
			Width:  8,
			Height: 2,
			Glyphs: make([]byte, 256*2),
		},
		Cells: []Cell{{Char: 'A', Attr: 0x12}},
	}
	img.Font.Glyphs[int('A')*2+0] = 0x80
	img.Font.Glyphs[int('A')*2+1] = 0x01

	got, err := RenderBitmap(img, Options{Opaque: true})
	if err != nil {
		t.Fatalf("render bitmap: %v", err)
	}

	fg := color.RGBA{R: 0x00, G: 0xaa, B: 0x00, A: 0xff}
	bg := color.RGBA{R: 0x00, G: 0x00, B: 0xaa, A: 0xff}
	if pixel := got.RGBAAt(0, 0); pixel != fg {
		t.Fatalf("foreground pixel = %#v, want %#v", pixel, fg)
	}
	if pixel := got.RGBAAt(7, 0); pixel != bg {
		t.Fatalf("background pixel = %#v, want %#v", pixel, bg)
	}
	if pixel := got.RGBAAt(7, 1); pixel != fg {
		t.Fatalf("foreground pixel = %#v, want %#v", pixel, fg)
	}
}

func TestRenderBitmapCanUseTransparentBackground(t *testing.T) {
	img := &Image{
		Width:   1,
		Height:  1,
		Palette: DefaultPalette,
		Font: &Font{
			Width:  8,
			Height: 1,
			Glyphs: make([]byte, 256),
		},
		Cells: []Cell{{Char: 'A', Attr: 0x07}},
	}
	img.Font.Glyphs['A'] = 0x80

	got, err := RenderBitmap(img, Options{})
	if err != nil {
		t.Fatalf("render bitmap: %v", err)
	}

	if pixel := got.RGBAAt(1, 0); pixel.A != 0 {
		t.Fatalf("background pixel alpha = %d, want transparent", pixel.A)
	}
}

func TestRenderBitmapUsesDefaultFontWhenImageHasNoFont(t *testing.T) {
	img := &Image{
		Width:   1,
		Height:  1,
		Palette: DefaultPalette,
		Cells:   []Cell{{Char: 'A', Attr: 0x07}},
	}

	got, err := RenderBitmap(img, Options{})
	if err != nil {
		t.Fatalf("render bitmap: %v", err)
	}

	if got.Bounds().Dx() != 8 || got.Bounds().Dy() != 16 {
		t.Fatalf("bitmap bounds = %v, want 8x16", got.Bounds())
	}

	hasInk := false
	for y := 0; y < got.Bounds().Dy(); y++ {
		for x := 0; x < got.Bounds().Dx(); x++ {
			if got.RGBAAt(x, y).A != 0 {
				hasInk = true
			}
		}
	}
	if !hasInk {
		t.Fatalf("default font rendered no visible pixels")
	}
}

func TestRenderKittyWrapsPNG(t *testing.T) {
	img := bitmapFixture()
	got, err := RenderKitty(img, Options{Opaque: true})
	if err != nil {
		t.Fatalf("render kitty: %v", err)
	}

	if !strings.HasPrefix(got, "\x1b_Gf=100,a=T,m=0;") || !strings.HasSuffix(got, "\x1b\\") {
		t.Fatalf("kitty output has unexpected framing: %q", got)
	}
}

func TestRenderSixelWrapsImage(t *testing.T) {
	img := bitmapFixture()
	got, err := RenderSixel(img, Options{Opaque: true})
	if err != nil {
		t.Fatalf("render sixel: %v", err)
	}

	if !strings.HasPrefix(got, "\x1bPq") || !strings.HasSuffix(got, "\x1b\\") {
		t.Fatalf("sixel output has unexpected framing: %q", got)
	}
}

func bitmapFixture() *Image {
	img := &Image{
		Width:   1,
		Height:  1,
		Palette: DefaultPalette,
		Font: &Font{
			Width:  8,
			Height: 1,
			Glyphs: make([]byte, 256),
		},
		Cells: []Cell{{Char: 'A', Attr: 0x07}},
	}
	img.Font.Glyphs['A'] = 0xff
	return img
}
