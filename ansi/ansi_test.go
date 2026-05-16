package ansi

import (
	"strings"
	"testing"

	"org.kremm/go-ansiart/color_modes"
	"org.kremm/go-ansiart/shared"
)

func TestDecodeANSIRendersCP437Text(t *testing.T) {
	got, err := DecodeANSI([]byte{'A', 0xdb, 'B'}, shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 80})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "A█B" {
		t.Fatalf("decoded ansi = %q, want %q", got, "A█B")
	}
}

func TestDecodeANSIUsesSGRAttributes(t *testing.T) {
	got, err := DecodeANSI([]byte("A\x1b[31mB\x1b[1mC\x1b[44mD"), shared.Options{Columns: 80})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;2;170;170;170m\x1b[49mA") {
		t.Fatalf("decoded ansi missing default colored A: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;170;0;0m\x1b[49mB") {
		t.Fatalf("decoded ansi missing red B: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;255;85;85m\x1b[49mC") {
		t.Fatalf("decoded ansi missing bright red C: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;255;85;85m\x1b[48;2;0;0;170mD") {
		t.Fatalf("decoded ansi missing bright red on blue D: %q", got)
	}
}

func TestDecodeANSIUsesTrueColorSGRAttributes(t *testing.T) {
	got, err := DecodeANSI([]byte("\x1b[38;2;1;2;3mF\x1b[48;2;4;5;6mB"), shared.Options{Columns: 80})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;2;1;2;3m\x1b[49mF") {
		t.Fatalf("decoded ansi missing truecolor foreground: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;1;2;3m\x1b[48;2;4;5;6mB") {
		t.Fatalf("decoded ansi missing truecolor background: %q", got)
	}
}

func TestDecodeANSITrueColorResetsForegroundAndBackground(t *testing.T) {
	got, err := DecodeANSI([]byte("\x1b[38;2;1;2;3mA\x1b[39mB\x1b[48;2;4;5;6mC\x1b[49mD"), shared.Options{Columns: 80})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;2;1;2;3m\x1b[49mA") {
		t.Fatalf("decoded ansi missing truecolor foreground before reset: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;170;170;170m\x1b[49mB") {
		t.Fatalf("decoded ansi missing default foreground after reset: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;170;170;170m\x1b[48;2;4;5;6mC") {
		t.Fatalf("decoded ansi missing truecolor background before reset: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;170;170;170m\x1b[49mD") {
		t.Fatalf("decoded ansi missing default background after reset: %q", got)
	}
}

func TestDecodeANSITrueColorIgnoredInMonochromeMode(t *testing.T) {
	got, err := DecodeANSI([]byte("\x1b[38;2;1;2;3mA\x1b[48;2;4;5;6mB"), shared.Options{
		ColorMode: color_modes.NewMonochromeMode(),
		Columns:   80,
	})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "AB" {
		t.Fatalf("monochrome ansi output = %q, want %q", got, "AB")
	}
}

func TestDecodeANSIPreservesIndexed256Colors(t *testing.T) {
	got, err := DecodeANSI([]byte("\x1b[38;5;200mF\x1b[48;5;201mB"), shared.Options{
		ColorMode: color_modes.NewIndexedANSIMode(),
		Columns:   80,
	})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;5;200m\x1b[49mF") {
		t.Fatalf("decoded ansi missing indexed foreground: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;5;200m\x1b[48;5;201mB") {
		t.Fatalf("decoded ansi missing indexed background: %q", got)
	}
}

func TestDecodeANSIConvertsIndexed256ColorsToTrueColor(t *testing.T) {
	got, err := DecodeANSI([]byte("\x1b[38;5;196mF\x1b[48;5;21mB"), shared.Options{Columns: 80})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;2;255;0;0m\x1b[49mF") {
		t.Fatalf("decoded ansi missing truecolor foreground converted from 256-color index: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;255;0;0m\x1b[48;2;0;0;255mB") {
		t.Fatalf("decoded ansi missing truecolor background converted from 256-color index: %q", got)
	}
}

func TestDecodeANSICanFillSolidBlocks(t *testing.T) {
	got, err := DecodeANSI([]byte{0xdb}, shared.Options{FillSolidBlocks: true})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if !strings.Contains(got, "\x1b[7m \x1b[27m") {
		t.Fatalf("decoded ansi should render full block as reverse-video space: %q", got)
	}
	if strings.Contains(got, "█") {
		t.Fatalf("decoded ansi should not emit full block glyph when solid block filling is enabled: %q", got)
	}
}

func TestDecodeANSIFillSolidBlocksDoesNotAffectMonochrome(t *testing.T) {
	got, err := DecodeANSI([]byte{0xdb}, shared.Options{
		ColorMode:       color_modes.NewMonochromeMode(),
		Columns:         80,
		FillSolidBlocks: true,
	})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "█" {
		t.Fatalf("monochrome ansi output = %q, want %q", got, "█")
	}
}

func TestDecodeANSIWrapsAtConfiguredColumns(t *testing.T) {
	got, err := DecodeANSI([]byte("ABCD"), shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 2})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "AB\nCD" {
		t.Fatalf("decoded ansi = %q, want %q", got, "AB\nCD")
	}
}

func TestDecodeANSICanDisableAutoWrap(t *testing.T) {
	got, err := DecodeANSI([]byte("AB"), shared.Options{
		ColorMode:       color_modes.NewMonochromeMode(),
		Columns:         80,
		DisableAutoWrap: true,
	})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "\x1b[?7lAB\x1b[?7h" {
		t.Fatalf("decoded ansi = %q, want autowrap disabled around output", got)
	}
}

func TestDecodeANSICanStabilizeColumns(t *testing.T) {
	got, err := DecodeANSI([]byte("AB"), shared.Options{
		ColorMode:     color_modes.NewMonochromeMode(),
		Columns:       80,
		StableColumns: true,
	})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "A\x1b[2GB" {
		t.Fatalf("decoded ansi = %q, want cursor repositioned after first cell", got)
	}
}

func TestDecodeANSIResetsBackgroundBeforeLineBreak(t *testing.T) {
	got, err := DecodeANSI([]byte("\x1b[41mA\nB"), shared.Options{Columns: 80})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if !strings.Contains(got, "A\x1b[0m\n\x1b[38;2;170;170;170m\x1b[48;2;170;0;0mB") {
		t.Fatalf("decoded ansi should reset background before newline and reapply after it: %q", got)
	}
}

func TestDecodeANSIHandlesCursorMovement(t *testing.T) {
	got, err := DecodeANSI([]byte("AB\x1b[1D C"), shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 4})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "A C" {
		t.Fatalf("decoded ansi = %q, want %q", got, "A C")
	}
}

func TestDecodeANSIRepeatsPreviousCharacter(t *testing.T) {
	got, err := DecodeANSI([]byte("A\x1b[3bB"), shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 80})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "AAAAB" {
		t.Fatalf("decoded ansi = %q, want %q", got, "AAAAB")
	}
}

func TestDecodeANSIHandlesHorizontalAbsolute(t *testing.T) {
	got, err := DecodeANSI([]byte("A\x1b[4GB"), shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 5})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "A  B" {
		t.Fatalf("decoded ansi = %q, want %q", got, "A  B")
	}
}

func TestDecodeANSICursorForwardAppliesPendingWrap(t *testing.T) {
	got, err := DecodeANSI([]byte("ABC\x1b[2CD"), shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 3})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "ABC\n  D" {
		t.Fatalf("decoded ansi = %q, want %q", got, "ABC\n  D")
	}
}

func TestDecodeANSIInsertsCharacters(t *testing.T) {
	got, err := DecodeANSI([]byte("ABCD\x1b[2D\x1b[2@XY"), shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 6})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "ABXYCD" {
		t.Fatalf("decoded ansi = %q, want %q", got, "ABXYCD")
	}
}

func TestDecodeANSIDeletesCharacters(t *testing.T) {
	got, err := DecodeANSI([]byte("ABCDE\x1b[4D\x1b[2P"), shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 5})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "ADE  " {
		t.Fatalf("decoded ansi = %q, want %q", got, "ADE  ")
	}
}

func TestDecodeANSIErasesCharacters(t *testing.T) {
	got, err := DecodeANSI([]byte("ABCDE\x1b[4D\x1b[2X"), shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 5})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "A  DE" {
		t.Fatalf("decoded ansi = %q, want %q", got, "A  DE")
	}
}

func TestDecodeANSIStripsSauce(t *testing.T) {
	data := []byte("A")
	data = append(data, 0x1a)
	data = append(data, []byte("SAUCE00")...)
	data = append(data, make([]byte, 128-len("SAUCE00"))...)

	got, err := DecodeANSI(data, shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 80})
	if err != nil {
		t.Fatalf("decode ansi: %v", err)
	}

	if got != "A" {
		t.Fatalf("decoded ansi = %q, want %q", got, "A")
	}
}
