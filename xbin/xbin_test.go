package xbin

import (
	"bytes"
	"encoding/binary"

	"strings"
	"testing"

	"github.com/Crushless/go-ansiart/char_sets"
	"github.com/Crushless/go-ansiart/color_modes"
	"github.com/Crushless/go-ansiart/shared"
	"github.com/go-restruct/restruct"
)

func TestXBinImageDataUsesCellByteLength(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x02, 0x00,
		0x02, 0x00,
		0x10,
		0x00,
		0x00, 0x01,
		0x02, 0x03,
		0x04, 0x05,
		0x06, 0x07,
	}

	var xb xBin
	if err := restruct.Unpack(data, binary.LittleEndian, &xb); err != nil {
		t.Fatalf("unpack xbin: %v", err)
	}

	if got, want := len(xb.ImageData), int(xb.Header.Width)*int(xb.Header.Height)*2; got != want {
		t.Fatalf("image data length = %d, want %d", got, want)
	}
}

func TestXBinOptionalPaletteAndFontSectionsCanBeOmitted(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x02, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		'A', 0x01,
		'B', 0x02,
	}

	var xb xBin
	if err := restruct.Unpack(data, binary.LittleEndian, &xb); err != nil {
		t.Fatalf("unpack xbin without optional sections: %v", err)
	}

	if got, want := xb.ImageData, []byte{'A', 0x01, 'B', 0x02}; !bytes.Equal(got, want) {
		t.Fatalf("image data = %v, want %v", got, want)
	}
}

func TestXBinCompressedImageData(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x0a, 0x00,
		0x01, 0x00,
		0x10,
		xBinFlagCompress,
		0x01, 'A', 0x01, 'B', 0x02,
		0x42, 'C', 0x03, 0x04, 0x05,
		0x81, 0x06, 'D', 'E',
		0xc2, 'F', 0x07,
	}
	want := []byte{
		'A', 0x01,
		'B', 0x02,
		'C', 0x03,
		'C', 0x04,
		'C', 0x05,
		'D', 0x06,
		'E', 0x06,
		'F', 0x07,
		'F', 0x07,
		'F', 0x07,
	}

	var xb xBin
	if err := restruct.Unpack(data, binary.LittleEndian, &xb); err != nil {
		t.Fatalf("unpack compressed xbin: %v", err)
	}

	if !bytes.Equal(xb.ImageData, want) {
		t.Fatalf("image data = %v, want %v", xb.ImageData, want)
	}
}

func TestDecompressXBinImageDataReturnsRemainder(t *testing.T) {
	got, rest, err := decompressImageData([]byte{0xc1, 'A', 0x07, 0xff}, 4)
	if err != nil {
		t.Fatalf("decompress image data: %v", err)
	}

	if want := []byte{'A', 0x07, 'A', 0x07}; !bytes.Equal(got, want) {
		t.Fatalf("image data = %v, want %v", got, want)
	}
	if want := []byte{0xff}; !bytes.Equal(rest, want) {
		t.Fatalf("remainder = %v, want %v", rest, want)
	}
}

func TestDecompressXBinImageDataRejectsOverlongRun(t *testing.T) {
	_, _, err := decompressImageData([]byte{0xc1, 'A', 0x07}, 2)
	if err == nil {
		t.Fatalf("expected overlong run error")
	}
}

func TestXBinSkipsOptionalFontSectionWhenPresent(t *testing.T) {
	data := xBinFixtureWithPaletteAndFont()

	var xb xBin
	if err := restruct.Unpack(data, binary.LittleEndian, &xb); err != nil {
		t.Fatalf("unpack xbin: %v", err)
	}

	if got, want := len(xb.ImageData), int(xb.Header.Width)*int(xb.Header.Height)*2; got != want {
		t.Fatalf("image data length = %d, want %d", got, want)
	}
	if got, want := xb.ImageData, []byte{'A', 0x21, 0xdb, 0x03}; !bytes.Equal(got, want) {
		t.Fatalf("image data = %v, want %v", got, want)
	}
	if got, want := len(xb.FontData), 2*256; got != want {
		t.Fatalf("font data length = %d, want %d", got, want)
	}
}

