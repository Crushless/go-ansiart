package xbin

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/go-restruct/restruct"
	"org.kremm/go-ansiart/color_modes"
	"org.kremm/go-ansiart/shared"
)

const (
	xBinHeaderLen        = 11
	xBinPaletteLen       = 48
	xBinCellLen          = shared.CellLen
	xBinFlagPalette      = 0x01
	xBinFlagFont         = 0x02
	xBinFlagCompress     = 0x04
	xBinFlagNonBlink     = 0x08
	xBinFlagExtendedFont = 0x10
)

type xBin struct {
	Header    xBinHeader
	Palette   xBinPalette
	FontData  []byte
	ImageData []byte
}

type xBinHeader struct {
	ID       string `struct:"[4]byte"`
	EofChar  byte   `struct:"byte"`
	Width    int16  `struct:"int16"`
	Height   int16  `struct:"int16"`
	FontSize int8   `struct:"int8"`
	Flags    byte   `struct:"byte"`
}

type xBinPalette struct {
	XBinColors [16]xBinColor
}

type xBinColor struct {
	R byte `struct:"byte"`
	G byte `struct:"byte"`
	B byte `struct:"byte"`
}

// Parse decodes XBin image data into the common parsed image model.
func Parse(data []byte, options ...shared.Options) (*shared.Image, error) {
	var xb xBin
	if err := restruct.Unpack(data, binary.LittleEndian, &xb); err != nil {
		return nil, err
	}
	return xb.image(), nil
}

// Decode decodes XBin image data and renders it as terminal text.
func Decode(data []byte, options ...shared.Options) (string, error) {
	usedOptions := shared.NormalizeOptions(options...)
	image, err := Parse(data, usedOptions)
	if err != nil {
		return "", err
	}
	return shared.Render(image, usedOptions), nil
}

// DecodeReader is a convenience wrapper around Decode that reads XBin data from an io.Reader.
func DecodeReader(r io.Reader, options ...shared.Options) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return Decode(data, options...)
}

func (xb *xBin) Unpack(buf []byte, order binary.ByteOrder) ([]byte, error) {
	if len(buf) < xBinHeaderLen {
		return nil, fmt.Errorf("xbin header: need %d bytes, got %d", xBinHeaderLen, len(buf))
	}
	if err := restruct.Unpack(buf[:xBinHeaderLen], order, &xb.Header); err != nil {
		return nil, err
	}
	buf = buf[xBinHeaderLen:]

	if xb.Header.Flags&xBinFlagPalette != 0 {
		if len(buf) < xBinPaletteLen {
			return nil, fmt.Errorf("xbin palette: need %d bytes, got %d", xBinPaletteLen, len(buf))
		}
		if err := restruct.Unpack(buf[:xBinPaletteLen], order, &xb.Palette); err != nil {
			return nil, err
		}
		buf = buf[xBinPaletteLen:]
	}

	if xb.Header.Flags&xBinFlagFont != 0 {
		fontLen, err := xb.fontLen()
		if err != nil {
			return nil, err
		}
		if len(buf) < fontLen {
			return nil, fmt.Errorf("xbin font: need %d bytes, got %d", fontLen, len(buf))
		}
		xb.FontData = append(xb.FontData[:0], buf[:fontLen]...)
		buf = buf[fontLen:]
	}

	imageLen, err := xb.imageDataLen()
	if err != nil {
		return nil, err
	}
	if xb.Header.Flags&xBinFlagCompress != 0 {
		imageData, rest, err := decompressImageData(buf, imageLen)
		if err != nil {
			return nil, err
		}
		xb.ImageData = append(xb.ImageData[:0], imageData...)
		return rest, nil
	}

	if len(buf) < imageLen {
		return nil, fmt.Errorf("xbin image data: need %d bytes, got %d", imageLen, len(buf))
	}
	xb.ImageData = append(xb.ImageData[:0], buf[:imageLen]...)
	return buf[imageLen:], nil
}

func (xb *xBin) fontLen() (int, error) {
	if xb.Header.FontSize <= 0 {
		return 0, fmt.Errorf("xbin font height must be positive: %d", xb.Header.FontSize)
	}
	charCount := 256
	if xb.Header.Flags&xBinFlagExtendedFont != 0 {
		charCount = 512
	}
	return int(xb.Header.FontSize) * charCount, nil
}

