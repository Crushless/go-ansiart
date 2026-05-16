package shared

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/binary"
	"fmt"
	"io"
)

//go:embed assets/default8x16.psfu.gz
var defaultFontPSFUGZ []byte

const psf2Magic = 0x864ab572

// DefaultFont returns the bundled 8x16 bitmap font used by bitmap renderers
// when the source image does not carry an embedded font.
func DefaultFont() (*Font, error) {
	font, err := parsePSF(defaultFontPSFUGZ)
	if err != nil {
		return nil, fmt.Errorf("load default font: %w", err)
	}
	return font, nil
}

func parsePSF(data []byte) (*Font, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(raw) < 32 {
		return nil, fmt.Errorf("psf header: need 32 bytes, got %d", len(raw))
	}
	if binary.LittleEndian.Uint32(raw[0:4]) != psf2Magic {
		return nil, fmt.Errorf("unsupported psf magic")
	}

	headerSize := int(binary.LittleEndian.Uint32(raw[8:12]))
	glyphCount := int(binary.LittleEndian.Uint32(raw[16:20]))
	charSize := int(binary.LittleEndian.Uint32(raw[20:24]))
	height := int(binary.LittleEndian.Uint32(raw[24:28]))
	width := int(binary.LittleEndian.Uint32(raw[28:32]))
	if headerSize < 32 || headerSize > len(raw) {
		return nil, fmt.Errorf("invalid psf header size: %d", headerSize)
	}
	if glyphCount <= 0 || charSize <= 0 || height <= 0 || width <= 0 {
		return nil, fmt.Errorf("invalid psf metrics")
	}
	if width > 8 {
		return nil, fmt.Errorf("unsupported psf width: %d", width)
	}

	glyphBytes := glyphCount * charSize
	if headerSize+glyphBytes > len(raw) {
		return nil, fmt.Errorf("psf glyph data: need %d bytes, got %d", glyphBytes, len(raw)-headerSize)
	}

	glyphs := append([]byte(nil), raw[headerSize:headerSize+glyphBytes]...)
	return &Font{
		Width:  width,
		Height: height,
		Glyphs: glyphs,
	}, nil
}