func TestParseXBinPreservesEmbeddedFont(t *testing.T) {
	data := xBinFixtureWithPaletteAndFont()

	img, err := Parse(data)
	if err != nil {
		t.Fatalf("parse xbin: %v", err)
	}

	if img.Font == nil {
		t.Fatalf("parsed xbin missing embedded font")
	}
	if got, want := img.Font.Height, 2; got != want {
		t.Fatalf("font height = %d, want %d", got, want)
	}
	if got, want := len(img.Font.Glyphs), 2*256; got != want {
		t.Fatalf("font glyph data length = %d, want %d", got, want)
	}
}

func TestDecodeXBinReturnsANSIText(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x02, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		'A', 0x01,
		0xdb, 0x4f,
	}

	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;2;0;0;170m\x1b[49mA") {
		t.Fatalf("decoded output missing blue foreground A: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;255;255;255m\x1b[48;2;170;0;0m█") {
		t.Fatalf("decoded output missing colored block: %q", got)
	}
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Fatalf("decoded output should reset colors, got %q", got)
	}
}

func TestDecodeXBinCanFillSolidBlocks(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x01, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		0xdb, 0x4f,
	}

	got, err := Decode(data, shared.Options{FillSolidBlocks: true})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, "\x1b[7m \x1b[27m") {
		t.Fatalf("decoded xbin should render full block as reverse-video space: %q", got)
	}
	if strings.Contains(got, "█") {
		t.Fatalf("decoded xbin should not emit full block glyph when solid block filling is enabled: %q", got)
	}
}

func TestDecodeXBinCanDisableAutoWrap(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x01, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		'A', 0x07,
	}

	got, err := Decode(data, shared.Options{
		ColorMode:       color_modes.NewMonochromeMode(),
		DisableAutoWrap: true,
	})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if got != "\x1b[?7lA\x1b[?7h" {
		t.Fatalf("decoded xbin = %q, want autowrap disabled around output", got)
	}
}

func TestDecodeXBinCanStabilizeColumns(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x02, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		'A', 0x07,
		'B', 0x07,
	}

	got, err := Decode(data, shared.Options{
		ColorMode:     color_modes.NewMonochromeMode(),
		StableColumns: true,
	})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if got != "A\x1b[2GB" {
		t.Fatalf("decoded xbin = %q, want cursor repositioned after first cell", got)
	}
}

func TestDecodeXBinSwitchesANSIAttributesOncePerRun(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x02, 0x00,
		0x02, 0x00,
		0x10,
		0x00,
		'A', 0x01,
		'B', 0x01,
		'C', 0x01,
		'D', 0x01,
	}

	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if got, want := strings.Count(got, "\x1b[38;2;"), 1; got != want {
		t.Fatalf("foreground ansi switch count = %d, want %d", got, want)
	}
	if got, want := strings.Count(got, "\x1b[49m"), 1; got != want {
		t.Fatalf("transparent background switch count = %d, want %d", got, want)
	}
	if !strings.Contains(got, "AB\nCD") {
		t.Fatalf("decoded output missing expected text: %q", got)
	}
}

func TestDecodeXBinResetsBackgroundBeforeLineBreak(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x01, 0x00,
		0x02, 0x00,
		0x10,
		0x00,
		'A', 0x41,
		'B', 0x41,
	}

	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, "A\x1b[0m\n\x1b[38;2;0;0;170m\x1b[48;2;170;0;0mB") {
		t.Fatalf("decoded output should reset background before newline and reapply after it: %q", got)
	}
}

func TestDecodeXBinUsesDefaultBackgroundForPaletteIndexZero(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x02, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		'A', 0x01,
		'B', 0x11,
	}

	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;2;0;0;170m\x1b[49mA") {
		t.Fatalf("palette index 0 background should use terminal default: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;0;0;170m\x1b[48;2;0;0;170mB") {
		t.Fatalf("non-zero background should use palette color: %q", got)
	}
}