func (xb *xBin) imageDataLen() (int, error) {
	width := int(xb.Header.Width)
	height := int(xb.Header.Height)
	if width < 0 || height < 0 {
		return 0, fmt.Errorf("xbin image dimensions must be non-negative: %dx%d", width, height)
	}
	return width * height * xBinCellLen, nil
}

func (xb *xBin) image() *shared.Image {
	width := int(xb.Header.Width)
	height := int(xb.Header.Height)
	cells := make([]shared.Cell, width*height)
	for i := 0; i < len(cells); i++ {
		offset := i * xBinCellLen
		cells[i] = shared.Cell{
			Char: xb.ImageData[offset],
			Attr: xb.ImageData[offset+1],
		}
	}
	image := &shared.Image{
		Width:          width,
		Height:         height,
		Cells:          cells,
		Palette:        xb.palette(),
		HighBackground: xb.Header.Flags&xBinFlagNonBlink != 0,
	}
	if xb.Header.Flags&xBinFlagFont != 0 {
		image.Font = &shared.Font{
			Width:    8,
			Height:   int(xb.Header.FontSize),
			Extended: xb.Header.Flags&xBinFlagExtendedFont != 0,
			Glyphs:   append([]byte(nil), xb.FontData...),
		}
	}
	return image
}

func decompressImageData(buf []byte, imageLen int) ([]byte, []byte, error) {
	if imageLen%xBinCellLen != 0 {
		return nil, nil, fmt.Errorf("xbin image data length must be a multiple of %d: %d", xBinCellLen, imageLen)
	}

	out := make([]byte, 0, imageLen)
	for len(out) < imageLen {
		if len(buf) == 0 {
			return nil, nil, fmt.Errorf("xbin compressed image data ended after %d of %d bytes", len(out), imageLen)
		}

		control := buf[0]
		buf = buf[1:]
		mode := control >> 6
		cellCount := int(control&0x3f) + 1
		if cellCount > (imageLen-len(out))/xBinCellLen {
			return nil, nil, fmt.Errorf("xbin compressed run expands past image data length")
		}

		switch mode {
		case 0:
			byteCount := cellCount * xBinCellLen
			if len(buf) < byteCount {
				return nil, nil, fmt.Errorf("xbin compressed literal run: need %d bytes, got %d", byteCount, len(buf))
			}
			out = append(out, buf[:byteCount]...)
			buf = buf[byteCount:]
		case 1:
			if len(buf) < 1+cellCount {
				return nil, nil, fmt.Errorf("xbin compressed character run: need %d bytes, got %d", 1+cellCount, len(buf))
			}
			character := buf[0]
			attrs := buf[1 : 1+cellCount]
			for _, attr := range attrs {
				out = append(out, character, attr)
			}
			buf = buf[1+cellCount:]
		case 2:
			if len(buf) < 1+cellCount {
				return nil, nil, fmt.Errorf("xbin compressed attribute run: need %d bytes, got %d", 1+cellCount, len(buf))
			}
			attr := buf[0]
			chars := buf[1 : 1+cellCount]
			for _, character := range chars {
				out = append(out, character, attr)
			}
			buf = buf[1+cellCount:]
		case 3:
			if len(buf) < xBinCellLen {
				return nil, nil, fmt.Errorf("xbin compressed character/attribute run: need %d bytes, got %d", xBinCellLen, len(buf))
			}
			character := buf[0]
			attr := buf[1]
			for range cellCount {
				out = append(out, character, attr)
			}
			buf = buf[xBinCellLen:]
		}
	}

	return out, buf, nil
}

func (xb *xBin) palette() [16]color_modes.Color {
	if xb.Header.Flags&xBinFlagPalette == 0 {
		return shared.DefaultPalette
	}

	var palette [16]color_modes.Color
	for i, color := range xb.Palette.XBinColors {
		palette[i] = color_modes.Color{
			R: vgaColorToANSI(color.R),
			G: vgaColorToANSI(color.G),
			B: vgaColorToANSI(color.B),
		}
	}
	return palette
}

func vgaColorToANSI(v byte) byte {
	if v > 63 {
		return v
	}
	return byte((uint16(v) * 255) / 63)
}
