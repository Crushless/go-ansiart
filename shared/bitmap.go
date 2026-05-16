package shared

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"org.kremm/go-ansiart/color_modes"
)

const kittyChunkSize = 4096

// RenderBitmap rasterizes a parsed textmode image with its embedded bitmap font,
// or the bundled 8x16 fallback font when the image has no font.
func RenderBitmap(img *Image, options Options) (*image.RGBA, error) {
	if img == nil || img.Width <= 0 || img.Height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0)), nil
	}
	font := img.Font
	if font == nil {
		var err error
		font, err = DefaultFont()
		if err != nil {
			return nil, err
		}
	}
	if font.Width <= 0 || font.Height <= 0 {
		return nil, fmt.Errorf("bitmap render requires a positive font size")
	}

	bounds := image.Rect(0, 0, img.Width*font.Width, img.Height*font.Height)
	out := image.NewRGBA(bounds)
	for cellY := 0; cellY < img.Height; cellY++ {
		for cellX := 0; cellX < img.Width; cellX++ {
			cell := img.CellAt(cellX, cellY)
			glyph := font.Glyph(cell)
			fg, bg, hasBG := bitmapCellColors(img, cell, options)
			for row := 0; row < font.Height; row++ {
				var bits byte
				if row < len(glyph) {
					bits = glyph[row]
				}
				for col := 0; col < font.Width; col++ {
					mask := byte(0x80 >> col)
					pixel := bg
					if bits&mask != 0 {
						pixel = fg
					} else if !hasBG {
						pixel.A = 0
					}
					out.SetRGBA(cellX*font.Width+col, cellY*font.Height+row, pixel)
				}
			}
		}
	}
	return out, nil
}

// RenderKitty rasterizes an image and wraps it in Kitty graphics protocol escapes.
func RenderKitty(img *Image, options Options) (string, error) {
	rgba, err := RenderBitmap(img, options)
	if err != nil {
		return "", err
	}

	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, rgba); err != nil {
		return "", fmt.Errorf("encode kitty png: %w", err)
	}

	payload := base64.StdEncoding.EncodeToString(pngBytes.Bytes())
	var out strings.Builder
	for len(payload) > 0 {
		chunk := payload
		if len(chunk) > kittyChunkSize {
			chunk = payload[:kittyChunkSize]
		}
		payload = payload[len(chunk):]
		more := 0
		if len(payload) > 0 {
			more = 1
		}
		fmt.Fprintf(&out, "\x1b_Gf=100,a=T,m=%d;%s\x1b\\", more, chunk)
	}
	return out.String(), nil
}

// RenderSixel rasterizes an image and wraps it in a Sixel device control string.
func RenderSixel(img *Image, options Options) (string, error) {
	rgba, err := RenderBitmap(img, options)
	if err != nil {
		return "", err
	}
	palette, indices, err := sixelPalette(rgba)
	if err != nil {
		return "", err
	}

	bounds := rgba.Bounds()
	var out strings.Builder
	out.WriteString("\x1bPq")
	fmt.Fprintf(&out, "\"1;1;%d;%d", bounds.Dx(), bounds.Dy())
	for i, c := range palette {
		fmt.Fprintf(&out, "#%d;2;%d;%d;%d", i, sixelColor(c.R), sixelColor(c.G), sixelColor(c.B))
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 6 {
		if y > bounds.Min.Y {
			out.WriteByte('-')
		}
		for colorIndex := range palette {
			out.WriteByte('#')
			out.WriteString(fmt.Sprint(colorIndex))
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				var bits byte
				for row := 0; row < 6 && y+row < bounds.Max.Y; row++ {
					if indices[(y+row-bounds.Min.Y)*bounds.Dx()+x-bounds.Min.X] == colorIndex {
						bits |= 1 << row
					}
				}
				out.WriteByte(0x3f + bits)
			}
			out.WriteByte('$')
		}
	}
	out.WriteString("\x1b\\")
	return out.String(), nil
}

// Glyph returns the bitmap rows for a cell, including XBin 512-character font selection.
func (font *Font) Glyph(cell Cell) []byte {
	if font == nil || font.Height <= 0 {
		return nil
	}
	index := int(cell.Char)
	if font.Extended && cell.Attr&0x08 != 0 {
		index += 256
	}
	offset := index * font.Height
	if offset < 0 || offset+font.Height > len(font.Glyphs) {
		return nil
	}
	return font.Glyphs[offset : offset+font.Height]
}

func bitmapCellColors(img *Image, cell Cell, options Options) (color.RGBA, color.RGBA, bool) {
	fgIndex, bgIndex := attrColors(cell.Attr, img.HighBackground)
	fg := rgbaColor(img.Palette[fgIndex], 0xff)
	if cell.FG.Set {
		fg = rgbaColor(overrideColor(cell.FG), 0xff)
	}
	if cell.BG.Set {
		return fg, rgbaColor(overrideColor(cell.BG), 0xff), true
	}
	if bgIndex == 0 && !options.Opaque {
		return fg, color.RGBA{}, false
	}
	return fg, rgbaColor(img.Palette[bgIndex], 0xff), true
}

func attrColors(attr byte, highBackground bool) (int, int) {
	foreground := int(attr & 0x0f)
	background := int((attr >> 4) & 0x07)
	if highBackground {
		background = int((attr >> 4) & 0x0f)
	}
	return foreground, background
}

func overrideColor(override color_modes.Override) color_modes.Color {
	if override.Indexed {
		return color_modes.XTerm256Color(override.Index)
	}
	return override.Color
}

func rgbaColor(c color_modes.Color, alpha byte) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: alpha}
}

func sixelPalette(rgba *image.RGBA) ([]color.RGBA, []int, error) {
	bounds := rgba.Bounds()
	indices := make([]int, bounds.Dx()*bounds.Dy())
	palette := []color.RGBA{}
	seen := map[color.RGBA]int{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := rgba.RGBAAt(x, y)
			if c.A == 0 {
				indices[(y-bounds.Min.Y)*bounds.Dx()+x-bounds.Min.X] = -1
				continue
			}
			index, ok := seen[c]
			if !ok {
				if len(palette) == 256 {
					return nil, nil, fmt.Errorf("sixel render supports up to 256 colors")
				}
				index = len(palette)
				seen[c] = index
				palette = append(palette, c)
			}
			indices[(y-bounds.Min.Y)*bounds.Dx()+x-bounds.Min.X] = index
		}
	}
	return palette, indices, nil
}

func sixelColor(v byte) int {
	return int(v) * 100 / 255
}
