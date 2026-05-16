package bin

import (
	"fmt"
	"io"

	"org.kremm/go-ansiart/shared"
)

const binDefaultColumns = 80

// Parse decodes a raw DOS textmode BIN byte stream into the common parsed image model.
func Parse(data []byte, options ...shared.Options) (*shared.Image, error) {
	usedOptions := shared.NormalizeOptions(options...)
	if len(data)%shared.CellLen != 0 {
		return nil, fmt.Errorf("bin data length must be a multiple of %d: %d", shared.CellLen, len(data))
	}
	width := usedOptions.Columns
	if width <= 0 {
		width = binDefaultColumns
	}
	cellCount := len(data) / shared.CellLen
	if cellCount%width != 0 {
		return nil, fmt.Errorf("bin cell count %d is not divisible by width %d", cellCount, width)
	}

	cells := make([]shared.Cell, cellCount)
	for i := 0; i < cellCount; i++ {
		offset := i * shared.CellLen
		cells[i] = shared.Cell{
			Char: data[offset],
			Attr: data[offset+1],
		}
	}
	return &shared.Image{
		Width:   width,
		Height:  cellCount / width,
		Cells:   cells,
		Palette: shared.DefaultPalette,
	}, nil
}

// DecodeBIN decodes a raw DOS textmode BIN byte stream and renders it with the selected options.
func DecodeBIN(data []byte, options ...shared.Options) (string, error) {
	usedOptions := shared.NormalizeOptions(options...)
	image, err := Parse(data, usedOptions)
	if err != nil {
		return "", err
	}
	return shared.Render(image, usedOptions), nil
}

// DecodeBINReader is a convenience wrapper around DecodeBIN that reads BIN data from an io.Reader.
func DecodeBINReader(r io.Reader, options ...shared.Options) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read bin: %w", err)
	}
	return DecodeBIN(data, options...)
}
