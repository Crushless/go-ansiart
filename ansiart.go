package ansiart

import (
	"image"
	"io"

	ansiparser "github.com/Crushless/go-ansiart/ansi"
	binparser "github.com/Crushless/go-ansiart/bin"
	"github.com/Crushless/go-ansiart/shared"
	xbinparser "github.com/Crushless/go-ansiart/xbin"
)

type Options = shared.Options
type Image = shared.Image
type Cell = shared.Cell
type Font = shared.Font

const (
	AnsiSub = shared.AnsiSub
	CellLen = shared.CellLen
)

var DefaultPalette = shared.DefaultPalette

func Render(image *Image, options Options) string {
	return shared.Render(image, options)
}

func RenderBitmap(image *Image, options Options) (*image.RGBA, error) {
	return shared.RenderBitmap(image, options)
}

func RenderKitty(image *Image, options Options) (string, error) {
	return shared.RenderKitty(image, options)
}

func RenderSixel(image *Image, options Options) (string, error) {
	return shared.RenderSixel(image, options)
}

func ParseANSI(data []byte, options ...Options) (*Image, error) {
	return ansiparser.Parse(data, options...)
}

func DecodeANSI(data []byte, options ...Options) (string, error) {
	return ansiparser.DecodeANSI(data, options...)
}

func DecodeANSIReader(r io.Reader, options ...Options) (string, error) {
	return ansiparser.DecodeANSIReader(r, options...)
}

func ParseBIN(data []byte, options ...Options) (*Image, error) {
	return binparser.Parse(data, options...)
}

func DecodeBIN(data []byte, options ...Options) (string, error) {
	return binparser.DecodeBIN(data, options...)
}

func DecodeBINReader(r io.Reader, options ...Options) (string, error) {
	return binparser.DecodeBINReader(r, options...)
}

func Parse(data []byte, options ...Options) (*Image, error) {
	return xbinparser.Parse(data, options...)
}

func Decode(data []byte, options ...Options) (string, error) {
	return xbinparser.Decode(data, options...)
}

func DecodeReader(r io.Reader, options ...Options) (string, error) {
	return xbinparser.DecodeReader(r, options...)
}
