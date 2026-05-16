package shared

import (
	"fmt"
	"strings"

	"github.com/Crushless/go-ansiart/char_sets"
	"github.com/Crushless/go-ansiart/color_modes"
)

const (
	// AnsiSub is the ASCII substitute character used by XBin as the default EOF marker.
	AnsiSub = 0x1a
	// CellLen is the length of one textmode cell in source byte streams.
	CellLen = 2
)

// Options controls parsing defaults and terminal rendering behavior.
type Options struct {
	ColorMode       color_modes.ColorMode
	Charset         *char_sets.Charset
	Opaque          bool
	Columns         int
	FillSolidBlocks bool
	DisableAutoWrap bool
	StableColumns   bool
}

// Cell is one parsed textmode image cell.
type Cell struct {
	Char byte
	Attr byte
	FG   color_modes.Override
	BG   color_modes.Override
}

// Font is an 8-pixel-wide bitmap textmode font carried by formats such as XBin.
type Font struct {
	Width    int
	Height   int
	Extended bool
	Glyphs   []byte
}

// Image is the common parsed representation produced by all format parsers.
type Image struct {
	Width          int
	Height         int
	Cells          []Cell
	Palette        [16]color_modes.Color
	HighBackground bool
	Font           *Font
}

// DefaultPalette defines the default 16-color DOS palette.
var DefaultPalette = [16]color_modes.Color{
	{R: 0x00, G: 0x00, B: 0x00},
	{R: 0x00, G: 0x00, B: 0xaa},
	{R: 0x00, G: 0xaa, B: 0x00},
	{R: 0x00, G: 0xaa, B: 0xaa},
	{R: 0xaa, G: 0x00, B: 0x00},
	{R: 0xaa, G: 0x00, B: 0xaa},
	{R: 0xaa, G: 0x55, B: 0x00},
	{R: 0xaa, G: 0xaa, B: 0xaa},
	{R: 0x55, G: 0x55, B: 0x55},
	{R: 0x55, G: 0x55, B: 0xff},
	{R: 0x55, G: 0xff, B: 0x55},
	{R: 0x55, G: 0xff, B: 0xff},
	{R: 0xff, G: 0x55, B: 0x55},
	{R: 0xff, G: 0x55, B: 0xff},
	{R: 0xff, G: 0xff, B: 0x55},
	{R: 0xff, G: 0xff, B: 0xff},
}

// NormalizeOptions applies package defaults without overriding explicitly supplied values.
func NormalizeOptions(options ...Options) Options {
	used := Options{
		ColorMode: color_modes.NewTrueColorMode(),
		Charset:   char_sets.CP437,
		Opaque:    false,
	}
	for _, opts := range options {
		if opts.ColorMode != nil {
			used.ColorMode = opts.ColorMode
		}
		if opts.Charset != nil {
			used.Charset = opts.Charset
		}
		if opts.Columns > 0 {
			used.Columns = opts.Columns
		}
		used.Opaque = opts.Opaque
		used.FillSolidBlocks = opts.FillSolidBlocks
		used.DisableAutoWrap = opts.DisableAutoWrap
		used.StableColumns = opts.StableColumns
	}
	return used
}

// Render converts a parsed image into terminal text and escape sequences.
func Render(image *Image, options Options) string {
	if image == nil || image.Width <= 0 || image.Height <= 0 {
		return ""
	}

	colorMapping := options.ColorMode.ColorMapping(color_modes.Context{
		Palette:        image.Palette,
		HighBackground: image.HighBackground,
		Opaque:         options.Opaque,
	})

	var out strings.Builder
	writeRenderPrefix(&out, options)
	currentStyle := colorMapping.InitialStyle()
	for y := 0; y < image.Height; y++ {
		for x := 0; x < image.Width; x++ {
			cell := image.CellAt(x, y)
			style := colorMapping.StyleWithOverrides(cell.Attr, cell.FG, cell.BG)
			if style != currentStyle {
				out.WriteString(colorMapping.Escape(style))
				currentStyle = style
			}

			writeRenderedCharacter(&out, options, colorMapping, cell.Char)
			writeStableColumn(&out, options, x, image.Width)
			if x == image.Width-1 && y != image.Height-1 {
				if colorMapping.HasBackgroundWithOverride(cell.Attr, cell.FG, cell.BG) {
					out.WriteString(colorMapping.Reset(currentStyle))
					currentStyle = colorMapping.InitialStyle()
				}
				out.WriteByte('\n')
			}
		}
	}

	out.WriteString(colorMapping.Reset(currentStyle))
	writeRenderSuffix(&out, options)
	return out.String()
}

// CellAt returns a cell, falling back to a CP437 blank with light-gray foreground.
func (image *Image) CellAt(x int, y int) Cell {
	index := y*image.Width + x
	if index >= 0 && index < len(image.Cells) {
		return image.Cells[index]
	}
	return Cell{Char: ' ', Attr: 0x07}
}

func writeRenderPrefix(out *strings.Builder, options Options) {
	if options.DisableAutoWrap {
		out.WriteString("\x1b[?7l")
	}
}

func writeRenderSuffix(out *strings.Builder, options Options) {
	if options.DisableAutoWrap {
		out.WriteString("\x1b[?7h")
	}
}

func writeStableColumn(out *strings.Builder, options Options, x int, width int) {
	if options.StableColumns && x < width-1 {
		fmt.Fprintf(out, "\x1b[%dG", x+2)
	}
}

func writeRenderedCharacter(out *strings.Builder, options Options, colorMapping color_modes.Mapping, character byte) {
	if options.FillSolidBlocks && colorMapping.CanFillSolidBlocks() && options.Charset[character] == '█' {
		out.WriteString("\x1b[7m \x1b[27m")
		return
	}
	out.WriteRune(options.Charset[character])
}
