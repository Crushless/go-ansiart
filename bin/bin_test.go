package bin

import (
	"strings"
	"testing"

	"org.kremm/go-ansiart/color_modes"
	"org.kremm/go-ansiart/shared"
)

func TestDecodeBINRendersRawCells(t *testing.T) {
	data := []byte{
		'A', 0x01,
		0xdb, 0x4f,
		'C', 0x20,
		'D', 0xf0,
	}

	got, err := DecodeBIN(data, shared.Options{ColorMode: color_modes.NewMonochromeMode(), Columns: 2})
	if err != nil {
		t.Fatalf("decode bin: %v", err)
	}

	if got != "A█\nCD" {
		t.Fatalf("decoded bin = %q, want %q", got, "A█\nCD")
	}
}

func TestDecodeBINUsesAttributes(t *testing.T) {
	data := []byte{
		'A', 0x01,
		0xdb, 0x4f,
	}

	got, err := DecodeBIN(data, shared.Options{Columns: 2})
	if err != nil {
		t.Fatalf("decode bin: %v", err)
	}

	if !strings.Contains(got, "\x1b[38;2;0;0;170m\x1b[49mA") {
		t.Fatalf("decoded bin missing blue foreground A: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;255;255;255m\x1b[48;2;170;0;0m█") {
		t.Fatalf("decoded bin missing colored block: %q", got)
	}
}

func TestDecodeBINRejectsOddDataLength(t *testing.T) {
	_, err := DecodeBIN([]byte{'A'}, shared.Options{Columns: 1})
	if err == nil {
		t.Fatalf("expected odd data length error")
	}
}

func TestDecodeBINRejectsPartialRows(t *testing.T) {
	_, err := DecodeBIN([]byte{'A', 0x07, 'B', 0x07, 'C', 0x07}, shared.Options{Columns: 2})
	if err == nil {
		t.Fatalf("expected partial row error")
	}
}