func TestDecodeXBinCanUseIndexedANSIColors(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x02, 0x00,
		0x01, 0x00,
		0x10,
		xBinFlagNonBlink,
		'A', 0x01,
		'B', 0x9f,
	}

	got, err := Decode(data, shared.Options{ColorMode: color_modes.NewIndexedANSIMode()})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;5;4m\x1b[49mA") {
		t.Fatalf("palette index 0 background should use terminal default: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;5;15m\x1b[48;5;12mB") {
		t.Fatalf("non-zero background should use indexed color escape: %q", got)
	}
	if strings.Contains(got, "\x1b[38;2;") || strings.Contains(got, "\x1b[48;2;") {
		t.Fatalf("indexed mode should not emit truecolor escapes: %q", got)
	}
}

func TestDecodeXBinCanUseMonochromeMode(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x02, 0x00,
		0x02, 0x00,
		0x10,
		0x00,
		'A', 0x01,
		0xdb, 0x4f,
		'C', 0x20,
		'D', 0xf0,
	}

	got, err := Decode(data, shared.Options{ColorMode: color_modes.NewMonochromeMode()})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if got != "A█\nCD" {
		t.Fatalf("monochrome output = %q, want %q", got, "A█\nCD")
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("monochrome mode should not emit ansi escapes: %q", got)
	}
}

func TestDecodeXBinCanUseMonochromeHalfBrightMode(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x04, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		'A', 0x01,
		'B', 0x02,
		'C', 0x0f,
		'D', 0x08,
	}

	got, err := Decode(data, shared.Options{ColorMode: color_modes.NewHalfBrightMode(color_modes.HalfBrightByPaletteColumn)})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	want := "\x1b[2mAB\x1b[22mCD"
	if got != want {
		t.Fatalf("monochrome half-bright output = %q, want %q", got, want)
	}
}

func TestDecodeXBinMonochromeHalfBrightUsesMoebiusPaletteColumns(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x04, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		'A', 0x0f,
		'B', 0x07,
		'C', 0x08,
		'D', 0x0f,
	}

	got, err := Decode(data, shared.Options{ColorMode: color_modes.NewHalfBrightMode(color_modes.HalfBrightByPaletteColumn)})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	want := "A\x1b[2mB\x1b[22mCD"
	if got != want {
		t.Fatalf("monochrome half-bright output = %q, want %q", got, want)
	}
}

func TestDecodeXBinMonochromeHalfBrightCanUseLuminance(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x05, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		'A', 0x0f,
		'B', 0x07,
		'C', 0x08,
		'D', 0x0f,
		'E', 0x09,
	}

	got, err := Decode(data, shared.Options{
		ColorMode: color_modes.NewHalfBrightMode(color_modes.HalfBrightByLuminance),
	})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	want := "AB\x1b[2mC\x1b[22mDE"
	if got != want {
		t.Fatalf("monochrome half-bright output = %q, want %q", got, want)
	}
}

func TestHalfBrightLuminanceKeepsRightColumnBrightExceptDarkGray(t *testing.T) {
	var xb xBin
	palette := xb.palette()

	for index := byte(9); index < 16; index++ {
		if color_modes.NewHalfBrightMode(color_modes.HalfBrightByLuminance).ColorMapping(color_modes.Context{Palette: palette}).Style(index) == "faint" {
			t.Fatalf("palette index %#x should be bright in luminance mode", index)
		}
	}
	if color_modes.NewHalfBrightMode(color_modes.HalfBrightByLuminance).ColorMapping(color_modes.Context{Palette: palette}).Style(8) != "faint" {
		t.Fatalf("palette index 8 should remain half-bright in luminance mode")
	}
}

func TestDecodeXBinMonochromeHalfBrightLuminanceUsesEmbeddedPalette(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x01, 0x00,
		0x01, 0x00,
		0x10,
		xBinFlagPalette,
	}
	palette := make([]byte, xBinPaletteLen)
	palette[1*3+0] = 63
	palette[1*3+1] = 63
	palette[1*3+2] = 63
	data = append(data, palette...)
	data = append(data, 'A', 0x01)

	got, err := Decode(data, shared.Options{
		ColorMode: color_modes.NewHalfBrightMode(color_modes.HalfBrightByLuminance),
	})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if got != "A" {
		t.Fatalf("monochrome half-bright should use embedded palette luminance: %q", got)
	}
}

func TestDecodeXBinCanRenderPaletteIndexZeroAsOpaque(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x01, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		'A', 0x01,
	}

	got, err := Decode(data, shared.Options{Opaque: true})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;2;0;0;170m\x1b[48;2;0;0;0mA") {
		t.Fatalf("palette index 0 background should be opaque: %q", got)
	}
}

func TestXBinRuneMapsCP437(t *testing.T) {
	tests := map[byte]rune{
		0x01: '☺',
		0x7f: '⌂',
		0xb0: '░',
		0xb3: '│',
		0xc4: '─',
		0xdb: '█',
		0xe0: 'α',
		0xf1: '±',
		0xfe: '■',
	}

	for b, want := range tests {
		if got := char_sets.CP437[b]; got != want {
			t.Fatalf("XBinCP437[%#02x] = %q, want %q", b, got, want)
		}
	}
}

func TestDecodeXBinCanUsePETSCIICharset(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x04, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		0x00, 0x07,
		0x01, 0x07,
		0x1c, 0x07,
		0x5e, 0x07,
	}

	got, err := Decode(data, shared.Options{ColorMode: color_modes.NewIndexedANSIMode(), Charset: char_sets.PETSCII})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, "@A£▗") {
		t.Fatalf("decoded output missing PETSCII characters: %q", got)
	}
}

func TestDecodeXBinCanUseAmigaCharset(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x13, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		0x00, 0x07,
		0x01, 0x07,
		0x1a, 0x07,
		0x1b, 0x07,
		0x1c, 0x07,
		0x1d, 0x07,
		0x1e, 0x07,
		0x1f, 0x07,
		0x41, 0x07,
		0x7f, 0x07,
		0x80, 0x07,
		0x8f, 0x07,
		0x90, 0x07,
		0x9f, 0x07,
		0xa3, 0x07,
		0xaa, 0x07,
		0xad, 0x07,
		0xba, 0x07,
		0xe4, 0x07,
	}

	got, err := Decode(data, shared.Options{ColorMode: color_modes.NewIndexedANSIMode(), Charset: char_sets.AmigaTopaz})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, " AZ[\\]^_A▧ÀÏÐß£ª-ºä") {
		t.Fatalf("decoded output missing Amiga characters: %q", got)
	}
}

func TestDecodeXBinDefaultsToCP437Charset(t *testing.T) {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x01, 0x00,
		0x01, 0x00,
		0x10,
		0x00,
		0x01, 0x07,
	}

	got, err := Decode(data, shared.Options{ColorMode: color_modes.NewIndexedANSIMode()})
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, "☺") {
		t.Fatalf("decoded output should use CP437 by default: %q", got)
	}
}

func TestDecodeXBinUsesEmbeddedPaletteInTrueColorMode(t *testing.T) {
	got, err := Decode(xBinFixtureWithPaletteAndFont())
	if err != nil {
		t.Fatalf("decode xbin: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;2;255;0;0m\x1b[48;2;0;255;0mA") {
		t.Fatalf("decoded xbin missing embedded palette colors: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;0;0;255m\x1b[49m█") {
		t.Fatalf("decoded xbin missing embedded palette transparent background: %q", got)
	}
}

func xBinFixtureWithPaletteAndFont() []byte {
	data := []byte{
		'X', 'B', 'I', 'N',
		0x1a,
		0x02, 0x00,
		0x01, 0x00,
		0x02,
		xBinFlagPalette | xBinFlagFont,
	}

	palette := make([]byte, xBinPaletteLen)
	palette[1*3+0] = 63
	palette[2*3+1] = 63
	palette[3*3+2] = 63
	data = append(data, palette...)

	data = append(data, make([]byte, 2*256)...)
	data = append(data, 'A', 0x21, 0xdb, 0x03)
	return data
}
